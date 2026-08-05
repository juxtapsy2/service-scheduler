package service

import (
    "context"
    "database/sql"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/sirupsen/logrus"

    "juxtapsy2/service-scheduler-backend/internal/models"
    "juxtapsy2/service-scheduler-backend/internal/repository"
)

var (
    ErrNoTechnician = errors.New("no qualified and available technician found")
    ErrNoBay        = errors.New("no available service bay found")
)

type BookingService struct {
    repo *repository.BookingRepository
}

func NewBookingService(repo *repository.BookingRepository) *BookingService {
    return &BookingService{repo: repo}
}

// Book attempts to create an appointment transactionally
func (s *BookingService) Book(ctx context.Context, req *models.BookingRequest) (string, error) {
    // compute end time from service duration
    durMin, err := s.repo.GetServiceDuration(ctx, req.ServiceTypeID)
    if err != nil {
        return "", err
    }
    end := req.DesiredStart.Add(time.Duration(durMin) * time.Minute)

    techs, err := s.repo.FindQualifiedTechnicians(ctx, req.DealershipID, req.ServiceTypeID)
    if err != nil {
        return "", err
    }
    if len(techs) == 0 {
        return "", ErrNoTechnician
    }

    // helper to attempt booking for a single technician
    tryTech := func(tech string) (string, error) {
        tx, err := s.repo.BeginTx(ctx)
        if err != nil {
            return "", err
        }

        // lock technician
        if err := s.repo.LockTechnician(ctx, tx, tech); err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("lock technician failed")
            return "", err
        }

        // re-check overlapping appointments within transaction
        hasOverlap, err := s.repo.TechnicianHasOverlappingAppointments(ctx, tech, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("checking technician overlap failed")
            return "", err
        }
        if hasOverlap {
            tx.Rollback()
            return "", nil
        }

        // check working period (outside tx is acceptable)
        ok, err := s.repo.TechnicianHasWorkingPeriod(ctx, tech, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("checking working period failed")
            return "", err
        }
        if !ok {
            tx.Rollback()
            return "", nil
        }

        // find available bay (we'll lock bay row after selection)
        bayID, err := s.repo.FindAvailableServiceBay(ctx, req.DealershipID, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("finding bay failed")
            return "", err
        }
        if bayID == "" {
            tx.Rollback()
            // no bay available for this technician/time
            return "", nil
        }

        if err := s.repo.LockServiceBay(ctx, tx, bayID); err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("lock bay failed")
            return "", err
        }

        // double-check bay still free within tx
        var overlap int
        err = tx.GetContext(ctx, &overlap, `SELECT 1 FROM appointment a WHERE a.service_bay_id = $1 AND a.start_time < $3 AND a.end_time > $2 LIMIT 1`, bayID, req.DesiredStart, end)
        if err != nil {
            if err == sql.ErrNoRows {
                // no overlap, good
            } else {
                tx.Rollback()
                logrus.WithError(err).Error("checking bay overlap failed")
                return "", err
            }
        } else {
            // row exists -> overlap
            tx.Rollback()
            return "", nil
        }

        // Create appointment
        apptID := uuid.New().String()
        err = s.repo.CreateAppointment(ctx, tx, apptID, req.CustomerID, req.VehicleID, req.DealershipID, req.ServiceTypeID, tech, bayID, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("create appointment failed")
            return "", err
        }

        if err := tx.Commit(); err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("commit failed")
            return "", err
        }

        return apptID, nil
    }

    // If client requested a particular technician try that first (if qualified)
    if req.PreferredTechnicianID != "" {
        // check requested technician is in the qualified list
        for _, t := range techs {
            if t == req.PreferredTechnicianID {
                id, err := tryTech(req.PreferredTechnicianID)
                if err != nil {
                    return "", err
                }
                if id != "" {
                    return id, nil
                }
                // requested tech not available; fall back to others
                break
            }
        }
    }

    // naive loop: try each technician and attempt transactional booking with locks
    for _, tech := range techs {
        id, err := tryTech(tech)
        if err != nil {
            return "", err
        }
        if id != "" {
            return id, nil
        }
    }

    return "", ErrNoTechnician
}

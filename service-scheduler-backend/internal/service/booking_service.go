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

    // naive loop: try each technician and attempt transactional booking with locks
    for _, tech := range techs {
        tx, err := s.repo.BeginTx(ctx)
        if err != nil {
            return "", err
        }

        // lock technician and bays
        if err := s.repo.LockTechnician(ctx, tx, tech); err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("lock technician failed")
            continue
        }

        // re-check overlapping appointments within transaction
        hasOverlap, err := s.repo.TechnicianHasOverlappingAppointments(ctx, tech, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("checking technician overlap failed")
            continue
        }
        if hasOverlap {
            tx.Rollback()
            continue
        }

        // check working period (outside tx is acceptable)
        ok, err := s.repo.TechnicianHasWorkingPeriod(ctx, tech, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("checking working period failed")
            continue
        }
        if !ok {
            tx.Rollback()
            continue
        }

        // find available bay (we'll lock bay row after selection)
        bayID, err := s.repo.FindAvailableServiceBay(ctx, req.DealershipID, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("finding bay failed")
            continue
        }
        if bayID == "" {
            tx.Rollback()
            // no bay available for this technician/time; try next technician
            continue
        }

        if err := s.repo.LockServiceBay(ctx, tx, bayID); err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("lock bay failed")
            continue
        }

        // double-check bay still free within tx
        // (since FindAvailableServiceBay used snapshot outside locks)
        // check overlapping appointments for bay
        var overlap int
        err = tx.GetContext(ctx, &overlap, `SELECT 1 FROM appointment a WHERE a.service_bay_id = $1 AND a.start_time < $3 AND a.end_time > $2 LIMIT 1`, bayID, req.DesiredStart, end)
        if err != nil {
            if err == sql.ErrNoRows {
                // no overlap, good
            } else {
                tx.Rollback()
                logrus.WithError(err).Error("checking bay overlap failed")
                continue
            }
        } else {
            // row exists -> overlap
            tx.Rollback()
            continue
        }

        // Create appointment
        apptID := uuid.New().String()
        err = s.repo.CreateAppointment(ctx, tx, apptID, req.CustomerID, req.VehicleID, req.DealershipID, req.ServiceTypeID, tech, bayID, req.DesiredStart, end)
        if err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("create appointment failed")
            continue
        }

        if err := tx.Commit(); err != nil {
            tx.Rollback()
            logrus.WithError(err).Error("commit failed")
            continue
        }

        return apptID, nil
    }

    return "", ErrNoTechnician
}

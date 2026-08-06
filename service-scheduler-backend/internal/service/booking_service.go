package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"juxtapsy2/service-scheduler-backend/internal/models"
	"juxtapsy2/service-scheduler-backend/internal/notify"
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
	// compute end time from service duration. Support ad-hoc "Other" services via DurationMinutes.
	var durMin int
	if req.ServiceTypeID != "" {
		var err error
		durMin, err = s.repo.GetServiceDuration(ctx, req.ServiceTypeID)
		if err != nil {
			return "", err
		}
	} else {
		durMin = req.DurationMinutes
		if durMin <= 0 {
			return "", errors.New("missing duration for Other service type")
		}
	}
	end := req.DesiredStart.Add(time.Duration(durMin) * time.Minute)

	logrus.WithFields(logrus.Fields{
		"dealership_id":        req.DealershipID,
		"service_type_id":      req.ServiceTypeID,
		"duration_minutes":     durMin,
		"desired_start":        req.DesiredStart.Format(time.RFC3339),
		"desired_end":          end.Format(time.RFC3339),
		"preferred_technician": req.PreferredTechnicianID,
	}).Info("booking request received")

	techs, err := s.repo.FindQualifiedTechnicians(ctx, req.DealershipID, req.ServiceTypeID)
	if err != nil {
		logrus.WithError(err).Error("failed to fetch qualified technicians")
		return "", err
	}
	logrus.WithField("count", len(techs)).Debug("qualified technicians found")
	if len(techs) == 0 {
		logrus.WithFields(logrus.Fields{"dealership": req.DealershipID, "service_type": req.ServiceTypeID}).Warn("no qualified technicians")
		return "", ErrNoTechnician
	}

	// helper to attempt booking for a single technician
	tryTech := func(tech string) (string, error) {
		logrus.WithField("technician", tech).Debug("attempting technician")
		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			logrus.WithError(err).Error("begin tx failed")
			return "", err
		}

		// lock technician
		if err := s.repo.LockTechnician(ctx, tx, tech); err != nil {
			tx.Rollback()
			logrus.WithError(err).WithField("technician", tech).Error("lock technician failed")
			return "", err
		}

		// re-check overlapping appointments within transaction
		hasOverlap, err := s.repo.TechnicianHasOverlappingAppointments(ctx, tech, req.DesiredStart, end)
		if err != nil {
			tx.Rollback()
			logrus.WithError(err).WithField("technician", tech).Error("checking technician overlap failed")
			return "", err
		}
		if hasOverlap {
			tx.Rollback()
			logrus.WithField("technician", tech).Info("technician has overlapping appointment")
			return "", nil
		}

		// check working period (outside tx is acceptable)
		ok, err := s.repo.TechnicianHasWorkingPeriod(ctx, tech, req.DesiredStart, end)
		if err != nil {
			tx.Rollback()
			logrus.WithError(err).WithField("technician", tech).Error("checking working period failed")
			return "", err
		}
		if !ok {
			tx.Rollback()
			logrus.WithField("technician", tech).Info("technician not working during requested time")
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
			logrus.WithField("technician", tech).Info("no available bay for requested time")
			// no bay available for this technician/time
			return "", nil
		}
		logrus.WithFields(logrus.Fields{"technician": tech, "bay": bayID}).Debug("selected bay")

		if err := s.repo.LockServiceBay(ctx, tx, bayID); err != nil {
			tx.Rollback()
			logrus.WithError(err).WithField("bay", bayID).Error("lock bay failed")
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
				logrus.WithError(err).WithField("bay", bayID).Error("checking bay overlap failed")
				return "", err
			}
		} else {
			// row exists -> overlap
			tx.Rollback()
			logrus.WithFields(logrus.Fields{"bay": bayID, "technician": tech}).Info("bay has overlapping appointment")
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

		logrus.WithFields(logrus.Fields{"appointment_id": apptID, "technician": tech, "bay": bayID}).Info("appointment created")
		// broadcast availability change for dealership so connected clients can refresh
		go func() {
			notifyMsg := map[string]interface{}{
				"type":           "appointment.created",
				"appointment_id": apptID,
				"technician_id":  tech,
				"service_bay_id": bayID,
				"start":          req.DesiredStart,
				"end":            end,
				"dealership_id":  req.DealershipID,
			}
			// best-effort broadcast
			notify.DefaultHub.Broadcast(req.DealershipID, notifyMsg)
		}()
		return apptID, nil
	}

	// If client requested a particular technician try that first (if qualified)
	if req.PreferredTechnicianID != "" {
		// check requested technician is in the qualified list
		for _, t := range techs {
			if t == req.PreferredTechnicianID {
				logrus.WithField("preferred_technician", req.PreferredTechnicianID).Info("trying preferred technician first")
				id, err := tryTech(req.PreferredTechnicianID)
				if err != nil {
					return "", err
				}
				if id != "" {
					return id, nil
				}
				// requested tech not available; fall back to others
				logrus.WithField("preferred_technician", req.PreferredTechnicianID).Info("preferred technician not available, falling back")
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

	logrus.WithFields(logrus.Fields{"dealership": req.DealershipID, "service_type": req.ServiceTypeID}).Warn("no technician/bay available after attempts")
	return "", ErrNoTechnician
}

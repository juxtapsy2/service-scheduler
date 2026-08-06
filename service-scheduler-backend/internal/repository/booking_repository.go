package repository

import (
    "context"
    "database/sql"
    "time"

    "github.com/jmoiron/sqlx"
    "github.com/sirupsen/logrus"

    "juxtapsy2/service-scheduler-backend/internal/config"
)

// BookingRepository provides DB operations needed by booking service
type BookingRepository struct {
    db *sqlx.DB
}

func NewBookingRepository(db *sqlx.DB) *BookingRepository {
    return &BookingRepository{db: db}
}

// FindQualifiedTechnicians returns technician IDs qualified for a service at a dealership
func (r *BookingRepository) FindQualifiedTechnicians(ctx context.Context, dealershipID, serviceTypeID string) ([]string, error) {
    var ids []string
    if serviceTypeID == "" {
        // no qualification required: return all active technicians for dealership
        query := `SELECT id FROM technician WHERE dealership_id = $1 AND active = true ORDER BY last_name, first_name`
        err := r.db.SelectContext(ctx, &ids, query, dealershipID)
        return ids, err
    }

    query := `
SELECT t.id
FROM technician t
JOIN technician_qualification q ON q.technician_id = t.id
WHERE t.dealership_id = $1
  AND q.service_type_id = $2
  AND t.active = true
`
    err := r.db.SelectContext(ctx, &ids, query, dealershipID, serviceTypeID)
    return ids, err
}

// TechnicianWorkingPeriod checks if technician has working schedule covering the interval (uses day_of_week/time)
func (r *BookingRepository) TechnicianHasWorkingPeriod(ctx context.Context, technicianID string, start, end time.Time) (bool, error) {
    // Interpret start/end in server local time when comparing to schedule TIMES stored without timezone.
    // This makes the schedule check resilient to client UTC conversion from datetime-local inputs.
    localStart := start
    localEnd := end

    // Check each day segment; for simplicity assume appointment is within single day
    query := `
SELECT 1 FROM technician_schedule s
WHERE s.technician_id = $1
  AND s.day_of_week = $2
  AND s.schedule_type = 'WORKING'
  AND s.start_time <= $3::time
  AND s.end_time >= $4::time
LIMIT 1
`
    dow := int(localStart.Weekday())
    if dow == 0 {
        dow = 7 // Go: Sunday=0, DB expects 7
    }
    // debug log
    logrus.WithFields(logrus.Fields{
        "technician_id": technicianID,
        "dow": dow,
        "start_time": localStart.Format(time.RFC3339),
        "end_time": localEnd.Format(time.RFC3339),
    }).Debug("checking technician working period")

    var dummy int
    err := r.db.GetContext(ctx, &dummy, query, technicianID, dow, localStart.Format("15:04:05"), localEnd.Format("15:04:05"))
    if err == nil {
        logrus.WithField("technician_id", technicianID).Debug("found explicit schedule row")
        return true, nil
    }
    if err != sql.ErrNoRows {
        return false, err
    }

    // No WORKING schedule covers the requested interval. Check if the technician has any schedule rows for that day.
    var any int
    err2 := r.db.GetContext(ctx, &any, "SELECT 1 FROM technician_schedule WHERE technician_id = $1 AND day_of_week = $2 LIMIT 1", technicianID, dow)
    if err2 == nil {
        // technician has schedule rows on that day but none of them are a WORKING segment covering the interval
        logrus.WithField("technician_id", technicianID).Debug("technician has schedule rows but no working coverage for interval")
        return false, nil
    }
    if err2 != sql.ErrNoRows {
        return false, err2
    }

    // No explicit schedule rows exist for this technician on that day. Apply a simple default working window from config.
    ls := localStart.Format("15:04:05")
    le := localEnd.Format("15:04:05")
    logrus.WithFields(logrus.Fields{"technician_id": technicianID, "ls": ls, "le": le, "default_start": config.DefaultWorkingStart, "default_end": config.DefaultWorkingEnd}).Debug("no explicit schedule, checking default window")
    if ls >= config.DefaultWorkingStart && le <= config.DefaultWorkingEnd {
        logrus.WithField("technician_id", technicianID).Debug("allowed by default working window")
        return true, nil
    }

    return false, nil
}

// TechnicianHasOverlappingAppointments checks for overlapping appointments
func (r *BookingRepository) TechnicianHasOverlappingAppointments(ctx context.Context, technicianID string, start, end time.Time) (bool, error) {
    query := `
SELECT 1 FROM appointment a
WHERE a.technician_id = $1
  AND a.start_time < $3
  AND a.end_time > $2
LIMIT 1
`
    var dummy int
    err := r.db.GetContext(ctx, &dummy, query, technicianID, start, end)
    if err == sql.ErrNoRows {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}

// FindAvailableServiceBay finds an available bay for the dealership during the interval
func (r *BookingRepository) FindAvailableServiceBay(ctx context.Context, dealershipID string, start, end time.Time) (string, error) {
    // find bay ids active and not having overlapping appointments
    query := `
SELECT b.id
FROM service_bay b
WHERE b.dealership_id = $1
  AND b.active = true
  AND NOT EXISTS (
    SELECT 1 FROM appointment a
    WHERE a.service_bay_id = b.id
      AND a.start_time < $3
      AND a.end_time > $2
  )
LIMIT 1
`
    var id string
    err := r.db.GetContext(ctx, &id, query, dealershipID, start, end)
    if err == sql.ErrNoRows {
        return "", nil
    }
    if err != nil {
        return "", err
    }
    return id, nil
}

// CreateAppointment inserts the appointment within a transaction
func (r *BookingRepository) CreateAppointment(ctx context.Context, tx *sqlx.Tx, apptID, customerID, vehicleID, dealershipID, serviceTypeID, technicianID, serviceBayID string, start, end time.Time) error {
    query := `
INSERT INTO appointment (
    id, customer_id, vehicle_id, dealership_id, service_type_id, technician_id, service_bay_id, start_time, end_time, status
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'CONFIRMED')
`
    _, err := tx.ExecContext(ctx, query, apptID, customerID, vehicleID, dealershipID, serviceTypeID, technicianID, serviceBayID, start, end)
    return err
}

// LockTechnician obtains a row-level lock on the technician row to avoid races
func (r *BookingRepository) LockTechnician(ctx context.Context, tx *sqlx.Tx, technicianID string) error {
    // Lock the technician row itself (simpler and reliable)
    query := `SELECT id FROM technician WHERE id = $1 FOR UPDATE`
    var id string
    err := tx.QueryRowContext(ctx, query, technicianID).Scan(&id)
    if err == sql.ErrNoRows {
        return nil // technician not found; let higher-level checks handle this
    }
    return err
}

// LockServiceBay obtains a lock on service_bay row
func (r *BookingRepository) LockServiceBay(ctx context.Context, tx *sqlx.Tx, serviceBayID string) error {
    query := `SELECT id FROM service_bay WHERE id = $1 FOR UPDATE`
    var id string
    err := tx.QueryRowContext(ctx, query, serviceBayID).Scan(&id)
    if err == sql.ErrNoRows {
        return nil
    }
    return err
}

// BeginTx starts a transaction
func (r *BookingRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
    return r.db.BeginTxx(ctx, nil)
}

// Health check helper for service_type duration
func (r *BookingRepository) GetServiceDuration(ctx context.Context, serviceTypeID string) (int, error) {
    var d int
    err := r.db.GetContext(ctx, &d, "SELECT duration_minutes FROM service_type WHERE id = $1", serviceTypeID)
    return d, err
}

// GetServiceTypes returns all service types
func (r *BookingRepository) GetServiceTypes(ctx context.Context) ([]map[string]interface{}, error) {
    var out []map[string]interface{}
    rows, err := r.db.QueryxContext(ctx, "SELECT id, name, duration_minutes FROM service_type ORDER BY name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    for rows.Next() {
        m := map[string]interface{}{}
        if err := rows.MapScan(m); err != nil {
            return nil, err
        }
        // normalize byte slices
        for k, v := range m {
            if b, ok := v.([]byte); ok {
                m[k] = string(b)
            }
        }
        out = append(out, m)
    }
    return out, nil
}

// GetServiceTypeIDByName returns the id for a service_type name (or empty string if not found)
func (r *BookingRepository) GetServiceTypeIDByName(ctx context.Context, name string) (string, error) {
    var id string
    err := r.db.GetContext(ctx, &id, "SELECT id FROM service_type WHERE name = $1 LIMIT 1", name)
    if err == sql.ErrNoRows {
        return "", nil
    }
    if err != nil {
        return "", err
    }
    return id, nil
}

// GetDealerships returns id and name for all dealerships
func (r *BookingRepository) GetDealerships(ctx context.Context) ([]map[string]interface{}, error) {
    var out []map[string]interface{}
    rows, err := r.db.QueryxContext(ctx, "SELECT id, name FROM dealership ORDER BY name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    for rows.Next() {
        m := map[string]interface{}{}
        if err := rows.MapScan(m); err != nil {
            return nil, err
        }
        for k, v := range m {
            if b, ok := v.([]byte); ok {
                m[k] = string(b)
            }
        }
        out = append(out, m)
    }
    return out, nil
}

// GetTechniciansByDealership returns list of technicians (id, first_name, last_name) for a dealership
func (r *BookingRepository) GetTechniciansByDealership(ctx context.Context, dealershipID string) ([]map[string]interface{}, error) {
    var out []map[string]interface{}
    rows, err := r.db.QueryxContext(ctx, "SELECT id, first_name, last_name FROM technician WHERE dealership_id = $1 AND active = true ORDER BY last_name, first_name", dealershipID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    for rows.Next() {
        m := map[string]interface{}{}
        if err := rows.MapScan(m); err != nil {
            return nil, err
        }
        for k, v := range m {
            if b, ok := v.([]byte); ok {
                m[k] = string(b)
            }
        }
        out = append(out, m)
    }
    return out, nil
}

// GetOrCreateCustomerByEmail returns customer id for email, creating a new customer if not exists
func (r *BookingRepository) GetOrCreateCustomerByEmail(ctx context.Context, email, firstName, lastName string) (string, error) {
    var id string
    err := r.db.GetContext(ctx, &id, "SELECT id FROM customer WHERE email = $1", email)
    if err == nil {
        return id, nil
    }
    if err != sql.ErrNoRows {
        return "", err
    }
    query := `INSERT INTO customer (id, first_name, last_name, email) VALUES (uuid_generate_v4(), $1, $2, $3) RETURNING id`
    err = r.db.GetContext(ctx, &id, query, firstName, lastName, email)
    return id, err
}

// GetOrCreateVehicleByVIN returns vehicle id for vin, creating new vehicle if not exists. Requires customerID.
func (r *BookingRepository) GetOrCreateVehicleByVIN(ctx context.Context, vin, makeStr, model string, year int, customerID string) (string, error) {
    var id string
    err := r.db.GetContext(ctx, &id, "SELECT id FROM vehicle WHERE vin = $1", vin)
    if err == nil {
        return id, nil
    }
    if err != sql.ErrNoRows {
        return "", err
    }
    query := `INSERT INTO vehicle (id, customer_id, vin, make, model, year) VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5) RETURNING id`
    err = r.db.GetContext(ctx, &id, query, customerID, vin, makeStr, model, year)
    return id, err
}

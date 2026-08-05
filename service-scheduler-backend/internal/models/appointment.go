package models

import "time"

type Appointment struct {
    ID           string    `db:"id" json:"id"`
    CustomerID   string    `db:"customer_id" json:"customer_id"`
    VehicleID    string    `db:"vehicle_id" json:"vehicle_id"`
    DealershipID string    `db:"dealership_id" json:"dealership_id"`
    ServiceTypeID string   `db:"service_type_id" json:"service_type_id"`
    TechnicianID string    `db:"technician_id" json:"technician_id"`
    ServiceBayID string    `db:"service_bay_id" json:"service_bay_id"`
    StartTime    time.Time `db:"start_time" json:"start_time"`
    EndTime      time.Time `db:"end_time" json:"end_time"`
    Status       string    `db:"status" json:"status"`
    CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

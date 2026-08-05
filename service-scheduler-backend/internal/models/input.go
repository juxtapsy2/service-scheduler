package models

import "time"

// BookingRequest represents incoming booking payload
type BookingRequest struct {
    CustomerID    string    `json:"customer_id" binding:"required,uuid"`
    VehicleID     string    `json:"vehicle_id" binding:"required,uuid"`
    DealershipID  string    `json:"dealership_id" binding:"required,uuid"`
    ServiceTypeID string    `json:"service_type_id" binding:"required,uuid"`
    DesiredStart  time.Time `json:"desired_start" binding:"required"`
}

// BookingResponse minimal response
type BookingResponse struct {
    AppointmentID string `json:"appointment_id"`
    Message       string `json:"message"`
}

package models

import "time"

// BookingRequest represents incoming booking payload
type BookingRequest struct {
    CustomerID    string    `json:"customer_id" binding:"required,uuid"`
    VehicleID     string    `json:"vehicle_id" binding:"required,uuid"`
    DealershipID  string    `json:"dealership_id" binding:"required,uuid"`
    ServiceTypeID string    `json:"service_type_id" binding:"required,uuid"`
    PreferredTechnicianID string `json:"preferred_technician_id" binding:"omitempty,uuid"`
    DesiredStart  time.Time `json:"desired_start" binding:"required"`
}

// QuickBookingRequest is a user-friendly booking payload (no pre-existing ids)
type QuickBookingRequest struct {
    CustomerFirstName string    `json:"customer_first_name" binding:"required"`
    CustomerLastName  string    `json:"customer_last_name" binding:"required"`
    CustomerEmail     string    `json:"customer_email" binding:"required,email"`
    CustomerPhone     string    `json:"customer_phone"`

    VehicleVIN   string `json:"vehicle_vin" binding:"required"`
    VehicleMake  string `json:"vehicle_make" binding:"required"`
    VehicleModel string `json:"vehicle_model" binding:"required"`
    VehicleYear  int    `json:"vehicle_year" binding:"required"`

    DealershipID   string    `json:"dealership_id" binding:"required,uuid"`
    ServiceType    string    `json:"service_type" binding:"required"` // name or id
    PreferredTechnicianID string `json:"preferred_technician_id" binding:"omitempty,uuid"`
    DesiredStart   time.Time `json:"desired_start" binding:"required"`
}

// BookingResponse minimal response
type BookingResponse struct {
    AppointmentID string `json:"appointment_id"`
    Message       string `json:"message"`
}

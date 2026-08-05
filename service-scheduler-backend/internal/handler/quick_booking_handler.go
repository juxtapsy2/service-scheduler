package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "juxtapsy2/service-scheduler-backend/internal/models"
    "juxtapsy2/service-scheduler-backend/internal/repository"
    "juxtapsy2/service-scheduler-backend/internal/service"
)

// QuickBookingHandler handles friendly quick booking flow
type QuickBookingHandler struct {
    repo *repository.BookingRepository
    svc  *service.BookingService
}

func NewQuickBookingHandler(repo *repository.BookingRepository, svc *service.BookingService) *QuickBookingHandler {
    return &QuickBookingHandler{repo: repo, svc: svc}
}

func (h *QuickBookingHandler) RegisterRoutes(r *gin.Engine) {
    r.POST("/api/quick-booking", h.QuickBook)
}

func (h *QuickBookingHandler) QuickBook(c *gin.Context) {
    var req models.QuickBookingRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // create or find customer
    customerID, err := h.repo.GetOrCreateCustomerByEmail(c.Request.Context(), req.CustomerEmail, req.CustomerFirstName, req.CustomerLastName)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    vehicleID, err := h.repo.GetOrCreateVehicleByVIN(c.Request.Context(), req.VehicleVIN, req.VehicleMake, req.VehicleModel, req.VehicleYear, customerID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // resolve service type by name
    var serviceTypeID string
    rows, err := h.repo.GetServiceTypes(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    for _, rmap := range rows {
        if name, ok := rmap["name"].(string); ok && name == req.ServiceType {
            if id, ok := rmap["id"].(string); ok {
                serviceTypeID = id
                break
            }
        }
    }
    if serviceTypeID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "unknown service type"})
        return
    }

    // build BookingRequest
    br := models.BookingRequest{
        CustomerID:    customerID,
        VehicleID:     vehicleID,
        DealershipID:  req.DealershipID,
        ServiceTypeID: serviceTypeID,
        PreferredTechnicianID: req.PreferredTechnicianID,
        DesiredStart:  req.DesiredStart,
    }

    apptID, err := h.svc.Book(c.Request.Context(), &br)
    if err != nil {
        switch err {
        case service.ErrNoTechnician:
            c.JSON(http.StatusConflict, gin.H{"error": "No available technician or bay for requested time"})
        default:
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        }
        return
    }

    c.JSON(http.StatusCreated, models.BookingResponse{AppointmentID: apptID, Message: "Booked"})
}

package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "juxtapsy2/service-scheduler-backend/internal/models"
    "juxtapsy2/service-scheduler-backend/internal/service"
)

type BookingHandler struct {
    svc *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
    return &BookingHandler{svc: svc}
}

func (h *BookingHandler) RegisterRoutes(r *gin.Engine) {
    r.POST("/api/bookings", h.CreateBooking)
}

func (h *BookingHandler) CreateBooking(c *gin.Context) {
    var req models.BookingRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    apptID, err := h.svc.Book(c.Request.Context(), &req)
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

package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"juxtapsy2/service-scheduler-backend/internal/repository"
)

// AvailabilityHandler provides lightweight availability checks for UI
type AvailabilityHandler struct {
	repo *repository.BookingRepository
}

func NewAvailabilityHandler(repo *repository.BookingRepository) *AvailabilityHandler {
	return &AvailabilityHandler{repo: repo}
}

func (h *AvailabilityHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/availability", h.Check)
}

// availability request from frontend
type availabilityRequest struct {
	DealershipID          string    `json:"dealership_id" binding:"required,uuid"`
	PreferredTechnicianID string    `json:"preferred_technician_id" binding:"omitempty,uuid"`
	ServiceType           string    `json:"service_type" binding:"required"`
	OtherDurationMinutes  int       `json:"other_duration_minutes,omitempty"`
	DesiredStart          time.Time `json:"desired_start" binding:"required"`
}

type availabilityResponse struct {
	TechnicianAvailable bool      `json:"technician_available"`
	TechnicianReason    string    `json:"technician_reason,omitempty"`
	BayAvailable        bool      `json:"bay_available"`
	BayID               string    `json:"bay_id,omitempty"`
	DesiredEnd          time.Time `json:"desired_end"`
}

func (h *AvailabilityHandler) Check(c *gin.Context) {
	var req availabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// determine duration
	var durationMinutes int
	if req.ServiceType == "__other__" {
		durationMinutes = req.OtherDurationMinutes
		if durationMinutes <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "other_duration_minutes must be provided and > 0 for Other"})
			return
		}
	} else {
		// resolve service type name -> id
		stid, err := h.repo.GetServiceTypeIDByName(ctx, req.ServiceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if stid == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown service type"})
			return
		}
		d, err := h.repo.GetServiceDuration(ctx, stid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		durationMinutes = d
	}

	start := req.DesiredStart
	end := start.Add(time.Duration(durationMinutes) * time.Minute)

	resp := availabilityResponse{DesiredEnd: end}

	// check bay availability (lightweight, non-locking)
	bayID, err := h.repo.FindAvailableServiceBay(ctx, req.DealershipID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bayID != "" {
		resp.BayAvailable = true
		resp.BayID = bayID
	} else {
		resp.BayAvailable = false
	}

	// check preferred technician if provided
	if req.PreferredTechnicianID != "" {
		tech := req.PreferredTechnicianID

		// if service type is not Other, ensure technician is qualified
		if req.ServiceType != "__other__" {
			stid, _ := h.repo.GetServiceTypeIDByName(ctx, req.ServiceType)
			// if stid empty it's already handled above
			ids, err := h.repo.FindQualifiedTechnicians(ctx, req.DealershipID, stid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			found := false
			for _, id := range ids {
				if id == tech {
					found = true
					break
				}
			}
			if !found {
				resp.TechnicianAvailable = false
				resp.TechnicianReason = "technician not qualified for selected service"
				logrus.WithFields(logrus.Fields{"technician": tech, "dealership": req.DealershipID, "service_type": req.ServiceType}).Info("technician not qualified")
				c.JSON(http.StatusOK, resp)
				return
			}
		}

		// check working period
		ok, err := h.repo.TechnicianHasWorkingPeriod(ctx, tech, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !ok {
			resp.TechnicianAvailable = false
			resp.TechnicianReason = "technician not working during selected interval"
			c.JSON(http.StatusOK, resp)
			return
		}

		// check overlapping appointments
		overlap, err := h.repo.TechnicianHasOverlappingAppointments(ctx, tech, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if overlap {
			resp.TechnicianAvailable = false
			resp.TechnicianReason = "technician has overlapping appointment"
			c.JSON(http.StatusOK, resp)
			return
		}

		resp.TechnicianAvailable = true
		resp.TechnicianReason = "available"
	}

	c.JSON(http.StatusOK, resp)
}

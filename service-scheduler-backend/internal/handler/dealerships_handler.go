package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"juxtapsy2/service-scheduler-backend/internal/repository"
)

// DealershipsHandler exposes dealerships list
type DealershipsHandler struct {
	repo *repository.BookingRepository
}

func NewDealershipsHandler(repo *repository.BookingRepository) *DealershipsHandler {
	return &DealershipsHandler{repo: repo}
}

func (h *DealershipsHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/dealerships", h.List)
}

func (h *DealershipsHandler) List(c *gin.Context) {
	rows, err := h.repo.GetDealerships(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

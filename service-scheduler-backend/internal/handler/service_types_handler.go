package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"juxtapsy2/service-scheduler-backend/internal/repository"
)

type ServiceTypesHandler struct {
	repo *repository.BookingRepository
}

func NewServiceTypesHandler(repo *repository.BookingRepository) *ServiceTypesHandler {
	return &ServiceTypesHandler{repo: repo}
}

func (h *ServiceTypesHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/service-types", h.List)
}

func (h *ServiceTypesHandler) List(c *gin.Context) {
	types, err := h.repo.GetServiceTypes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, types)
}

package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "juxtapsy2/service-scheduler-backend/internal/repository"
)

// TechniciansHandler exposes simple read endpoints for technicians
type TechniciansHandler struct {
    repo *repository.BookingRepository
}

func NewTechniciansHandler(repo *repository.BookingRepository) *TechniciansHandler {
    return &TechniciansHandler{repo: repo}
}

func (h *TechniciansHandler) RegisterRoutes(r *gin.Engine) {
    r.GET("/api/technicians", h.List)
}

func (h *TechniciansHandler) List(c *gin.Context) {
    dealer := c.Query("dealership_id")
    if dealer == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing dealership_id"})
        return
    }

    // load technicians for dealership first (narrow by dealership)
    rows, err := h.repo.GetTechniciansByDealership(c.Request.Context(), dealer)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // optional service_type name to filter qualified technicians
    svcName := c.Query("service_type")
    if svcName == "" {
        c.JSON(http.StatusOK, rows)
        return
    }

    // resolve service type name to id
    stid, err := h.repo.GetServiceTypeIDByName(c.Request.Context(), svcName)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    if stid == "" {
        // unknown service type -> return empty list
        c.JSON(http.StatusOK, []map[string]interface{}{})
        return
    }

    // find qualified technician ids for dealership + service type
    ids, err := h.repo.FindQualifiedTechnicians(c.Request.Context(), dealer, stid)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    idSet := map[string]struct{}{}
    for _, id := range ids {
        idSet[id] = struct{}{}
    }
    var out []map[string]interface{}
    for _, rmap := range rows {
        if idv, ok := rmap["id"].(string); ok {
            if _, found := idSet[idv]; found {
                out = append(out, rmap)
            }
        } else if b, ok := rmap["id"].([]byte); ok {
            sid := string(b)
            if _, found := idSet[sid]; found {
                out = append(out, rmap)
            }
        }
    }
    c.JSON(http.StatusOK, out)
}

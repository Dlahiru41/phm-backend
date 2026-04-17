package handlers

import (
	"net/http"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
)

type MOHDashboardHandler struct {
	DashboardStore *store.MOHDashboardStore
}

// TotalChildren returns the total number of children in the system
func (h *MOHDashboardHandler) TotalChildren(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	total, err := h.DashboardStore.TotalChildren(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch total children")
		return
	}

	response.OK(c, map[string]interface{}{
		"totalChildren": total,
	})
}

// ChildrenDistribution returns children grouped by GN division
func (h *MOHDashboardHandler) ChildrenDistribution(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	distribution, err := h.DashboardStore.ChildrenDistribution(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch distribution")
		return
	}

	response.OK(c, distribution)
}

// VaccinationCoverage returns overall vaccination coverage percentage
func (h *MOHDashboardHandler) VaccinationCoverage(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	total, vaccinated, coverage, err := h.DashboardStore.VaccinationCoverage(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch coverage")
		return
	}

	response.OK(c, map[string]interface{}{
		"totalChildren":      total,
		"vaccinatedChildren": vaccinated,
		"coverage":           coverage,
	})
}

// MissedVaccinations returns count of missed vaccinations
func (h *MOHDashboardHandler) MissedVaccinations(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	missed, err := h.DashboardStore.MissedVaccinations(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch missed vaccinations")
		return
	}

	response.OK(c, map[string]interface{}{
		"missedVaccinations": missed,
	})
}

// PHMPerformance returns performance summary for each PHM
func (h *MOHDashboardHandler) PHMPerformance(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	performance, err := h.DashboardStore.PHMPerformanceSummary(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch PHM performance")
		return
	}

	response.OK(c, performance)
}

// RecentChildren returns the latest registered children
func (h *MOHDashboardHandler) RecentChildren(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	recent, err := h.DashboardStore.RecentChildren(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch recent children")
		return
	}

	response.OK(c, recent)
}

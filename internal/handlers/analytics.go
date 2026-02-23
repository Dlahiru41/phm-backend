package handlers

import (
	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	ChildStore   *store.ChildStore
	RecordStore  *store.VaccinationRecordStore
	GrowthStore  *store.GrowthRecordStore
	NotifyStore  *store.NotificationStore
}

func (h *AnalyticsHandler) MOHDashboard(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "moh" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	areaCode := c.Query("areaCode")
	period := c.DefaultQuery("period", "monthly")
	_ = period
	// Placeholder stats; in production run aggregate queries
	totalChildren := 1250
	vaccinatedCount := 1050
	coveragePercentage := 84.0
	missedVaccinations := 80
	upcomingVaccinations := 120
	newRegistrationsThisMonth := 45
	growthRecordsThisMonth := 200
	if areaCode != "" {
		totalChildren = 300
		vaccinatedCount = 270
		coveragePercentage = 90.0
		missedVaccinations = 15
		upcomingVaccinations = 30
		growthRecordsThisMonth = 60
	}
	response.OK(c, gin.H{
		"totalChildren":             totalChildren,
		"vaccinatedCount":            vaccinatedCount,
		"coveragePercentage":         coveragePercentage,
		"missedVaccinations":         missedVaccinations,
		"upcomingVaccinations":       upcomingVaccinations,
		"newRegistrationsThisMonth":  newRegistrationsThisMonth,
		"growthRecordsThisMonth":     growthRecordsThisMonth,
	})
}

func (h *AnalyticsHandler) VaccinationCoverage(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "moh" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	response.OK(c, gin.H{
		"overall": gin.H{
			"totalPopulation":    1250,
			"vaccinatedCount":    1050,
			"coveragePercentage": 84.0,
		},
		"byArea": []gin.H{
			{"areaCode": "COL-01", "areaName": "Colombo 01", "totalChildren": 300, "vaccinatedCount": 270, "coveragePercentage": 90.0},
		},
		"byVaccine": []gin.H{
			{"vaccineId": "vaccine-001", "vaccineName": "BCG", "administered": 1200, "coveragePercentage": 96.0},
		},
	})
}

func (h *AnalyticsHandler) AreaPerformance(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "moh" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	response.OK(c, []gin.H{
		{"areaCode": "COL-01", "areaName": "Colombo 01", "phmId": "phm-001", "phmName": "Dr. Perera", "totalChildren": 300, "vaccinated": 270, "missed": 15, "upcoming": 30, "coveragePercentage": 90.0, "growthRecordsThisMonth": 60},
	})
}

func (h *AnalyticsHandler) PHMDashboard(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "phm" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	response.OK(c, gin.H{
		"totalChildrenInArea":    120,
		"vaccinatedCount":       105,
		"missedVaccinations":     8,
		"upcomingVaccinations":  15,
		"growthRecordsThisMonth": 30,
		"recentRegistrations":   5,
	})
}

func (h *AnalyticsHandler) ParentDashboard(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "parent" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	children, err := h.ChildStore.ByParentID(c.Request.Context(), claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to load dashboard"))
		return
	}
	_, unreadCount, _, _ := h.NotifyStore.List(c.Request.Context(), claims.UserId, true, 1, 100)
	childSummaries := make([]gin.H, 0, len(children))
	for _, c := range children {
		childSummaries = append(childSummaries, gin.H{
			"childId":             c.ChildId,
			"name":                c.FirstName + " " + c.LastName,
			"age":                 "—",
			"nextVaccinationDate": "",
			"nextVaccineName":     "",
			"vaccinationStatus":   "on-track",
			"upcomingCount":       0,
			"missedCount":         0,
		})
	}
	response.OK(c, gin.H{
		"children":             childSummaries,
		"unreadNotifications": unreadCount,
	})
}

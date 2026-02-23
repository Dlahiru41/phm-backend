package handlers

import (
	"strconv"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReportsHandler struct {
	ReportStore *store.ReportStore
}

type GenerateReportRequest struct {
	ReportType  string `json:"reportType" binding:"required,oneof=vaccination_coverage area_performance audit_report growth_analysis monthly_summary"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	District    string `json:"district"`
	DsDivision  string `json:"dsDivision"`
	VaccineId   string `json:"vaccineId"`
	Format      string `json:"format" binding:"required,oneof=pdf excel csv"`
}

func (h *ReportsHandler) Generate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	reportID := "rpt-" + uuid.New().String()[:8]
	filePath := "/reports/" + reportID + "." + req.Format
	filterParams := map[string]interface{}{
		"district":   req.District,
		"dsDivision": req.DsDivision,
		"vaccineId":  req.VaccineId,
	}
	err := h.ReportStore.Create(c.Request.Context(), reportID, req.ReportType, claims.UserId, req.StartDate, req.EndDate, req.Format, filePath, filterParams)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to generate report"))
		return
	}
	report, err := h.ReportStore.GetByID(c.Request.Context(), reportID)
	if err != nil {
		response.Created(c, gin.H{
			"reportId":      reportID,
			"reportType":    req.ReportType,
			"generatedBy":   claims.UserId,
			"startDate":     req.StartDate,
			"endDate":       req.EndDate,
			"downloadUrl":   "/api/v1/reports/" + reportID + "/download",
			"message":       "Report generated successfully.",
		})
		return
	}
	response.Created(c, gin.H{
		"reportId":      reportID,
		"reportType":    req.ReportType,
		"generatedBy":   report.GeneratedBy,
		"generatedDate": report.GeneratedDate,
		"startDate":     req.StartDate,
		"endDate":       req.EndDate,
		"downloadUrl":   "/api/v1/reports/" + reportID + "/download",
		"message":       "Report generated successfully.",
	})
}

func (h *ReportsHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 20
	}
	list, err := h.ReportStore.List(c.Request.Context(), claims.UserId, c.Query("reportType"), page, limit)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list reports"))
		return
	}
	response.OK(c, list)
}

func (h *ReportsHandler) Download(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	reportID := c.Param("reportId")
	report, err := h.ReportStore.GetByID(c.Request.Context(), reportID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	if report.GeneratedBy != claims.UserId {
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	format := c.DefaultQuery("format", "pdf")
	// In production: stream file from report.FilePath; for now return 200 with a placeholder message
	c.Header("Content-Disposition", "attachment; filename=report-"+reportID+"."+format)
	response.OK(c, gin.H{"message": "Report file would be streamed from " + report.DownloadUrl})
}

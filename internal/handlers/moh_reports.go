package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

type MOHReportsHandler struct {
	ReportStore *store.MOHReportStore
}

type ReportRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	GNDivision string `json:"gnDivision"`
	Role       string `json:"role"`
	Action     string `json:"action"`
}

// VaccinationCoverageReport returns vaccination coverage by GN division
func (h *MOHReportsHandler) VaccinationCoverageReport(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("[VaccinationCoverageReport] Binding error: %v", err)
		response.ValidationError(c, "Invalid query parameters", nil)
		return
	}

	log.Printf("[VaccinationCoverageReport Handler] Processing request from user: %s, startDate: %s, endDate: %s, gnDivision: %s",
		claims.UserId, req.StartDate, req.EndDate, req.GNDivision)

	data, err := h.ReportStore.VaccinationCoverageReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)
	if err != nil {
		log.Printf("[VaccinationCoverageReport Handler] Store error: %v", err)
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	log.Printf("[VaccinationCoverageReport Handler] Successfully generated report with %d records", len(data))
	response.OK(c, map[string]interface{}{
		"reportType": "vaccination_coverage",
		"filters": map[string]interface{}{
			"startDate":  req.StartDate,
			"endDate":    req.EndDate,
			"gnDivision": req.GNDivision,
		},
		"data": data,
	})
}

// MissedVaccinationReport returns missed vaccinations with child details
func (h *MOHReportsHandler) MissedVaccinationReport(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("[MissedVaccinationReport] Binding error: %v", err)
		response.ValidationError(c, "Invalid query parameters", nil)
		return
	}

	log.Printf("[MissedVaccinationReport Handler] Processing request from user: %s, startDate: %s, endDate: %s, gnDivision: %s",
		claims.UserId, req.StartDate, req.EndDate, req.GNDivision)

	data, err := h.ReportStore.MissedVaccinationReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)
	if err != nil {
		log.Printf("[MissedVaccinationReport Handler] Store error: %v", err)
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	log.Printf("[MissedVaccinationReport Handler] Successfully generated report with %d records", len(data))
	response.OK(c, map[string]interface{}{
		"reportType": "missed_vaccination",
		"filters": map[string]interface{}{
			"startDate":  req.StartDate,
			"endDate":    req.EndDate,
			"gnDivision": req.GNDivision,
		},
		"data": data,
	})
}

// PHMPerformanceReport returns PHM performance metrics
func (h *MOHReportsHandler) PHMPerformanceReport(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("[PHMPerformanceReport] Binding error: %v", err)
		response.ValidationError(c, "Invalid query parameters", nil)
		return
	}

	log.Printf("[PHMPerformanceReport Handler] Processing request from user: %s, startDate: %s, endDate: %s, gnDivision: %s",
		claims.UserId, req.StartDate, req.EndDate, req.GNDivision)

	data, err := h.ReportStore.PHMPerformanceReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)
	if err != nil {
		log.Printf("[PHMPerformanceReport Handler] Store error: %v", err)
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	log.Printf("[PHMPerformanceReport Handler] Successfully generated report with %d records", len(data))
	response.OK(c, map[string]interface{}{
		"reportType": "phm_performance",
		"filters": map[string]interface{}{
			"startDate":  req.StartDate,
			"endDate":    req.EndDate,
			"gnDivision": req.GNDivision,
		},
		"data": data,
	})
}

// AuditReport returns audit logs
func (h *MOHReportsHandler) AuditReport(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("[AuditReport] Binding error: %v", err)
		response.ValidationError(c, "Invalid query parameters", nil)
		return
	}

	log.Printf("[AuditReport Handler] Processing request from user: %s, startDate: %s, endDate: %s, role: %s, action: %s",
		claims.UserId, req.StartDate, req.EndDate, req.Role, req.Action)

	data, err := h.ReportStore.AuditReport(c.Request.Context(), req.StartDate, req.EndDate, req.Role, req.Action)
	if err != nil {
		log.Printf("[AuditReport Handler] Store error: %v", err)
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", fmt.Sprintf("Failed to generate report: %v", err))
		return
	}

	log.Printf("[AuditReport Handler] Successfully generated report with %d records", len(data))
	response.OK(c, map[string]interface{}{
		"reportType": "audit",
		"filters": map[string]interface{}{
			"startDate": req.StartDate,
			"endDate":   req.EndDate,
			"role":      req.Role,
			"action":    req.Action,
		},
		"data": data,
	})
}

// DownloadReport generates and downloads report as PDF or CSV
func (h *MOHReportsHandler) DownloadReport(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	reportType := c.Param("type")
	format := c.DefaultQuery("format", "pdf")

	var req ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationError(c, "Invalid query parameters", nil)
		return
	}

	// Generate report data based on type
	var data []map[string]interface{}
	var title string
	var columns []string
	var err error

	switch reportType {
	case "coverage":
		title = "Vaccination Coverage Report"
		columns = []string{"GN Division", "Total", "Vaccinated", "Coverage %"}
		data, err = h.ReportStore.VaccinationCoverageReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)

	case "missed":
		title = "Missed Vaccinations Report"
		columns = []string{"Child Name", "GN Division", "Vaccine", "Due Date"}
		data, err = h.ReportStore.MissedVaccinationReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)

	case "phm-performance":
		title = "PHM Performance Report"
		columns = []string{"PHM", "Area", "Total", "Vaccinated", "Missed", "Coverage %"}
		data, err = h.ReportStore.PHMPerformanceReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)

	case "audit":
		title = "Audit Report"
		columns = []string{"Date", "User", "Role", "Action", "Details"}
		data, err = h.ReportStore.AuditReport(c.Request.Context(), req.StartDate, req.EndDate, req.Role, req.Action)

	default:
		response.Error(c, http.StatusBadRequest, "INVALID_REPORT_TYPE", "Report type not found")
		return
	}

	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to generate report")
		return
	}

	switch format {
	case "csv":
		h.downloadAsCSV(c, title, columns, data)
	case "pdf":
		h.downloadAsPDF(c, title, columns, data, req)
	default:
		response.Error(c, http.StatusBadRequest, "INVALID_FORMAT", "Format must be pdf or csv")
	}
}

// downloadAsCSV generates and downloads CSV file
func (h *MOHReportsHandler) downloadAsCSV(c *gin.Context, title string, columns []string, data []map[string]interface{}) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, strings.ReplaceAll(title, " ", "_")))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write(columns)

	// Write data rows
	for _, row := range data {
		var values []string
		for _, col := range columns {
			// Map column names to data keys (with special handling for different column names)
			key := strings.ToLower(strings.ReplaceAll(col, " ", ""))
			value := ""
			if v, ok := row[key]; ok {
				value = fmt.Sprintf("%v", v)
			} else if v, ok := row[strings.ToLower(col)]; ok {
				value = fmt.Sprintf("%v", v)
			}
			values = append(values, value)
		}
		writer.Write(values)
	}
}

// downloadAsPDF generates and downloads PDF file
func (h *MOHReportsHandler) downloadAsPDF(c *gin.Context, title string, columns []string, data []map[string]interface{}, req ReportRequest) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, title)
	pdf.Ln(12)

	// Add filter information
	pdf.SetFont("Arial", "", 10)
	if req.StartDate != "" {
		pdf.Cell(0, 5, fmt.Sprintf("Start Date: %s", req.StartDate))
		pdf.Ln(4)
	}
	if req.EndDate != "" {
		pdf.Cell(0, 5, fmt.Sprintf("End Date: %s", req.EndDate))
		pdf.Ln(4)
	}
	if req.GNDivision != "" {
		pdf.Cell(0, 5, fmt.Sprintf("GN Division: %s", req.GNDivision))
		pdf.Ln(4)
	}
	pdf.Ln(4)

	// Add table header
	pdf.SetFont("Arial", "B", 10)
	colWidth := 190.0 / float64(len(columns))
	for _, col := range columns {
		pdf.CellFormat(colWidth, 7, col, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Add table data
	pdf.SetFont("Arial", "", 9)
	for _, row := range data {
		for i, col := range columns {
			key := strings.ToLower(strings.ReplaceAll(col, " ", ""))
			value := ""
			if v, ok := row[key]; ok {
				value = fmt.Sprintf("%v", v)
			} else if v, ok := row[strings.ToLower(col)]; ok {
				value = fmt.Sprintf("%v", v)
			}
			if i == len(columns)-1 {
				pdf.CellFormat(colWidth, 6, value, "1", 1, "L", false, 0, "")
			} else {
				pdf.CellFormat(colWidth, 6, value, "1", 0, "L", false, 0, "")
			}
		}
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, strings.ReplaceAll(title, " ", "_")))
	pdf.Output(c.Writer)
}

// GetReportData is a convenience endpoint that returns report data as JSON
// This is used by the frontend to fetch data before downloading
func (h *MOHReportsHandler) GetReportData(c *gin.Context) {
	reportType := c.Param("type")

	var req ReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.ValidationError(c, "Invalid query parameters", nil)
		return
	}

	var data interface{}
	var err error

	switch reportType {
	case "coverage":
		data, err = h.ReportStore.VaccinationCoverageReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)
	case "missed":
		data, err = h.ReportStore.MissedVaccinationReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)
	case "phm-performance":
		data, err = h.ReportStore.PHMPerformanceReport(c.Request.Context(), req.StartDate, req.EndDate, req.GNDivision)
	case "audit":
		data, err = h.ReportStore.AuditReport(c.Request.Context(), req.StartDate, req.EndDate, req.Role, req.Action)
	default:
		response.Error(c, http.StatusBadRequest, "INVALID_REPORT_TYPE", "Report type not found")
		return
	}

	if err != nil {
		response.Error(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch report data")
		return
	}

	// Return as raw JSON
	jsonData, _ := json.Marshal(data)
	c.Data(http.StatusOK, "application/json", jsonData)
}

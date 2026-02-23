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

type VaccinationRecordsHandler struct {
	RecordStore *store.VaccinationRecordStore
}

type CreateVaccinationRecordRequest struct {
	ChildId           string  `json:"childId" binding:"required"`
	VaccineId         string  `json:"vaccineId" binding:"required"`
	AdministeredDate  string  `json:"administeredDate" binding:"required"`
	BatchNumber       string  `json:"batchNumber"`
	AdministeredBy    string  `json:"administeredBy"`
	Location          string  `json:"location"`
	Site              string  `json:"site"`
	DoseNumber        *int    `json:"doseNumber"`
	NextDueDate       *string `json:"nextDueDate"`
	Status            string  `json:"status"`
	Notes             string  `json:"notes"`
}

func (h *VaccinationRecordsHandler) Create(c *gin.Context) {
	var req CreateVaccinationRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	if req.Status == "" {
		req.Status = "administered"
	}
	recordID := "rec-" + uuid.New().String()[:8]
	err := h.RecordStore.Create(c.Request.Context(), recordID, req.ChildId, req.VaccineId, req.AdministeredDate, req.BatchNumber,
		req.AdministeredBy, req.Location, req.Site, req.DoseNumber, req.NextDueDate, req.Status, req.Notes)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to create record"))
		return
	}
	response.Created(c, gin.H{"recordId": recordID, "message": "Vaccination recorded successfully."})
}

func (h *VaccinationRecordsHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	childID := c.Query("childId")
	if childID != "" {
		list, err := h.RecordStore.ByChildID(c.Request.Context(), childID)
		if err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list records"))
			return
		}
		response.OK(c, list)
		return
	}
	if claims.Role != "moh" {
		response.AbortWithError(c, errors.ErrForbidden)
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
	total, list, err := h.RecordStore.ListMOH(c.Request.Context(), c.Query("areaCode"), c.Query("vaccineId"), c.Query("status"), c.Query("startDate"), c.Query("endDate"), page, limit)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list records"))
		return
	}
	response.OK(c, gin.H{"total": total, "page": page, "limit": limit, "data": list})
}

func (h *VaccinationRecordsHandler) GetByID(c *gin.Context) {
	recordID := c.Param("recordId")
	r, err := h.RecordStore.GetByID(c.Request.Context(), recordID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	response.OK(c, r)
}

func (h *VaccinationRecordsHandler) Update(c *gin.Context) {
	recordID := c.Param("recordId")
	var req struct {
		VaccineId        *string `json:"vaccineId"`
		AdministeredDate *string `json:"administeredDate"`
		BatchNumber      *string `json:"batchNumber"`
		AdministeredBy   *string `json:"administeredBy"`
		Location         *string `json:"location"`
		Site             *string `json:"site"`
		DoseNumber       *int    `json:"doseNumber"`
		NextDueDate      *string `json:"nextDueDate"`
		Status           *string `json:"status"`
		Notes            *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	vaccineID := ""
	adminDate := ""
	batchNum := ""
	adminBy := ""
	loc := ""
	site := ""
	nextDue := ""
	status := ""
	notes := ""
	if req.VaccineId != nil {
		vaccineID = *req.VaccineId
	}
	if req.AdministeredDate != nil {
		adminDate = *req.AdministeredDate
	}
	if req.BatchNumber != nil {
		batchNum = *req.BatchNumber
	}
	if req.AdministeredBy != nil {
		adminBy = *req.AdministeredBy
	}
	if req.Location != nil {
		loc = *req.Location
	}
	if req.Site != nil {
		site = *req.Site
	}
	if req.NextDueDate != nil {
		nextDue = *req.NextDueDate
	}
	if req.Status != nil {
		status = *req.Status
	}
	if req.Notes != nil {
		notes = *req.Notes
	}
	err := h.RecordStore.Update(c.Request.Context(), recordID, vaccineID, adminDate, batchNum, adminBy, loc, site, req.DoseNumber, &nextDue, status, notes)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update record"))
		return
	}
	response.OK(c, gin.H{"message": "Vaccination record updated successfully."})
}

func (h *VaccinationRecordsHandler) Delete(c *gin.Context) {
	recordID := c.Param("recordId")
	err := h.RecordStore.Delete(c.Request.Context(), recordID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to delete record"))
		return
	}
	response.OK(c, gin.H{"message": "Vaccination record deleted successfully."})
}

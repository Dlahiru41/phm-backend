package handlers

import (
	"ncvms/internal/errors"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GrowthHandler struct {
	GrowthStore *store.GrowthRecordStore
}

type CreateGrowthRecordRequest struct {
	ChildId           string   `json:"childId" binding:"required"`
	RecordedDate      string   `json:"recordedDate" binding:"required"`
	Height            *float64 `json:"height"`
	Weight            *float64 `json:"weight"`
	HeadCircumference *float64 `json:"headCircumference"`
	RecordedBy        string   `json:"recordedBy"`
	Notes             string   `json:"notes"`
}

func (h *GrowthHandler) Create(c *gin.Context) {
	var req CreateGrowthRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	recordID := "growth-" + uuid.New().String()[:8]
	err := h.GrowthStore.Create(c.Request.Context(), recordID, req.ChildId, req.RecordedDate, req.RecordedBy, req.Notes,
		req.Height, req.Weight, req.HeadCircumference)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to record growth"))
		return
	}
	response.Created(c, gin.H{"recordId": recordID, "message": "Growth data recorded successfully."})
}

func (h *GrowthHandler) List(c *gin.Context) {
	childID := c.Query("childId")
	if childID == "" {
		response.ValidationError(c, "childId is required", nil)
		return
	}
	list, err := h.GrowthStore.ByChildID(c.Request.Context(), childID, c.Query("startDate"), c.Query("endDate"))
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list growth records"))
		return
	}
	response.OK(c, list)
}

func (h *GrowthHandler) Charts(c *gin.Context) {
	childID := c.Query("childId")
	if childID == "" {
		response.ValidationError(c, "childId is required", nil)
		return
	}
	charts, err := h.GrowthStore.ChartsByChildID(c.Request.Context(), childID, c.Query("startDate"), c.Query("endDate"))
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to load growth chart data"))
		return
	}
	response.OK(c, charts)
}

func (h *GrowthHandler) WHOByChildID(c *gin.Context) {
	childID := c.Param("childId")
	if childID == "" {
		response.ValidationError(c, "childId is required", nil)
		return
	}
	payload, err := h.GrowthStore.WHOPayloadByChildID(c.Request.Context(), childID, c.Query("startDate"), c.Query("endDate"))
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to load WHO growth payload"))
		return
	}
	if payload == nil {
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	response.OK(c, payload)
}

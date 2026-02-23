package handlers

import (
	"ncvms/internal/errors"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SchedulesHandler struct {
	ScheduleStore *store.ScheduleStore
}

type CreateScheduleRequest struct {
	ChildId       string `json:"childId" binding:"required"`
	VaccineId     string `json:"vaccineId" binding:"required"`
	ScheduledDate string `json:"scheduledDate" binding:"required"`
	DueDate       string `json:"dueDate" binding:"required"`
}

func (h *SchedulesHandler) List(c *gin.Context) {
	childID := c.Query("childId")
	if childID == "" {
		response.ValidationError(c, "childId is required", nil)
		return
	}
	list, err := h.ScheduleStore.ByChildID(c.Request.Context(), childID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list schedule"))
		return
	}
	response.OK(c, list)
}

func (h *SchedulesHandler) Create(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	scheduleID := "sch-" + uuid.New().String()[:8]
	err := h.ScheduleStore.Create(c.Request.Context(), scheduleID, req.ChildId, req.VaccineId, req.ScheduledDate, req.DueDate)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to create schedule"))
		return
	}
	response.Created(c, gin.H{"scheduleId": scheduleID, "message": "Schedule item created successfully."})
}

func (h *SchedulesHandler) UpdateStatus(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	var req struct {
		Status string `json:"status" binding:"required,oneof=pending scheduled completed missed cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	err := h.ScheduleStore.UpdateStatus(c.Request.Context(), scheduleID, req.Status)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update status"))
		return
	}
	response.OK(c, gin.H{"message": "Schedule status updated successfully."})
}

func (h *SchedulesHandler) SendReminder(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	err := h.ScheduleStore.SetReminderSent(c.Request.Context(), scheduleID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to send reminder"))
		return
	}
	response.OK(c, gin.H{"message": "Reminder sent successfully."})
}

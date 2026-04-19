package handlers

import (
	"fmt"
	"log"
	"strings"

	"ncvms/internal/errors"
	"ncvms/internal/messaging"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SchedulesHandler struct {
	ScheduleStore     *store.ScheduleStore
	NotificationStore *store.NotificationStore
	WhatsAppSender    messaging.WhatsAppSender
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

	if item, err := h.ScheduleStore.GetNotificationContextByScheduleID(c.Request.Context(), scheduleID); err != nil {
		log.Printf("[schedule] failed to load notification context schedule=%s err=%v", scheduleID, err)
	} else if item != nil {
		message := fmt.Sprintf("Your child has a vaccination due on %s. Please attend the clinic.", strings.TrimSpace(item.DueDate))
		if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
			notificationID := "notif-" + uuid.New().String()[:8]
			_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "vaccination_due", message, &item.ChildId)
		}
		if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
			if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
				log.Printf("[schedule] failed to send scheduled vaccination sms schedule=%s child=%s err=%v", scheduleID, item.ChildId, err)
			}
		}
		_ = h.ScheduleStore.SetReminderSent(c.Request.Context(), scheduleID)
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

	if req.Status == "cancelled" {
		if item, err := h.ScheduleStore.GetNotificationContextByScheduleID(c.Request.Context(), scheduleID); err != nil {
			log.Printf("[schedule] failed to load cancellation context schedule=%s err=%v", scheduleID, err)
		} else if item != nil {
			message := fmt.Sprintf("The scheduled vaccination on %s has been cancelled. Please wait for further updates.", strings.TrimSpace(item.DueDate))
			if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
				notificationID := "notif-" + uuid.New().String()[:8]
				_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "cancelled_vaccination", message, &item.ChildId)
			}
			if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
				if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
					log.Printf("[schedule] failed to send cancellation sms schedule=%s child=%s err=%v", scheduleID, item.ChildId, err)
				}
			}
		}
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

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

type NotificationsHandler struct {
	NotificationStore *store.NotificationStore
}

type CreateNotificationRequest struct {
	RecipientId    string  `json:"recipientId" binding:"required"`
	Type           string  `json:"type" binding:"required,oneof=reminder missed upcoming info vaccination_due growth_record clinic_reminder missed_vaccination missed_clinic cancelled_clinic cancelled_vaccination"`
	Message        string  `json:"message" binding:"required"`
	RelatedChildId *string `json:"relatedChildId"`
}

func (h *NotificationsHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	unreadOnly := c.Query("unread") == "true"
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 20
	}
	total, unreadCount, list, err := h.NotificationStore.List(c.Request.Context(), claims.UserId, unreadOnly, page, limit)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list notifications"))
		return
	}
	response.OK(c, gin.H{"total": total, "unreadCount": unreadCount, "data": list})
}

func (h *NotificationsHandler) MarkRead(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	notificationID := c.Param("notificationId")
	err := h.NotificationStore.MarkRead(c.Request.Context(), notificationID, claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to mark as read"))
		return
	}
	response.OK(c, gin.H{"message": "Notification marked as read."})
}

func (h *NotificationsHandler) MarkAllRead(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	err := h.NotificationStore.MarkAllRead(c.Request.Context(), claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to mark all as read"))
		return
	}
	response.OK(c, gin.H{"message": "All notifications marked as read."})
}

func (h *NotificationsHandler) Create(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	notificationID := "notif-" + uuid.New().String()[:8]
	err := h.NotificationStore.Create(c.Request.Context(), notificationID, req.RecipientId, req.Type, req.Message, req.RelatedChildId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to send notification"))
		return
	}
	response.Created(c, gin.H{"notificationId": notificationID, "message": "Notification sent successfully."})
}

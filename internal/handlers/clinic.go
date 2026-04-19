package handlers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ncvms/internal/errors"
	"ncvms/internal/messaging"
	"ncvms/internal/middleware"
	"ncvms/internal/models"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClinicHandler struct {
	ClinicStore       *store.ClinicStore
	UserStore         *store.UserStore
	NotificationStore *store.NotificationStore
	WhatsAppSender    messaging.WhatsAppSender
}

type CreateClinicRequest struct {
	ClinicDate  string `json:"clinicDate" binding:"required"`
	GnDivision  string `json:"gnDivision"`
	Location    string `json:"location" binding:"required"`
	Description string `json:"description"`
}

type UpdateClinicStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=scheduled completed cancelled"`
}

type UpdateAttendanceRequest struct {
	ChildId string `json:"childId" binding:"required"`
	Status  string `json:"status" binding:"required,oneof=attended not_attended"`
}

// CreateClinic creates a new clinic and notifies all parents in the clinic GN division.
func (h *ClinicHandler) CreateClinic(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req CreateClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	if h.UserStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "User store is not configured"))
		return
	}

	phm, err := h.UserStore.GetByID(c.Request.Context(), claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "PHM user not found"))
		return
	}

	assignedArea := ""
	if phm.AssignedArea != nil {
		assignedArea = strings.TrimSpace(*phm.AssignedArea)
	}
	if assignedArea == "" {
		response.AbortWithError(c, errors.New(errors.ErrBadRequest.Status, "BAD_REQUEST", "PHM assigned area is not configured"))
		return
	}

	if strings.TrimSpace(req.GnDivision) != "" && !strings.EqualFold(strings.TrimSpace(req.GnDivision), assignedArea) {
		response.ValidationError(c, "gnDivision must match the PHM assigned area", nil)
		return
	}

	clinicID := "clinic-" + uuid.New().String()[:8]
	clinic := &models.ClinicSchedule{
		ClinicId:    clinicID,
		PhmId:       claims.UserId,
		ClinicDate:  req.ClinicDate,
		GnDivision:  assignedArea,
		Location:    req.Location,
		Description: req.Description,
		Status:      "scheduled",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.ClinicStore.Create(c.Request.Context(), clinic); err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to create clinic"))
		return
	}

	allChildren, err := h.ClinicStore.ListChildrenForClinic(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch clinic children"))
		return
	}

	notificationCount := 0
	smsCount := 0
	for _, item := range allChildren {
		clinicChild := &models.ClinicChild{
			ClinicChildId:    "cc-" + uuid.New().String()[:8],
			ClinicId:         clinicID,
			ChildId:          item.ChildId,
			Attended:         false,
			AttendanceStatus: "pending",
			MissedNotified:   false,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if err := h.ClinicStore.CreateClinicChild(c.Request.Context(), clinicChild); err != nil {
			log.Printf("[clinic] failed to create clinic child mapping clinic=%s child=%s err=%v", clinicID, item.ChildId, err)
		}

		if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
			message := fmt.Sprintf("%s has a scheduled clinic session at %s on %s.", defaultChildName(item.ChildName), req.Location, req.ClinicDate)
			notificationID := "notif-" + uuid.New().String()[:8]
			if err := h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "clinic_reminder", message, &item.ChildId); err == nil {
				notificationCount++
			}
		}

		if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
			smsMessage := fmt.Sprintf("%s has a clinic session at %s on %s. Please attend.", defaultChildName(item.ChildName), req.Location, req.ClinicDate)
			if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), smsMessage); err == nil {
				smsCount++
			} else {
				log.Printf("[clinic] failed to send clinic sms clinic=%s child=%s err=%v", clinicID, item.ChildId, err)
			}
		}
	}

	dueChildren, dueErr := h.ClinicStore.GetDueChildren(c.Request.Context(), clinicID)
	if dueErr != nil {
		log.Printf("[clinic] failed to fetch due children clinic=%s err=%v", clinicID, dueErr)
		dueChildren = []models.DueChild{}
	}

	response.Created(c, gin.H{
		"clinic":                  clinic,
		"childrenInDivision":      len(allChildren),
		"dueChildren":             dueChildren,
		"parentNotificationCount": notificationCount,
		"parentSMSCount":          smsCount,
	})
}

// GetClinic retrieves a clinic by ID
func (h *ClinicHandler) GetClinic(c *gin.Context) {
	clinicID := c.Param("clinicId")
	if clinicID == "" {
		response.ValidationError(c, "clinicId is required", nil)
		return
	}

	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	response.OK(c, clinic)
}

// ListMyClinics retrieves all clinics for the authenticated PHM
func (h *ClinicHandler) ListMyClinics(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	fromDate := c.Query("fromDate")
	toDate := c.Query("toDate")

	clinics, err := h.ClinicStore.ListByPHM(c.Request.Context(), claims.UserId, &fromDate, &toDate)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list clinics"))
		return
	}

	if clinics == nil {
		clinics = []models.ClinicSchedule{}
	}

	response.OK(c, clinics)
}

// GetDueChildren retrieves children due for a specific clinic
func (h *ClinicHandler) GetDueChildren(c *gin.Context) {
	clinicID := c.Param("clinicId")
	if clinicID == "" {
		response.ValidationError(c, "clinicId is required", nil)
		return
	}

	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	claims := middleware.GetClaims(c)
	if claims != nil && claims.Role == "phm" && clinic.PhmId != claims.UserId {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	dueChildren, err := h.ClinicStore.GetDueChildren(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch due children"))
		return
	}

	if dueChildren == nil {
		dueChildren = []models.DueChild{}
	}

	response.OK(c, dueChildren)
}

// UpdateClinicStatus updates the status of a clinic and triggers missed-clinic alerts when completed.
func (h *ClinicHandler) UpdateClinicStatus(c *gin.Context) {
	clinicID := c.Param("clinicId")
	if clinicID == "" {
		response.ValidationError(c, "clinicId is required", nil)
		return
	}

	var req UpdateClinicStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	claims := middleware.GetClaims(c)
	if claims != nil && claims.Role == "phm" && clinic.PhmId != claims.UserId {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if clinic.Status == "completed" && req.Status != "completed" {
		response.AbortWithError(c, errors.New(errors.ErrBadRequest.Status, "BAD_REQUEST", "Clinic is completed and locked"))
		return
	}

	if clinic.Status == req.Status {
		response.OK(c, gin.H{
			"clinic":           clinic,
			"missedAlerted":    0,
			"cancelledAlerted": 0,
		})
		return
	}

	if err := h.ClinicStore.UpdateClinicStatus(c.Request.Context(), clinicID, req.Status); err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update clinic status"))
		return
	}

	missedAlerts := 0
	cancelledAlerts := 0
	if req.Status == "completed" {
		alerts, err := h.ClinicStore.MarkMissedClinicAttendance(c.Request.Context(), clinicID)
		if err != nil {
			log.Printf("[clinic] failed to mark missed clinic attendance clinic=%s err=%v", clinicID, err)
		} else {
			for _, item := range alerts {
				message := clinicMissedMessage(clinic.ClinicDate)
				if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
					notificationID := "notif-" + uuid.New().String()[:8]
					_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "missed_clinic", message, &item.ChildId)
				}
				if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
					if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
						log.Printf("[clinic] failed to send missed clinic sms clinic=%s child=%s err=%v", clinicID, item.ChildId, err)
					}
				}
				_ = h.ClinicStore.SetClinicChildMissedNotified(c.Request.Context(), clinicID, item.ChildId)
				missedAlerts++
			}
		}
	}

	if req.Status == "cancelled" {
		children, err := h.ClinicStore.ListChildrenForClinic(c.Request.Context(), clinicID)
		if err != nil {
			log.Printf("[clinic] failed to load clinic children for cancellation clinic=%s err=%v", clinicID, err)
		} else {
			message := clinicCancelledMessage(clinic.ClinicDate)
			for _, item := range children {
				if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
					notificationID := "notif-" + uuid.New().String()[:8]
					_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "cancelled_clinic", message, &item.ChildId)
				}
				if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
					if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
						log.Printf("[clinic] failed to send cancelled clinic sms clinic=%s child=%s err=%v", clinicID, item.ChildId, err)
					}
				}
				cancelledAlerts++
			}
		}
	}

	updatedClinic, _ := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	response.OK(c, gin.H{
		"clinic":           updatedClinic,
		"missedAlerted":    missedAlerts,
		"cancelledAlerted": cancelledAlerts,
	})
}

// UpdateAttendance marks clinic attendance as attended or not_attended.
func (h *ClinicHandler) UpdateAttendance(c *gin.Context) {
	clinicID := c.Param("clinicId")
	if clinicID == "" {
		response.ValidationError(c, "clinicId is required", nil)
		return
	}

	var req UpdateAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	claims := middleware.GetClaims(c)
	if claims != nil && claims.Role == "phm" && clinic.PhmId != claims.UserId {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if clinic.Status == "completed" {
		response.AbortWithError(c, errors.New(errors.ErrBadRequest.Status, "BAD_REQUEST", "Clinic is completed and attendance is locked"))
		return
	}

	if err := h.ClinicStore.UpdateClinicChildAttendance(c.Request.Context(), clinicID, req.ChildId, req.Status); err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update attendance"))
		return
	}

	if req.Status == "not_attended" {
		item, err := h.ClinicStore.GetClinicAttendanceAlertByChild(c.Request.Context(), clinicID, req.ChildId)
		if err != nil {
			log.Printf("[clinic] failed to fetch clinic attendance alert payload clinic=%s child=%s err=%v", clinicID, req.ChildId, err)
		} else if item != nil && !item.MissedNotified {
			message := clinicMissedMessage(clinic.ClinicDate)
			if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
				notificationID := "notif-" + uuid.New().String()[:8]
				_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "missed_clinic", message, &item.ChildId)
			}
			if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
				if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
					log.Printf("[clinic] failed to send manual not-attended sms clinic=%s child=%s err=%v", clinicID, req.ChildId, err)
				}
			}
			_ = h.ClinicStore.SetClinicChildMissedNotified(c.Request.Context(), clinicID, req.ChildId)
		}
	}

	response.OK(c, gin.H{"message": "Attendance updated successfully"})
}

// GetClinicChildren retrieves all children for a clinic (with attendance status)
func (h *ClinicHandler) GetClinicChildren(c *gin.Context) {
	clinicID := c.Param("clinicId")
	if clinicID == "" {
		response.ValidationError(c, "clinicId is required", nil)
		return
	}

	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	claims := middleware.GetClaims(c)
	if claims != nil && claims.Role == "phm" && clinic.PhmId != claims.UserId {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	clinicChildren, err := h.ClinicStore.GetClinicChildren(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch clinic children"))
		return
	}

	if clinicChildren == nil {
		clinicChildren = []models.ClinicChild{}
	}

	response.OK(c, clinicChildren)
}

// ListParentDueVaccinations returns due vaccinations for the authenticated parent's linked children
// when those children have an upcoming scheduled clinic.
func (h *ClinicHandler) ListParentDueVaccinations(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "parent" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}

	items, err := h.ClinicStore.ListParentDueVaccinations(c.Request.Context(), claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch due vaccinations"))
		return
	}

	if items == nil {
		items = []models.ParentDueVaccination{}
	}

	for i := range items {
		childName := strings.TrimSpace(items[i].ChildName)
		if childName == "" {
			childName = "Your child"
		}
		items[i].ClinicReminder = fmt.Sprintf("%s has a vaccination due. Please go to the scheduled clinic at %s on %s.", childName, items[i].ClinicLocation, items[i].ClinicDate)
	}

	response.OK(c, gin.H{
		"count": len(items),
		"items": items,
	})
}

func defaultChildName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Your child"
	}
	return name
}

func clinicMissedMessage(clinicDate string) string {
	return fmt.Sprintf("Your child missed the clinic session held on %s. Please contact your PHM or visit the next available clinic.", strings.TrimSpace(clinicDate))
}

func clinicCancelledMessage(clinicDate string) string {
	return fmt.Sprintf("The clinic scheduled on %s has been cancelled. Please wait for further updates.", strings.TrimSpace(clinicDate))
}

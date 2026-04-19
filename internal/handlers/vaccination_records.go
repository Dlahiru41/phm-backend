package handlers

import (
	"fmt"
	"log"
	"sort"
	"strconv"
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

type VaccinationRecordsHandler struct {
	RecordStore       *store.VaccinationRecordStore
	ChildStore        *store.ChildStore
	ScheduleStore     *store.ScheduleStore
	NotificationStore *store.NotificationStore
	WhatsAppSender    messaging.WhatsAppSender
}

type CreateVaccinationRecordRequest struct {
	ChildId          string  `json:"childId" binding:"required"`
	VaccineId        string  `json:"vaccineId" binding:"required"`
	AdministeredDate string  `json:"administeredDate" binding:"required"`
	BatchNumber      string  `json:"batchNumber"`
	AdministeredBy   string  `json:"administeredBy"`
	Location         string  `json:"location"`
	Site             string  `json:"site"`
	DoseNumber       *int    `json:"doseNumber"`
	NextDueDate      *string `json:"nextDueDate"`
	Status           string  `json:"status"`
	Notes            string  `json:"notes"`
}

type UpdateVaccinationTrackingRequest struct {
	ScheduleId       string `json:"scheduleId" binding:"required"`
	Status           string `json:"status" binding:"required,oneof=completed not_attended"`
	AdministeredDate string `json:"administeredDate"`
	Location         string `json:"location"`
	Notes            string `json:"notes"`
}

type VaccinationCardPayload struct {
	Title              string                   `json:"title"`
	FileName           string                   `json:"fileName"`
	GeneratedAt        string                   `json:"generatedAt"`
	Child              VaccinationCardChild     `json:"child"`
	VaccinationHistory []VaccinationCardHistory `json:"vaccinationHistory"`
}

type VaccinationCardChild struct {
	ChildId            string `json:"childId"`
	Name               string `json:"name"`
	DateOfBirth        string `json:"dateOfBirth"`
	RegistrationNumber string `json:"registrationNumber"`
	Gender             string `json:"gender"`
}

type VaccinationCardHistory struct {
	VaccineName string `json:"vaccineName"`
	DoseNumber  *int   `json:"doseNumber,omitempty"`
	DateGiven   string `json:"dateGiven"`
	NextDueDate string `json:"nextDueDate,omitempty"`
	Status      string `json:"status"`
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
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
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

// ListDueForPHM returns due vaccinations, triggers due reminders, and marks overdue items as missed.
func (h *VaccinationRecordsHandler) ListDueForPHM(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "phm" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}

	if h.ScheduleStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Schedule store is not configured"))
		return
	}

	if err := h.processMissedVaccinations(c, claims.UserId); err != nil {
		log.Printf("[vaccination-due] missed processing failed phm=%s err=%v", claims.UserId, err)
	}

	items, err := h.ScheduleStore.ListDueForPHM(c.Request.Context(), claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch due vaccinations"))
		return
	}

	for i := range items {
		items[i].DueNotificationText = fmt.Sprintf("%s has a vaccination due on %s. Please attend the clinic.", safeChildName(items[i].ChildName), items[i].DueDate)
		if !items[i].ReminderSent {
			h.sendDueVaccinationReminder(c, items[i])
			_ = h.ScheduleStore.SetReminderSent(c.Request.Context(), items[i].ScheduleId)
			items[i].ReminderSent = true
		}
	}

	response.OK(c, gin.H{
		"count": len(items),
		"items": items,
	})
}

// UpdateTracking allows PHM to mark vaccination as completed or not_attended.
func (h *VaccinationRecordsHandler) UpdateTracking(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "phm" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	if h.ScheduleStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Schedule store is not configured"))
		return
	}

	var req UpdateVaccinationTrackingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	sch, err := h.ScheduleStore.GetByID(c.Request.Context(), req.ScheduleId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Schedule not found"))
		return
	}

	if req.Status == "completed" {
		adminDate := strings.TrimSpace(req.AdministeredDate)
		if adminDate == "" {
			adminDate = time.Now().Format("2006-01-02")
		}
		recordID := "rec-" + uuid.New().String()[:8]
		if err := h.RecordStore.Create(
			c.Request.Context(),
			recordID,
			sch.ChildId,
			sch.VaccineId,
			adminDate,
			"",
			claims.UserId,
			strings.TrimSpace(req.Location),
			"",
			nil,
			nil,
			"administered",
			strings.TrimSpace(req.Notes),
		); err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to record vaccination completion"))
			return
		}

		if err := h.ScheduleStore.UpdateStatus(c.Request.Context(), sch.ScheduleId, "completed"); err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update schedule status"))
			return
		}

		if item, err := h.ScheduleStore.GetNotificationContextByScheduleID(c.Request.Context(), sch.ScheduleId); err != nil {
			log.Printf("[vaccination-tracking] completion context fetch failed schedule=%s err=%v", sch.ScheduleId, err)
		} else if item != nil && h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
			successMessage := "Your child has successfully received the scheduled vaccination."
			if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), successMessage); err != nil {
				log.Printf("[vaccination-tracking] completion sms failed schedule=%s child=%s err=%v", sch.ScheduleId, item.ChildId, err)
			}
		}

		response.OK(c, gin.H{"message": "Vaccination marked as completed"})
		return
	}

	if err := h.ScheduleStore.UpdateStatus(c.Request.Context(), sch.ScheduleId, "missed"); err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update schedule status"))
		return
	}

	if item, err := h.ScheduleStore.GetNotificationContextByScheduleID(c.Request.Context(), sch.ScheduleId); err != nil {
		log.Printf("[vaccination-tracking] missed context fetch failed schedule=%s err=%v", sch.ScheduleId, err)
	} else if item != nil {
		h.sendMissedVaccinationAlert(c, *item)
		_ = h.ScheduleStore.SetMissedNotified(c.Request.Context(), item.ScheduleId)
	}

	response.OK(c, gin.H{"message": "Vaccination marked as not attended"})
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

// DownloadVaccinationCard returns payload data for client-side vaccination card PDF generation.
func (h *VaccinationRecordsHandler) DownloadVaccinationCard(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "parent" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
	if h.ChildStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "INTERNAL_ERROR", "Child store is not configured"))
		return
	}

	childID := strings.TrimSpace(c.Param("child_id"))
	if childID == "" {
		response.AbortWithError(c, errors.New(errors.ErrBadRequest.Status, "BAD_REQUEST", "child_id is required"))
		return
	}

	child, err := h.ChildStore.GetByID(c.Request.Context(), childID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "INTERNAL_ERROR", "Failed to load child details"))
		return
	}
	if child.ParentId == nil || strings.TrimSpace(*child.ParentId) != claims.UserId {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}

	records, err := h.RecordStore.ByChildID(c.Request.Context(), childID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "INTERNAL_ERROR", "Failed to load vaccination records"))
		return
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].AdministeredDate < records[j].AdministeredDate
	})

	fullName := strings.TrimSpace(strings.TrimSpace(child.FirstName) + " " + strings.TrimSpace(child.LastName))
	if fullName == "" {
		fullName = "-"
	}

	history := make([]VaccinationCardHistory, 0, len(records))
	for _, record := range records {
		item := VaccinationCardHistory{
			VaccineName: valueOrDash(record.VaccineName),
			DoseNumber:  record.DoseNumber,
			DateGiven:   valueOrDash(record.AdministeredDate),
			Status:      valueOrDash(record.Status),
		}
		if record.NextDueDate != nil {
			item.NextDueDate = valueOrDash(*record.NextDueDate)
		}
		history = append(history, item)
	}

	payload := VaccinationCardPayload{
		Title:       "Child Vaccination Card",
		FileName:    fmt.Sprintf("vaccination-card-%s.pdf", child.ChildId),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Child: VaccinationCardChild{
			ChildId:            child.ChildId,
			Name:               fullName,
			DateOfBirth:        valueOrDash(child.DateOfBirth),
			RegistrationNumber: valueOrDash(child.RegistrationNumber),
			Gender:             valueOrDash(child.Gender),
		},
		VaccinationHistory: history,
	}

	response.OK(c, payload)
}

func (h *VaccinationRecordsHandler) processMissedVaccinations(c *gin.Context, phmID string) error {
	items, err := h.ScheduleStore.MarkMissedDueVaccinations(c.Request.Context(), phmID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.MissedNotified {
			continue
		}
		h.sendMissedVaccinationAlert(c, item)
		_ = h.ScheduleStore.SetMissedNotified(c.Request.Context(), item.ScheduleId)
	}
	return nil
}

func (h *VaccinationRecordsHandler) sendDueVaccinationReminder(c *gin.Context, item models.PHMDueVaccination) {
	message := fmt.Sprintf("%s has a vaccination due on %s. Please attend the clinic.", safeChildName(item.ChildName), item.DueDate)
	if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
		notificationID := "notif-" + uuid.New().String()[:8]
		_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "vaccination_due", message, &item.ChildId)
	}
	if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
		if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
			log.Printf("[vaccination-due] sms failed schedule=%s child=%s err=%v", item.ScheduleId, item.ChildId, err)
		}
	}
}

func (h *VaccinationRecordsHandler) sendMissedVaccinationAlert(c *gin.Context, item models.PHMDueVaccination) {
	message := fmt.Sprintf("Your child missed the scheduled vaccination on %s. Please visit your nearest clinic or contact your PHM.", strings.TrimSpace(item.DueDate))
	if h.NotificationStore != nil && item.ParentId != nil && strings.TrimSpace(*item.ParentId) != "" {
		notificationID := "notif-" + uuid.New().String()[:8]
		_ = h.NotificationStore.Create(c.Request.Context(), notificationID, strings.TrimSpace(*item.ParentId), "missed_vaccination", message, &item.ChildId)
	}
	if h.WhatsAppSender != nil && item.ParentPhone != nil && strings.TrimSpace(*item.ParentPhone) != "" {
		if err := h.WhatsAppSender.SendMessage(c.Request.Context(), strings.TrimSpace(*item.ParentPhone), message); err != nil {
			log.Printf("[vaccination-missed] sms failed schedule=%s child=%s err=%v", item.ScheduleId, item.ChildId, err)
		}
	}
}

func safeChildName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Your child"
	}
	return name
}

func valueOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

func pointerOrDash(v *string) string {
	if v == nil {
		return "-"
	}
	return valueOrDash(*v)
}

func doseOrDash(v *int) string {
	if v == nil {
		return "-"
	}
	return strconv.Itoa(*v)
}

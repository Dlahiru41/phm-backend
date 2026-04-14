package handlers

import (
	"log"
	"time"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/models"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClinicHandler struct {
	ClinicStore       *store.ClinicStore
	NotificationStore *store.NotificationStore
}

type CreateClinicRequest struct {
	ClinicDate  string `json:"clinicDate" binding:"required"`
	GnDivision  string `json:"gnDivision" binding:"required"`
	Location    string `json:"location" binding:"required"`
	Description string `json:"description"`
}

type UpdateClinicStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=scheduled completed cancelled"`
}

type UpdateAttendanceRequest struct {
	ChildId  string `json:"childId" binding:"required"`
	Attended bool   `json:"attended"`
}

// CreateClinic creates a new clinic and automatically identifies due children
func (h *ClinicHandler) CreateClinic(c *gin.Context) {
	log.Println("=== CreateClinic START ===")

	claims := middleware.GetClaims(c)
	if claims == nil {
		log.Println("ERROR: No claims found")
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	log.Printf("✓ Auth OK - PHM ID: %s\n", claims.UserId)

	var req CreateClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR: JSON binding failed: %v\n", err)
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	log.Printf("✓ Request parsed - Date: %s, GN: %s, Location: %s\n", req.ClinicDate, req.GnDivision, req.Location)

	// Create clinic
	clinicID := "clinic-" + uuid.New().String()[:8]
	clinic := &models.ClinicSchedule{
		ClinicId:    clinicID,
		PhmId:       claims.UserId,
		ClinicDate:  req.ClinicDate,
		GnDivision:  req.GnDivision,
		Location:    req.Location,
		Description: req.Description,
		Status:      "scheduled",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	log.Printf("✓ Clinic object created - ID: %s\n", clinicID)

	err := h.ClinicStore.Create(c.Request.Context(), clinic)
	if err != nil {
		log.Printf("ERROR: Failed to create clinic record: %v\n", err)
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to create clinic"))
		return
	}
	log.Printf("✓ Clinic record created in database\n")

	// Get due children for this clinic
	log.Printf("→ Calling GetDueChildren for clinic: %s\n", clinicID)
	dueChildren, err := h.ClinicStore.GetDueChildren(c.Request.Context(), clinicID)
	if err != nil {
		log.Printf("ERROR: GetDueChildren failed: %v\n", err)
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch due children"))
		return
	}
	log.Printf("✓ GetDueChildren returned %d children\n", len(dueChildren))

	// Create clinic_children mappings
	uniqueChildren := make(map[string]bool)
	for _, dueChild := range dueChildren {
		if !uniqueChildren[dueChild.ChildId] {
			clinicChild := &models.ClinicChild{
				ClinicChildId: "cc-" + uuid.New().String()[:8],
				ClinicId:      clinicID,
				ChildId:       dueChild.ChildId,
				Attended:      false,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			err := h.ClinicStore.CreateClinicChild(c.Request.Context(), clinicChild)
			if err != nil {
				log.Printf("WARNING: Failed to create clinic_child mapping: %v\n", err)
			} else {
				log.Printf("✓ Created clinic_child mapping for child: %s\n", dueChild.ChildId)
			}
			uniqueChildren[dueChild.ChildId] = true

			// Send notification to parent if available
			if dueChild.ParentId != nil && *dueChild.ParentId != "" {
				clinicDateFormatted := req.ClinicDate
				message := "Your child has an upcoming vaccination appointment at " + req.Location + " on " + clinicDateFormatted + ". Please bring your child for immunization."

				notificationID := "notif-" + uuid.New().String()[:8]
				err := h.NotificationStore.Create(c.Request.Context(), notificationID, *dueChild.ParentId, "clinic_reminder", message, &dueChild.ChildId)
				if err != nil {
					log.Printf("WARNING: Failed to send notification: %v\n", err)
				} else {
					log.Printf("✓ Notification sent to parent: %s\n", *dueChild.ParentId)
				}
			}
		}
	}
	log.Printf("✓ All clinic_children mappings created: %d\n", len(uniqueChildren))

	log.Println("=== CreateClinic SUCCESS ===")
	response.Created(c, gin.H{
		"clinic":      clinic,
		"dueChildren": dueChildren,
		"childCount":  len(uniqueChildren),
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

	// Verify clinic exists
	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	// Check authorization (PHM can only see their own clinics)
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

// UpdateClinicStatus updates the status of a clinic
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

	// Verify clinic exists and user has permission
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

	err = h.ClinicStore.UpdateClinicStatus(c.Request.Context(), clinicID, req.Status)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update clinic status"))
		return
	}

	// Fetch and return updated clinic
	updatedClinic, _ := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	response.OK(c, updatedClinic)
}

// UpdateAttendance marks a child as attended
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

	// Verify clinic exists and user has permission
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

	err = h.ClinicStore.UpdateClinicChildAttendance(c.Request.Context(), clinicID, req.ChildId, req.Attended)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update attendance"))
		return
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

	// Verify clinic exists
	clinic, err := h.ClinicStore.GetByID(c.Request.Context(), clinicID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrNotFound.Status, "NOT_FOUND", "Clinic not found"))
		return
	}

	// Check authorization
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

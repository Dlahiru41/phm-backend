package handlers

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strings"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"ncvms/internal/messaging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	UserStore      *store.UserStore
	WhatsAppSender messaging.WhatsAppSender
	PHMLoginURL    string
}

type UpdateProfileRequest struct {
	Name               *string `json:"name"`
	PhoneNumber        *string `json:"phoneNumber"`
	Address            *string `json:"address"`
	LanguagePreference *string `json:"languagePreference"`
}

type UpdateSettingsRequest struct {
	LanguagePreference *string                `json:"languagePreference"`
	Notifications      map[string]interface{} `json:"notifications"`
}

type CreatePHMRequest struct {
	EmployeeId   string `json:"employeeId" binding:"required"`
	Name         string `json:"name" binding:"required"`
	NIC          string `json:"nic" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	PhoneNumber  string `json:"phoneNumber"`
	AssignedArea string `json:"assignedArea" binding:"required"`
}

func (h *UsersHandler) GetMe(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	user, err := h.UserStore.GetByID(c.Request.Context(), claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	response.OK(c, gin.H{
		"userId":             user.UserId,
		"email":              user.Email,
		"nic":                user.NIC,
		"name":               user.Name,
		"role":               user.Role,
		"phoneNumber":        user.PhoneNumber,
		"address":            user.Address,
		"languagePreference": user.LanguagePreference,
		"createdAt":          user.CreatedAt,
	})
}

func (h *UsersHandler) UpdateMe(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	err := h.UserStore.UpdateProfile(c.Request.Context(), claims.UserId, req.Name, req.PhoneNumber, req.Address, req.LanguagePreference)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update profile"))
		return
	}
	user, _ := h.UserStore.GetByID(c.Request.Context(), claims.UserId)
	response.OK(c, gin.H{
		"message": "Profile updated successfully.",
		"user": gin.H{
			"userId":             user.UserId,
			"email":              user.Email,
			"nic":                user.NIC,
			"name":               user.Name,
			"role":               user.Role,
			"phoneNumber":        user.PhoneNumber,
			"address":            user.Address,
			"languagePreference": user.LanguagePreference,
			"createdAt":          user.CreatedAt,
		},
	})
}

func (h *UsersHandler) UpdateSettings(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	err := h.UserStore.UpdateSettings(c.Request.Context(), claims.UserId, req.LanguagePreference, req.Notifications)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to save settings"))
		return
	}
	response.OK(c, gin.H{"message": "Settings saved successfully."})
}

func generateTemporaryPassword() (string, error) {
	// Generate a secure temporary password with format: TEMP-XXXXXXXX.
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := "TEMP-"

	for i := 0; i < 8; i++ {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		index := int(b[0]) % len(chars)
		password += string(chars[index])
	}

	return password, nil
}

func (h *UsersHandler) CreatePHM(c *gin.Context) {
	// Only MOH can create PHM accounts
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if claims.Role != "moh" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only MOH users can create PHM accounts")
		return
	}

	var req CreatePHMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	if strings.TrimSpace(req.PhoneNumber) == "" {
		response.ValidationError(c, "phoneNumber is required for PHM onboarding", []response.ErrorDetail{{Field: "phoneNumber", Message: "Must be provided to deliver temporary password."}})
		return
	}

	ctx := c.Request.Context()

	// Check if email already exists
	exists, _ := h.UserStore.ExistsByEmail(ctx, req.Email)
	if exists {
		response.Error(c, http.StatusConflict, "CONFLICT", "Email already registered")
		return
	}

	// Check if NIC already exists
	exists, _ = h.UserStore.ExistsByNIC(ctx, req.NIC)
	if exists {
		response.Error(c, http.StatusConflict, "CONFLICT", "NIC already registered")
		return
	}

	// Generate temporary password
	temporaryPassword, err := generateTemporaryPassword()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to generate temporary password")
		return
	}

	if h.WhatsAppSender == nil {
		response.AbortWithError(c, errors.New(http.StatusInternalServerError, "ERROR", "Messaging service is not configured"))
		return
	}

	phone := strings.TrimSpace(req.PhoneNumber)

	// Hash the temporary password
	hash, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to create account")
		return
	}

	// Create user ID
	userID := "user-phm-" + uuid.New().String()[:8]

	// Create PHM account with firstLogin=true and temporary password
	err = h.UserStore.CreatePHM(ctx, userID, req.EmployeeId, req.Email, req.NIC, string(hash), req.Name, req.PhoneNumber, req.AssignedArea, claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(http.StatusInternalServerError, "ERROR", "Failed to create PHM account"))
		return
	}

	message := h.buildPHMOnboardingMessage(req, temporaryPassword)

	deliveryStatus := "sent"
	note := "Temporary password has been sent to the provided phone number."
	if err := h.WhatsAppSender.SendMessage(ctx, phone, message); err != nil {
		// Account is already created; keep request successful and surface delivery issue.
		log.Printf("[phm-onboarding-message] failed userId=%s phone=%s: %v", userID, phone, err)
		deliveryStatus = "failed"
		note = "PHM account created, but temporary password delivery failed. Please retry sending the password."
	}

	response.Created(c, gin.H{
		"message":        "PHM account created successfully.",
		"userId":         userID,
		"employeeId":     req.EmployeeId,
		"firstLogin":     true,
		"deliveryStatus": deliveryStatus,
		"note":           note,
	})
}

func (h *UsersHandler) buildPHMOnboardingMessage(req CreatePHMRequest, temporaryPassword string) string {
	loginURL := strings.TrimSpace(h.PHMLoginURL)
	if loginURL == "" {
		loginURL = "https://suwacare.lk/login"
	}

	return fmt.Sprintf(
		"Hello %s,\n\nYour SuwaCareLK account has been created.\n\nEmployee ID: %s\nTemporary Password: %s\nAssigned Area: %s\nEmail: %s\n\nPlease login and change your password immediately.\n\nSystem Link: %s\n\n- Ministry of Health",
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.EmployeeId),
		temporaryPassword,
		strings.TrimSpace(req.AssignedArea),
		strings.TrimSpace(req.Email),
		loginURL,
	)
}

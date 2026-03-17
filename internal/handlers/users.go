package handlers

import (
	"crypto/rand"
	"net/http"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UsersHandler struct {
	UserStore *store.UserStore
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

func generateTemporaryPassword() string {
	// Generate a secure temporary password with format: TEMP-XXXXXXXX
	// Using crypto/rand for cryptographically secure random generation
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := "TEMP-"

	for i := 0; i < 8; i++ {
		// Generate a random index for the chars string
		b := make([]byte, 1)
		rand.Read(b)
		index := int(b[0]) % len(chars)
		password += string(chars[index])
	}

	return password
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
	temporaryPassword := generateTemporaryPassword()

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

	response.Created(c, gin.H{
		"message":           "PHM account created successfully.",
		"userId":            userID,
		"employeeId":        req.EmployeeId,
		"temporaryPassword": temporaryPassword,
		"firstLogin":        true,
		"note":              "Please use the temporary password for the first login and change it immediately.",
	})
}

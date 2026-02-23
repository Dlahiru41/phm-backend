package handlers

import (
	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
)

type UsersHandler struct {
	UserStore *store.UserStore
}

type UpdateProfileRequest struct {
	Name               *string `json:"name"`
	PhoneNumber        *string `json:"phoneNumber"`
	Address            *string `json:"address"`
	LanguagePreference  *string `json:"languagePreference"`
}

type UpdateSettingsRequest struct {
	LanguagePreference *string                `json:"languagePreference"`
	Notifications      map[string]interface{} `json:"notifications"`
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
		"email":             user.Email,
		"nic":               user.NIC,
		"name":              user.Name,
		"role":              user.Role,
		"phoneNumber":       user.PhoneNumber,
		"address":           user.Address,
		"languagePreference": user.LanguagePreference,
		"createdAt":         user.CreatedAt,
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
			"email":             user.Email,
			"nic":               user.NIC,
			"name":              user.Name,
			"role":              user.Role,
			"phoneNumber":       user.PhoneNumber,
			"address":           user.Address,
			"languagePreference": user.LanguagePreference,
			"createdAt":         user.CreatedAt,
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

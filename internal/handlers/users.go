package handlers

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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
	UserStore             *store.UserStore
	UserMobileChangeStore *store.UserMobileChangeOTPStore
	WhatsAppSender        messaging.WhatsAppSender
	PHMLoginURL           string
	OTPTTL                time.Duration
	OTPResendCooldown     time.Duration
	OTPMaxAttempts        int
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

type RequestMobileChangeRequest struct {
	NewPhoneNumber string `json:"newPhoneNumber" binding:"required"`
}

type VerifyMobileChangeRequest struct {
	NewPhoneNumber string `json:"newPhoneNumber" binding:"required"`
	OTPCode        string `json:"otpCode" binding:"required,len=6"`
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

	if req.PhoneNumber != nil {
		response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "Use /api/v1/users/request-mobile-change and OTP verification to change phone number"))
		return
	}

	err := h.UserStore.UpdateProfile(c.Request.Context(), claims.UserId, req.Name, nil, req.Address, req.LanguagePreference)
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

func (h *UsersHandler) RequestMobileChange(c *gin.Context) {
	if h.UserMobileChangeStore == nil || h.WhatsAppSender == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "OTP service is not configured"))
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req RequestMobileChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	newPhone, err := normalizePhone(req.NewPhoneNumber)
	if err != nil {
		response.AbortWithError(c, errors.New(422, "VALIDATION_ERROR", "newPhoneNumber must be a valid phone number"))
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

	if currentPhone, nerr := normalizePhone(user.PhoneNumber); nerr == nil && currentPhone == newPhone {
		response.AbortWithError(c, errors.New(409, "CONFLICT", "New mobile number is the same as your current mobile number"))
		return
	}

	exists, err := h.UserStore.ExistsByPhoneForOther(c.Request.Context(), newPhone, claims.UserId)
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process mobile change request"))
		return
	}
	if exists {
		response.AbortWithError(c, errors.New(409, "CONFLICT", "Mobile number is already in use"))
		return
	}

	if active, err := h.UserMobileChangeStore.GetLatestActive(c.Request.Context(), claims.UserId); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process mobile change request"))
		return
	} else if active != nil {
		retryAfter := active.CreatedAt.Add(h.getOTPResendCooldown()).Sub(time.Now())
		if retryAfter > 0 {
			response.AbortWithError(c, errors.New(429, "TOO_MANY_REQUESTS", fmt.Sprintf("Please wait %d seconds before requesting another OTP", int(retryAfter.Seconds())+1)))
			return
		}
	}

	otpCode, err := generateOTPCode()
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to generate OTP"))
		return
	}

	otpID := "otp-mc-" + uuid.NewString()[:8]
	expiresAt := time.Now().Add(h.getOTPTTL())

	if err := h.UserMobileChangeStore.InvalidateActive(c.Request.Context(), claims.UserId); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process mobile change request"))
		return
	}

	if err := h.UserMobileChangeStore.Create(c.Request.Context(), otpID, claims.UserId, newPhone, hashOTP(otpCode), expiresAt, h.getOTPMaxAttempts()); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process mobile change request"))
		return
	}

	if err := h.WhatsAppSender.SendOTP(c.Request.Context(), newPhone, otpCode, h.getOTPTTL()); err != nil {
		_ = h.UserMobileChangeStore.InvalidateActive(c.Request.Context(), claims.UserId)
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to send OTP"))
		return
	}

	response.OK(c, gin.H{
		"message":           "OTP sent to new mobile number",
		"maskedDestination": maskPhone(newPhone),
		"expiresInSeconds":  int(h.getOTPTTL().Seconds()),
	})
}

func (h *UsersHandler) VerifyMobileChange(c *gin.Context) {
	if h.UserMobileChangeStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "OTP service is not configured"))
		return
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	var req VerifyMobileChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	newPhone, err := normalizePhone(req.NewPhoneNumber)
	if err != nil {
		response.AbortWithError(c, errors.New(422, "VALIDATION_ERROR", "newPhoneNumber must be a valid phone number"))
		return
	}

	exists, err := h.UserStore.ExistsByPhoneForOther(c.Request.Context(), newPhone, claims.UserId)
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
		return
	}
	if exists {
		response.AbortWithError(c, errors.New(409, "CONFLICT", "Mobile number is already in use"))
		return
	}

	ok, err := h.UserMobileChangeStore.ConsumeValid(c.Request.Context(), claims.UserId, newPhone, hashOTP(req.OTPCode))
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
		return
	}
	if !ok {
		attempt, incErr := h.UserMobileChangeStore.IncrementAttempt(c.Request.Context(), claims.UserId, newPhone)
		if incErr != nil {
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
			return
		}
		if attempt == nil {
			response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "No active OTP found for this mobile number. Please request a new OTP"))
			return
		}
		if attempt.AttemptCount >= attempt.MaxAttempts {
			_ = h.UserMobileChangeStore.InvalidateActiveByPhone(c.Request.Context(), claims.UserId, newPhone)
			response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "OTP attempts exceeded. Please request a new OTP"))
			return
		}
		remaining := attempt.MaxAttempts - attempt.AttemptCount
		response.AbortWithError(c, errors.New(400, "BAD_REQUEST", fmt.Sprintf("Invalid or expired OTP. %d attempts remaining", remaining)))
		return
	}

	if err := h.UserStore.UpdatePhoneNumber(c.Request.Context(), claims.UserId, newPhone); err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update mobile number"))
		return
	}

	response.OK(c, gin.H{
		"message":     "Mobile number updated successfully",
		"phoneNumber": newPhone,
	})
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

func (h *UsersHandler) getOTPTTL() time.Duration {
	if h.OTPTTL <= 0 {
		return 5 * time.Minute
	}
	return h.OTPTTL
}

func (h *UsersHandler) getOTPResendCooldown() time.Duration {
	if h.OTPResendCooldown <= 0 {
		return 60 * time.Second
	}
	return h.OTPResendCooldown
}

func (h *UsersHandler) getOTPMaxAttempts() int {
	if h.OTPMaxAttempts <= 0 {
		return 5
	}
	return h.OTPMaxAttempts
}

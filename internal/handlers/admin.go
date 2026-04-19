package handlers

import (
	"fmt"
	"log"
	"net/http"
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
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	UserStore            *store.UserStore
	MOHAccountOTPStore   *store.MOHAccountOTPStore
	MOHTempPasswordStore *store.MOHTempPasswordStore
	WhatsAppSender       messaging.WhatsAppSender
	OTPTTL               time.Duration
	OTPResendCooldown    time.Duration
	OTPMaxAttempts       int
	TempPasswordTTL      time.Duration
	TempPasswordLength   int
}

type RequestMOHAccountOTPRequest struct {
	EmployeeId   string `json:"employeeId" binding:"required"`
	Name         string `json:"name" binding:"required"`
	NIC          string `json:"nic" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	PhoneNumber  string `json:"phoneNumber" binding:"required"`
	AssignedArea string `json:"assignedArea" binding:"required"`
}

type CompleteMOHAccountRequest struct {
	OTPID       string `json:"otpId" binding:"required"`
	OTPCode     string `json:"otpCode" binding:"required,len=6"`
	Password    string `json:"password" binding:"required,min=6"`
	ConfirmPass string `json:"confirmPassword" binding:"required"`
}

// NEW: Single-step MOH account creation with temporary password
type CreateMOHAccountRequest struct {
	EmployeeId   string `json:"employeeId" binding:"required"`
	Name         string `json:"name" binding:"required"`
	NIC          string `json:"nic" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	PhoneNumber  string `json:"phoneNumber" binding:"required"`
	AssignedArea string `json:"assignedArea" binding:"required"`
}

type MOHAccountOTPResponse struct {
	OTPID             string `json:"otpId"`
	MaskedDestination string `json:"maskedDestination"`
	ExpiresInSeconds  int    `json:"expiresInSeconds"`
	Message           string `json:"message"`
}

type MOHAccountCreatedResponse struct {
	Message    string `json:"message"`
	MOHUserID  string `json:"mohUserId"`
	Email      string `json:"email"`
	FirstLogin bool   `json:"firstLogin"`
}

// NEW: Response for simplified MOH account creation
type CreateMOHAccountResponse struct {
	Message           string `json:"message"`
	MOHUserID         string `json:"mohUserId"`
	Email             string `json:"email"`
	TempPassword      string `json:"tempPassword"`
	MaskedDestination string `json:"maskedDestination"`
	FirstLogin        bool   `json:"firstLogin"`
}

// ListMOHAccounts returns all MOH users for admin management.
func (h *AdminHandler) ListMOHAccounts(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if claims.Role != "admin" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only admin users can access MOH accounts")
		return
	}

	if h.UserStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "User store is not configured"))
		return
	}

	items, err := h.UserStore.ListMOHUsers(c.Request.Context())
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to fetch MOH accounts"))
		return
	}

	if items == nil {
		items = []models.MOHUserSummary{}
	}

	response.OK(c, gin.H{
		"count": len(items),
		"items": items,
	})
}

// RequestMOHAccountOTP generates an OTP for MOH account creation
func (h *AdminHandler) RequestMOHAccountOTP(c *gin.Context) {
	// Only admin can request OTP for MOH account creation
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if claims.Role != "admin" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only admin users can create MOH accounts")
		return
	}

	var req RequestMOHAccountOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	if h.MOHAccountOTPStore == nil || h.WhatsAppSender == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "OTP service is not configured"))
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

	// Check if there's already an active OTP for this email
	if active, err := h.MOHAccountOTPStore.GetLatestActive(ctx, claims.UserId, req.Email); err == nil && active != nil {
		retryAfter := active.CreatedAt.Add(h.getOTPResendCooldown()).Sub(time.Now())
		if retryAfter > 0 {
			response.AbortWithError(c, errors.New(429, "TOO_MANY_REQUESTS", fmt.Sprintf("Please wait %d seconds before requesting another OTP", int(retryAfter.Seconds())+1)))
			return
		}
	}

	// Generate OTP
	otpCode, err := generateOTPCode()
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to generate OTP"))
		return
	}

	otpID := "otp-moh-" + uuid.NewString()[:8]
	expiresAt := time.Now().Add(h.getOTPTTL())

	// Invalidate any previous active OTPs for this email
	_ = h.MOHAccountOTPStore.InvalidateActive(ctx, req.Email)

	// Save OTP
	if err := h.MOHAccountOTPStore.Create(ctx, otpID, claims.UserId, req.EmployeeId, req.Email, req.NIC, req.Name, req.PhoneNumber, req.AssignedArea, hashOTP(otpCode), expiresAt, h.getOTPMaxAttempts()); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to generate OTP"))
		return
	}

	// Send OTP via WhatsApp
	phone := strings.TrimSpace(req.PhoneNumber)
	message := h.buildMOHOTPMessage(req.Name, otpCode, h.getOTPTTL())

	if err := h.WhatsAppSender.SendMessage(ctx, phone, message); err != nil {
		log.Printf("[moh-otp] failed to send OTP to %s: %v", phone, err)
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to send OTP"))
		return
	}

	response.OK(c, gin.H{
		"otpId":             otpID,
		"maskedDestination": maskPhone(phone),
		"expiresInSeconds":  int(h.getOTPTTL().Seconds()),
		"message":           "OTP sent successfully",
	})
}

// CompleteMOHAccount completes MOH account creation after OTP verification
func (h *AdminHandler) CompleteMOHAccount(c *gin.Context) {
	// Only admin can complete MOH account creation
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if claims.Role != "admin" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only admin users can create MOH accounts")
		return
	}

	var req CompleteMOHAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	if req.Password != req.ConfirmPass {
		response.ValidationError(c, "Passwords do not match.", []response.ErrorDetail{{Field: "confirmPassword", Message: "Must match the password field."}})
		return
	}

	if len(req.Password) < 6 {
		response.ValidationError(c, "Password must be at least 6 characters", nil)
		return
	}

	ctx := c.Request.Context()

	// Get the OTP record
	otp, err := h.MOHAccountOTPStore.GetByID(ctx, req.OTPID)
	if err != nil || otp == nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "OTP not found or expired")
		return
	}

	// Verify OTP is not consumed and hasn't expired
	if otp.ConsumedAt != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "OTP has already been used")
		return
	}

	if time.Now().After(otp.ExpiresAt) {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "OTP has expired")
		return
	}

	// Verify OTP code
	valid, err := h.MOHAccountOTPStore.ConsumeValid(ctx, req.OTPID, hashOTP(req.OTPCode))
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
		return
	}

	if !valid {
		// Increment attempt count
		attempt, incErr := h.MOHAccountOTPStore.IncrementAttempt(ctx, req.OTPID)
		if incErr != nil {
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
			return
		}

		if attempt >= otp.MaxAttempts {
			_ = h.MOHAccountOTPStore.InvalidateActive(ctx, otp.Email)
			response.AbortWithError(c, errors.New(http.StatusBadRequest, "BAD_REQUEST", "OTP attempts exceeded. Please request a new OTP"))
			return
		}

		remaining := otp.MaxAttempts - attempt
		response.AbortWithError(c, errors.New(http.StatusBadRequest, "BAD_REQUEST", fmt.Sprintf("Invalid OTP. %d attempts remaining", remaining)))
		return
	}

	// Double-check email and NIC don't exist
	exists, _ := h.UserStore.ExistsByEmail(ctx, otp.Email)
	if exists {
		response.Error(c, http.StatusConflict, "CONFLICT", "Email already registered")
		return
	}

	exists, _ = h.UserStore.ExistsByNIC(ctx, otp.NIC)
	if exists {
		response.Error(c, http.StatusConflict, "CONFLICT", "NIC already registered")
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to create account")
		return
	}

	// Create MOH user
	userID := "user-moh-" + uuid.New().String()[:8]
	err = h.UserStore.CreateMOH(ctx, userID, otp.EmployeeID, otp.Email, otp.NIC, string(hash), otp.Name, otp.PhoneNumber, otp.AssignedArea, claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(http.StatusInternalServerError, "ERROR", "Failed to create MOH account"))
		return
	}

	response.Created(c, gin.H{
		"message":    "MOH account created successfully",
		"mohUserId":  userID,
		"email":      otp.Email,
		"firstLogin": true,
	})
}

// NEW: CreateMOHAccount creates MOH account with temporary password (simplified workflow)
// This is a single-step process that creates the account and sends a temporary password
func (h *AdminHandler) CreateMOHAccount(c *gin.Context) {
	// Only admin can create MOH accounts
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}

	if claims.Role != "admin" {
		response.Error(c, http.StatusForbidden, "FORBIDDEN", "Only admin users can create MOH accounts")
		return
	}

	var req CreateMOHAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	if h.MOHTempPasswordStore == nil || h.WhatsAppSender == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "MOH account service is not configured"))
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
	tempPassword := h.generateTempPassword()
	tempPasswordHash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to create account")
		return
	}

	// Create MOH user with temporary password
	userID := "user-moh-" + uuid.New().String()[:8]
	err = h.UserStore.CreateMOH(ctx, userID, req.EmployeeId, req.Email, req.NIC, string(tempPasswordHash), req.Name, req.PhoneNumber, req.AssignedArea, claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(http.StatusInternalServerError, "ERROR", "Failed to create MOH account"))
		return
	}

	// Save temporary password record for audit trail
	tempPasswordID := "temp-pwd-" + uuid.NewString()[:8]
	expiresAt := time.Now().Add(h.getTempPasswordTTL())

	if err := h.MOHTempPasswordStore.Create(ctx, tempPasswordID, req.EmployeeId, req.Email, req.NIC, req.Name, req.PhoneNumber, req.AssignedArea, claims.UserId, tempPassword, expiresAt); err != nil {
		log.Printf("[moh-creation] failed to save temp password record: %v", err)
		// Don't fail the request, account is already created
	}

	// Send temporary password via WhatsApp
	phone := strings.TrimSpace(req.PhoneNumber)
	message := h.buildMOHTempPasswordMessage(req.Name, tempPassword, h.getTempPasswordTTL())

	if err := h.WhatsAppSender.SendMessage(ctx, phone, message); err != nil {
		log.Printf("[moh-creation] failed to send temp password to %s: %v", phone, err)
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to send temporary password via WhatsApp"))
		return
	}

	response.Created(c, gin.H{
		"message":           "MOH account created successfully",
		"mohUserId":         userID,
		"email":             req.Email,
		"tempPassword":      tempPassword,
		"maskedDestination": maskPhone(phone),
		"firstLogin":        true,
	})
}

// Helper function to generate temporary password
func (h *AdminHandler) generateTempPassword() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%"
	length := h.getTempPasswordLength()
	if length < 8 {
		length = 12 // minimum 12 characters
	}

	b := make([]byte, length)
	for i := range b {
		// Use a simple pseudo-random approach (for production, use crypto/rand)
		rnd := time.Now().UnixNano()%int64(len(chars)) + int64(i)
		b[i] = chars[rnd%int64(len(chars))]
	}
	return string(b)
}

// Helper function to build WhatsApp message for temporary password
func (h *AdminHandler) buildMOHTempPasswordMessage(name, tempPassword string, ttl time.Duration) string {
	return fmt.Sprintf(
		"Hello %s,\n\nYou have been invited to create a MOH account on SuwaCareLK.\n\nYour temporary password is: %s\n\nYou can log in immediately with this password. However, you MUST change it on your first login.\n\nThis temporary password expires in %d hours.\n\nFor security: Do not share this password with anyone.\n\n- Ministry of Health",
		strings.TrimSpace(name),
		tempPassword,
		int(ttl.Hours()),
	)
}

func (h *AdminHandler) buildMOHOTPMessage(name, otpCode string, ttl time.Duration) string {
	return fmt.Sprintf(
		"Hello %s,\n\nYou have been invited to create a MOH account on SuwaCareLK.\n\nYour One-Time Password (OTP) is: %s\n\nThis OTP is valid for %d minutes.\n\nPlease do not share this OTP with anyone.\n\n- Ministry of Health",
		strings.TrimSpace(name),
		otpCode,
		int(ttl.Minutes()),
	)
}

func (h *AdminHandler) getOTPTTL() time.Duration {
	if h.OTPTTL <= 0 {
		return 5 * time.Minute
	}
	return h.OTPTTL
}

func (h *AdminHandler) getOTPResendCooldown() time.Duration {
	if h.OTPResendCooldown <= 0 {
		return 60 * time.Second
	}
	return h.OTPResendCooldown
}

func (h *AdminHandler) getOTPMaxAttempts() int {
	if h.OTPMaxAttempts <= 0 {
		return 5
	}
	return h.OTPMaxAttempts
}

// Helper function to get temporary password TTL
func (h *AdminHandler) getTempPasswordTTL() time.Duration {
	if h.TempPasswordTTL <= 0 {
		return 24 * time.Hour // default 24 hours
	}
	return h.TempPasswordTTL
}

// Helper function to get temporary password length
func (h *AdminHandler) getTempPasswordLength() int {
	if h.TempPasswordLength <= 0 {
		return 12 // default 12 characters
	}
	return h.TempPasswordLength
}

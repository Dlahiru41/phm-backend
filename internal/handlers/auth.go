package handlers

import (
	"net/http"
	"strings"
	"time"

	"ncvms/internal/auth"
	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/models"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserStore *store.UserStore
	AuditStore *store.AuditStore
	JWTSecret string
	JWTExpiry int
}

type LoginRequest struct {
	UsernameOrEmail string `json:"usernameOrEmail" binding:"required"`
	Password       string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	FullName        string `json:"fullName" binding:"required"`
	NIC             string `json:"nic" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	PhoneNumber     string `json:"phoneNumber"`
	Password        string `json:"password" binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
	Role            string `json:"role" binding:"required,oneof=parent phm moh"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	req.UsernameOrEmail = strings.TrimSpace(req.UsernameOrEmail)
	req.Password = strings.TrimSpace(req.Password)
	if req.UsernameOrEmail == "" || req.Password == "" {
		response.ValidationError(c, "usernameOrEmail and password are required", nil)
		return
	}

	var user *models.UserWithPassword
	var err error
	if strings.Contains(req.UsernameOrEmail, "@") {
		user, err = h.UserStore.GetByEmail(c.Request.Context(), req.UsernameOrEmail)
	} else {
		user, err = h.UserStore.GetByNIC(c.Request.Context(), req.UsernameOrEmail)
	}
	if err != nil || user == nil {
		response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		response.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials")
		return
	}

	token, err := auth.NewToken(user.UserId, user.Role, user.Email, h.JWTSecret, h.JWTExpiry)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	if h.AuditStore != nil {
		auditID := "log-" + uuid.New().String()[:8]
		_ = h.AuditStore.Insert(c.Request.Context(), auditID, &user.UserId, user.Role, user.Name, "LOGIN", "User", user.UserId, "Login", c.ClientIP())
	}

	response.OK(c, gin.H{
		"token": token,
		"user":  userToResponse(user),
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	if req.Password != req.ConfirmPassword {
		response.ValidationError(c, "Passwords do not match.", []response.ErrorDetail{{Field: "confirmPassword", Message: "Must match the password field."}})
		return
	}
	if len(req.Password) < 6 {
		response.ValidationError(c, "Password must be at least 6 characters", nil)
		return
	}

	ctx := c.Request.Context()
	exists, _ := h.UserStore.ExistsByEmail(ctx, req.Email)
	if exists {
		response.Error(c, http.StatusConflict, "CONFLICT", "Email or NIC already registered")
		return
	}
	exists, _ = h.UserStore.ExistsByNIC(ctx, req.NIC)
	if exists {
		response.Error(c, http.StatusConflict, "CONFLICT", "Email or NIC already registered")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to hash password")
		return
	}
	userID := "user-" + req.Role + "-" + uuid.New().String()[:8]
	err = h.UserStore.Create(ctx, userID, req.Email, req.NIC, string(hash), req.Role, req.FullName, req.PhoneNumber, "")
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(http.StatusInternalServerError, "ERROR", "Failed to create account"))
		return
	}

	response.Created(c, gin.H{"message": "Account created successfully.", "userId": userID})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims != nil && h.AuditStore != nil {
		auditID := "log-" + uuid.New().String()[:8]
		_ = h.AuditStore.Insert(c.Request.Context(), auditID, &claims.UserId, claims.Role, claims.Email, "LOGOUT", "User", claims.UserId, "Logout", c.ClientIP())
	}
	response.OK(c, gin.H{"message": "Logged out successfully."})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	user, err := h.UserStore.GetByEmail(c.Request.Context(), req.Email)
	if err != nil || user == nil {
		response.OK(c, gin.H{"message": "Password reset email sent."})
		return
	}
	token := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)
	_ = h.UserStore.SavePasswordResetToken(c.Request.Context(), token, user.UserId, expiresAt)
	// TODO: send email with reset link containing token
	response.OK(c, gin.H{"message": "Password reset email sent."})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		response.ValidationError(c, "Passwords do not match.", nil)
		return
	}
	userID, err := h.UserStore.ConsumeResetToken(c.Request.Context(), req.Token)
	if err != nil {
		response.AbortWithError(c, errors.New(http.StatusUnprocessableEntity, "INVALID_TOKEN", "Invalid or expired reset token"))
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to update password")
		return
	}
	err = h.UserStore.UpdatePassword(c.Request.Context(), userID, string(hash))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ERROR", "Failed to update password")
		return
	}
	response.OK(c, gin.H{"message": "Password reset successfully."})
}

func userToResponse(u *models.UserWithPassword) gin.H {
	return gin.H{
		"userId":             u.UserId,
		"email":              u.Email,
		"nic":                u.NIC,
		"role":               u.Role,
		"name":               u.Name,
		"phoneNumber":        u.PhoneNumber,
		"address":            u.Address,
		"languagePreference": u.LanguagePreference,
		"createdAt":          u.CreatedAt,
	}
}

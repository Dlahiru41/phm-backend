package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
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

type ChildrenHandler struct {
	ChildStore        *store.ChildStore
	UserStore         *store.UserStore
	ChildLinkOTPStore *store.ChildLinkOTPStore
	WhatsAppSender    messaging.WhatsAppSender
	OTPTTL            time.Duration
	OTPResendCooldown time.Duration
	OTPMaxAttempts    int
}

type RegisterChildRequest struct {
	FirstName            string   `json:"firstName" binding:"required"`
	LastName             string   `json:"lastName" binding:"required"`
	DateOfBirth          string   `json:"dateOfBirth" binding:"required"`
	Gender               string   `json:"gender" binding:"required,oneof=male female other"`
	BirthWeight          *float64 `json:"birthWeight"`
	BirthHeight          *float64 `json:"birthHeight"`
	HeadCircumference    *float64 `json:"headCircumference"`
	BloodGroup           string   `json:"bloodGroup"`
	MotherName           string   `json:"motherName"`
	MotherNIC            string   `json:"motherNIC"`
	FatherName           string   `json:"fatherName"`
	FatherNIC            string   `json:"fatherNIC"`
	District             string   `json:"district"`
	DsDivision           string   `json:"dsDivision"`
	GnDivision           string   `json:"gnDivision"`
	Address              string   `json:"address"`
	PhmId                string   `json:"phmId"`
	AreaCode             string   `json:"areaCode"`
	ParentWhatsAppNumber string   `json:"parentWhatsAppNumber" binding:"required"`
}

type LinkParentRequest struct {
	RegistrationNumber string `json:"registrationNumber" binding:"required"`
	OTPCode            string `json:"otpCode" binding:"required,len=6"`
}

type RequestLinkOTPRequest struct {
	RegistrationNumber string `json:"registrationNumber" binding:"required"`
}

func (h *ChildrenHandler) Register(c *gin.Context) {
	var req RegisterChildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	claims := middleware.GetClaims(c)

	parentWhatsAppNumber, err := normalizePhone(req.ParentWhatsAppNumber)
	if err != nil {
		response.AbortWithError(c, errors.New(422, "VALIDATION_ERROR", "parentWhatsAppNumber must be a valid phone number"))
		return
	}

	// Determine which PHM ID to associate — explicit request value takes precedence
	registeredBy := claims.UserId
	if req.PhmId != "" {
		registeredBy = req.PhmId
	}

	// Determine areaCode — explicit request value takes precedence, then fall back to PHM's DB record
	areaCode := req.AreaCode
	if areaCode == "" && h.UserStore != nil {
		phmUser, err := h.UserStore.GetByID(c.Request.Context(), registeredBy)
		if err == nil && phmUser != nil && phmUser.AreaCode != nil {
			areaCode = *phmUser.AreaCode
		}
	}

	// Build registration number: NCVMS-YYYY-MMDD-xxxx
	// DateOfBirth is expected as YYYY-MM-DD
	regSuffix := uuid.New().String()[:4]
	var regNum string
	if len(req.DateOfBirth) >= 10 {
		regNum = fmt.Sprintf("NCVMS-%s-%s-%s", req.DateOfBirth[:4], req.DateOfBirth[5:7]+req.DateOfBirth[8:10], regSuffix)
	} else {
		regNum = fmt.Sprintf("NCVMS-%s-%s", req.DateOfBirth[:4], regSuffix)
	}

	childID := "child-" + uuid.New().String()[:8]
	err = h.ChildStore.Create(c.Request.Context(), childID, regNum, req.FirstName, req.LastName, req.DateOfBirth, req.Gender, req.BloodGroup,
		req.BirthWeight, req.BirthHeight, req.HeadCircumference, req.MotherName, req.MotherNIC, req.FatherName, req.FatherNIC,
		registeredBy, req.District, req.DsDivision, req.GnDivision, req.Address, areaCode, parentWhatsAppNumber)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to register child"))
		return
	}
	response.Created(c, gin.H{"childId": childID, "registrationNumber": regNum, "message": "Child registered successfully."})
}

func (h *ChildrenHandler) ListMy(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	if claims.Role != "phm" {
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}

	_, pageProvided := c.GetQuery("page")
	_, limitProvided := c.GetQuery("limit")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Pass page/limit=0 to signal "no pagination" (return all)
	fetchPage, fetchLimit := page, limit
	if !pageProvided && !limitProvided {
		fetchPage, fetchLimit = 0, 0
	}

	total, list, err := h.ChildStore.ByRegisteredBy(c.Request.Context(), claims.UserId, fetchPage, fetchLimit)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list children"))
		return
	}
	for i := range list {
		list[i].VaccinationStatus = "on-track"
	}

	if pageProvided || limitProvided {
		response.OK(c, gin.H{"total": total, "page": page, "limit": limit, "data": list})
	} else {
		response.OK(c, list)
	}
}

func (h *ChildrenHandler) GetByID(c *gin.Context) {
	childID := c.Param("childId")
	child, err := h.ChildStore.GetByID(c.Request.Context(), childID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	response.OK(c, childToDetail(child))
}

func (h *ChildrenHandler) Search(c *gin.Context) {
	regNum := c.Query("registrationNumber")
	if regNum == "" {
		response.ValidationError(c, "registrationNumber is required", nil)
		return
	}
	child, err := h.ChildStore.GetByRegistrationNumber(c.Request.Context(), regNum)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	response.OK(c, childToSummary(child))
}

func (h *ChildrenHandler) RequestLinkOTP(c *gin.Context) {
	if h.ChildLinkOTPStore == nil || h.WhatsAppSender == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "OTP service is not configured"))
		return
	}

	var req RequestLinkOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	claims := middleware.GetClaims(c)
	childID := c.Param("childId")
	linkInfo, err := h.ChildStore.GetLinkInfo(c.Request.Context(), childID, req.RegistrationNumber)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	if linkInfo.ParentID != nil {
		response.AbortWithError(c, errors.New(409, "CONFLICT", "Child is already linked to a parent account"))
		return
	}
	if linkInfo.ParentWhatsAppNumber == "" {
		response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "Child record does not have a parent WhatsApp number"))
		return
	}

	if active, err := h.ChildLinkOTPStore.GetLatestActive(c.Request.Context(), childID, claims.UserId); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process OTP request"))
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
	otpHash := hashOTP(otpCode)
	otpID := "otp-" + uuid.NewString()[:8]
	expiresAt := time.Now().Add(h.getOTPTTL())

	if err := h.ChildLinkOTPStore.InvalidateActive(c.Request.Context(), childID, claims.UserId); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process OTP request"))
		return
	}
	if err := h.ChildLinkOTPStore.Create(c.Request.Context(), otpID, childID, claims.UserId, otpHash, expiresAt, h.getOTPMaxAttempts()); err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to process OTP request"))
		return
	}

	if err := h.WhatsAppSender.SendOTP(c.Request.Context(), linkInfo.ParentWhatsAppNumber, otpCode, h.getOTPTTL()); err != nil {
		_ = h.ChildLinkOTPStore.InvalidateActive(c.Request.Context(), childID, claims.UserId)
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to send OTP"))
		return
	}

	response.OK(c, gin.H{
		"message":           "OTP sent to parent WhatsApp number",
		"maskedDestination": maskPhone(linkInfo.ParentWhatsAppNumber),
		"expiresInSeconds":  int(h.getOTPTTL().Seconds()),
	})
}

func (h *ChildrenHandler) LinkParent(c *gin.Context) {
	if h.ChildLinkOTPStore == nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "OTP service is not configured"))
		return
	}

	childID := c.Param("childId")
	var req LinkParentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}

	claims := middleware.GetClaims(c)
	linkInfo, err := h.ChildStore.GetLinkInfo(c.Request.Context(), childID, req.RegistrationNumber)
	if err != nil || linkInfo == nil {
		if err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	if linkInfo.ParentID != nil {
		response.AbortWithError(c, errors.New(409, "CONFLICT", "Child is already linked to a parent account"))
		return
	}

	ok, err := h.ChildLinkOTPStore.ConsumeValid(c.Request.Context(), childID, claims.UserId, hashOTP(req.OTPCode))
	if err != nil {
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
		return
	}
	if !ok {
		attempt, incErr := h.ChildLinkOTPStore.IncrementAttempt(c.Request.Context(), childID, claims.UserId)
		if incErr != nil {
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to verify OTP"))
			return
		}
		if attempt == nil {
			response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "No active OTP found. Please request a new OTP"))
			return
		}
		if attempt.AttemptCount >= attempt.MaxAttempts {
			_ = h.ChildLinkOTPStore.InvalidateActive(c.Request.Context(), childID, claims.UserId)
			response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "OTP attempts exceeded. Please request a new OTP"))
			return
		}
		remaining := attempt.MaxAttempts - attempt.AttemptCount
		response.AbortWithError(c, errors.New(400, "BAD_REQUEST", fmt.Sprintf("Invalid or expired OTP. %d attempts remaining", remaining)))
		return
	}

	err = h.ChildStore.LinkParent(c.Request.Context(), childID, claims.UserId)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to link child"))
		return
	}
	response.OK(c, gin.H{"message": "Child successfully linked to your account."})
}

func (h *ChildrenHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	parentID := c.Query("parentId")
	phmID := c.Query("phmId")
	registeredBy := c.Query("registeredBy")

	if parentID != "" {
		if claims.Role != "parent" && claims.Role != "phm" && claims.Role != "moh" {
			response.AbortWithError(c, errors.ErrForbidden)
			return
		}
		if claims.Role == "parent" && parentID != claims.UserId {
			response.AbortWithError(c, errors.ErrForbidden)
			return
		}
		list, err := h.ChildStore.ByParentID(c.Request.Context(), parentID)
		if err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list children"))
			return
		}
		for i := range list {
			list[i].VaccinationStatus = "on-track"
		}
		response.OK(c, list)
		return
	}

	if registeredBy != "" {
		if claims.Role != "phm" && claims.Role != "moh" {
			response.AbortWithError(c, errors.ErrForbidden)
			return
		}
		if claims.Role == "phm" && registeredBy != claims.UserId {
			response.AbortWithError(c, errors.ErrForbidden)
			return
		}

		_, pageProvided := c.GetQuery("page")
		_, limitProvided := c.GetQuery("limit")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 10
		}

		fetchPage, fetchLimit := page, limit
		if !pageProvided && !limitProvided {
			fetchPage, fetchLimit = 0, 0
		}

		total, list, err := h.ChildStore.ByRegisteredBy(c.Request.Context(), registeredBy, fetchPage, fetchLimit)
		if err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list children"))
			return
		}
		for i := range list {
			list[i].VaccinationStatus = "on-track"
		}

		if pageProvided || limitProvided {
			response.OK(c, gin.H{"total": total, "page": page, "limit": limit, "data": list})
		} else {
			response.OK(c, list)
		}
		return
	}

	if phmID != "" {
		if claims.Role != "phm" && claims.Role != "moh" {
			response.AbortWithError(c, errors.ErrForbidden)
			return
		}
		if claims.Role == "phm" && phmID != claims.UserId {
			response.AbortWithError(c, errors.ErrForbidden)
			return
		}

		_, pageProvided := c.GetQuery("page")
		_, limitProvided := c.GetQuery("limit")
		if pageProvided || limitProvided {
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if page < 1 {
				page = 1
			}
			if limit < 1 || limit > 100 {
				limit = 10
			}
			total, list, err := h.ChildStore.ByPHMIDPaginated(c.Request.Context(), phmID, page, limit)
			if err != nil {
				if appErr := errors.FromErr(err); appErr != nil {
					response.AbortWithError(c, appErr)
					return
				}
				response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list children"))
				return
			}
			for i := range list {
				list[i].VaccinationStatus = "on-track"
			}
			response.OK(c, gin.H{"total": total, "page": page, "limit": limit, "data": list})
			return
		}

		list, err := h.ChildStore.ByPHMID(c.Request.Context(), phmID)
		if err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
			response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list children"))
			return
		}
		for i := range list {
			list[i].VaccinationStatus = "on-track"
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
	if limit < 1 || limit > 100 {
		limit = 20
	}
	areaCode := c.Query("areaCode")
	status := c.Query("status")
	search := c.Query("search")
	total, list, err := h.ChildStore.ListMOH(c.Request.Context(), areaCode, status, search, page, limit)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list children"))
		return
	}
	for i := range list {
		list[i].VaccinationStatus = "on-track"
	}
	response.OK(c, gin.H{"total": total, "page": page, "limit": limit, "data": list})
}

func (h *ChildrenHandler) Update(c *gin.Context) {
	childID := c.Param("childId")
	var req struct {
		FirstName  *string `json:"firstName"`
		LastName   *string `json:"lastName"`
		BloodGroup *string `json:"bloodGroup"`
		Address    *string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	err := h.ChildStore.Update(c.Request.Context(), childID, req.FirstName, req.LastName, req.BloodGroup, req.Address)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to update child"))
		return
	}
	response.OK(c, gin.H{"message": "Child profile updated successfully."})
}

func childToDetail(c *models.ChildDetail) gin.H {
	return gin.H{
		"childId":            c.ChildId,
		"registrationNumber": c.RegistrationNumber,
		"firstName":          c.FirstName,
		"lastName":           c.LastName,
		"dateOfBirth":        c.DateOfBirth,
		"gender":             c.Gender,
		"bloodGroup":         c.BloodGroup,
		"birthWeight":        c.BirthWeight,
		"birthHeight":        c.BirthHeight,
		"headCircumference":  c.HeadCircumference,
		"parentId":           c.ParentId,
		"registeredBy":       c.RegisteredBy,
		"areaCode":           c.AreaCode,
		"areaName":           c.AreaName,
		"createdAt":          c.CreatedAt,
		"motherName":         c.MotherName,
		"motherNIC":          c.MotherNIC,
		"fatherName":         c.FatherName,
		"fatherNIC":          c.FatherNIC,
		"district":           c.District,
		"dsDivision":         c.DsDivision,
		"gnDivision":         c.GnDivision,
		"address":            c.Address,
	}
}

func childToSummary(c *models.Child) gin.H {
	return gin.H{
		"childId":            c.ChildId,
		"registrationNumber": c.RegistrationNumber,
		"firstName":          c.FirstName,
		"lastName":           c.LastName,
		"dateOfBirth":        c.DateOfBirth,
		"gender":             c.Gender,
	}
}

func hashOTP(otp string) string {
	sum := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(sum[:])
}

func generateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func normalizePhone(value string) (string, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "", fmt.Errorf("empty")
	}
	if strings.HasPrefix(clean, "+") {
		clean = "+" + strings.ReplaceAll(clean[1:], " ", "")
	} else {
		clean = strings.ReplaceAll(clean, " ", "")
	}
	if len(clean) < 10 || len(clean) > 16 {
		return "", fmt.Errorf("invalid length")
	}
	for i, ch := range clean {
		if i == 0 && ch == '+' {
			continue
		}
		if ch < '0' || ch > '9' {
			return "", fmt.Errorf("invalid chars")
		}
	}
	if clean[0] != '+' {
		clean = "+" + clean
	}
	return clean, nil
}

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return phone
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

func (h *ChildrenHandler) getOTPTTL() time.Duration {
	if h.OTPTTL <= 0 {
		return 5 * time.Minute
	}
	return h.OTPTTL
}

func (h *ChildrenHandler) getOTPResendCooldown() time.Duration {
	if h.OTPResendCooldown <= 0 {
		return 60 * time.Second
	}
	return h.OTPResendCooldown
}

func (h *ChildrenHandler) getOTPMaxAttempts() int {
	if h.OTPMaxAttempts <= 0 {
		return 5
	}
	return h.OTPMaxAttempts
}

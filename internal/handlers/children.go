package handlers

import (
	"fmt"
	"strconv"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/models"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChildrenHandler struct {
	ChildStore *store.ChildStore
	UserStore  *store.UserStore
}

type RegisterChildRequest struct {
	FirstName         string   `json:"firstName" binding:"required"`
	LastName          string   `json:"lastName" binding:"required"`
	DateOfBirth       string   `json:"dateOfBirth" binding:"required"`
	Gender            string   `json:"gender" binding:"required,oneof=male female other"`
	BirthWeight       *float64 `json:"birthWeight"`
	BirthHeight       *float64 `json:"birthHeight"`
	HeadCircumference *float64 `json:"headCircumference"`
	BloodGroup        string   `json:"bloodGroup"`
	MotherName        string   `json:"motherName"`
	MotherNIC         string   `json:"motherNIC"`
	FatherName        string   `json:"fatherName"`
	FatherNIC         string   `json:"fatherNIC"`
	District          string   `json:"district"`
	DsDivision        string   `json:"dsDivision"`
	GnDivision        string   `json:"gnDivision"`
	Address           string   `json:"address"`
	PhmId             string   `json:"phmId"`
	AreaCode          string   `json:"areaCode"`
}

type LinkParentRequest struct {
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
	err := h.ChildStore.Create(c.Request.Context(), childID, regNum, req.FirstName, req.LastName, req.DateOfBirth, req.Gender, req.BloodGroup,
		req.BirthWeight, req.BirthHeight, req.HeadCircumference, req.MotherName, req.MotherNIC, req.FatherName, req.FatherNIC,
		registeredBy, req.District, req.DsDivision, req.GnDivision, req.Address, areaCode)
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

func (h *ChildrenHandler) LinkParent(c *gin.Context) {
	childID := c.Param("childId")
	var req LinkParentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if response.ValidationErrorFromBind(c, err) {
			return
		}
		response.ValidationError(c, "Validation failed", nil)
		return
	}
	child, err := h.ChildStore.GetByRegistrationNumber(c.Request.Context(), req.RegistrationNumber)
	if err != nil || child == nil {
		if err != nil {
			if appErr := errors.FromErr(err); appErr != nil {
				response.AbortWithError(c, appErr)
				return
			}
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	if child.ChildId != childID {
		response.AbortWithError(c, errors.New(400, "BAD_REQUEST", "Child ID does not match registration number"))
		return
	}
	claims := middleware.GetClaims(c)
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

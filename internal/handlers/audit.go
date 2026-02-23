package handlers

import (
	"strconv"

	"ncvms/internal/errors"
	"ncvms/internal/middleware"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	AuditStore *store.AuditStore
}

func (h *AuditHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 50
	}
	total, list, err := h.AuditStore.List(c.Request.Context(),
		c.Query("userId"), c.Query("userRole"), c.Query("entityType"), c.Query("action"),
		c.Query("startDate"), c.Query("endDate"), c.Query("search"), page, limit)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list audit logs"))
		return
	}
	response.OK(c, gin.H{"total": total, "page": page, "limit": limit, "data": list})
}

func (h *AuditHandler) Export(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.AbortWithError(c, errors.ErrUnauthorized)
		return
	}
	format := c.DefaultQuery("format", "csv")
	_, list, err := h.AuditStore.List(c.Request.Context(),
		c.Query("userId"), c.Query("userRole"), c.Query("entityType"), c.Query("action"),
		c.Query("startDate"), c.Query("endDate"), c.Query("search"), 1, 10000)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to export"))
		return
	}
	if format == "csv" {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=audit-logs.csv")
	}
	response.OK(c, gin.H{"format": format, "count": len(list), "data": list})
}

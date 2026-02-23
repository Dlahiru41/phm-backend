package handlers

import (
	"ncvms/internal/errors"
	"ncvms/internal/response"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
)

type VaccinesHandler struct {
	VaccineStore *store.VaccineStore
}

func (h *VaccinesHandler) List(c *gin.Context) {
	list, err := h.VaccineStore.ListActive(c.Request.Context())
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.New(errors.ErrInternal.Status, "ERROR", "Failed to list vaccines"))
		return
	}
	response.OK(c, list)
}

func (h *VaccinesHandler) GetByID(c *gin.Context) {
	vaccineID := c.Param("vaccineId")
	v, err := h.VaccineStore.GetByID(c.Request.Context(), vaccineID)
	if err != nil {
		if appErr := errors.FromErr(err); appErr != nil {
			response.AbortWithError(c, appErr)
			return
		}
		response.AbortWithError(c, errors.ErrNotFound)
		return
	}
	response.OK(c, v)
}

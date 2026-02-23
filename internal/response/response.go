package response

import (
	"errors"
	"net/http"

	apperr "ncvms/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Error struct {
		Code    string        `json:"code"`
		Message string        `json:"message"`
		Details []ErrorDetail `json:"details,omitempty"`
	} `json:"error"`
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorBody{
		Error: struct {
			Code    string        `json:"code"`
			Message string        `json:"message"`
			Details []ErrorDetail `json:"details,omitempty"`
		}{Code: code, Message: message},
	})
}

func ValidationError(c *gin.Context, message string, details []ErrorDetail) {
	c.JSON(http.StatusUnprocessableEntity, ErrorBody{
		Error: struct {
			Code    string        `json:"code"`
			Message string        `json:"message"`
			Details []ErrorDetail `json:"details,omitempty"`
		}{Code: "VALIDATION_ERROR", Message: message, Details: details},
	})
}

func OK(c *gin.Context, data interface{}) {
	if data == nil {
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// AbortWithError sends the AppError as JSON and aborts the context. Use after store/validation errors.
func AbortWithError(c *gin.Context, appErr *apperr.AppError) {
	if appErr == nil {
		return
	}
	details := make([]ErrorDetail, 0, len(appErr.Details))
	for _, d := range appErr.Details {
		details = append(details, ErrorDetail{Field: d.Field, Message: d.Message})
	}
	c.Abort()
	c.JSON(appErr.Status, ErrorBody{
		Error: struct {
			Code    string        `json:"code"`
			Message string        `json:"message"`
			Details []ErrorDetail `json:"details,omitempty"`
		}{Code: appErr.Code, Message: appErr.Message, Details: details},
	})
}

// ValidationErrorFromBind handles binding/validation errors from ShouldBindJSON etc.
// Returns true if it wrote a response (caller should return); false if err is nil.
func ValidationErrorFromBind(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var verr validator.ValidationErrors
	if !errors.As(err, &verr) {
		ValidationError(c, "Validation failed", nil)
		return true
	}
	details := make([]ErrorDetail, 0, len(verr))
	for _, e := range verr {
		details = append(details, ErrorDetail{Field: e.Field(), Message: e.Error()})
	}
	ValidationError(c, "Validation failed", details)
	return true
}

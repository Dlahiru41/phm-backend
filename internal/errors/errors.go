package errors

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AppError carries HTTP status and API error code for consistent responses.
type AppError struct {
	Status  int
	Code    string
	Message string
	Details []Detail
}

type Detail struct {
	Field   string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

// New returns an AppError with the given status, code, and message.
func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// NewValidation returns 422 with VALIDATION_ERROR and optional field details.
func NewValidation(message string, details []Detail) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Details: details,
	}
}

// FromErr maps known errors (e.g. DB) to an AppError. Returns nil if err is nil.
// - pgx.ErrNoRows -> 404 NOT_FOUND
// - pgconn unique violation (23505) / foreign key (23503) -> 409 CONFLICT or 400 BAD_REQUEST
// - other -> 500 INTERNAL_ERROR (generic message; caller may replace)
func FromErr(err error) *AppError {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return New(http.StatusNotFound, "NOT_FOUND", "Resource not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return New(http.StatusConflict, "CONFLICT", "A record with this value already exists.")
		case "23503": // foreign_key_violation
			return New(http.StatusBadRequest, "BAD_REQUEST", "Referenced resource does not exist.")
		case "23502": // not_null_violation
			return New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Required field is missing.")
		case "23514": // check_violation
			return New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid value for one or more fields.")
		default:
			return New(http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
	}
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
}

// Predefined helpers for common status codes (handlers can use these or New).
var (
	ErrUnauthorized   = New(http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid authorization.")
	ErrForbidden      = New(http.StatusForbidden, "FORBIDDEN", "Insufficient permissions.")
	ErrNotFound       = New(http.StatusNotFound, "NOT_FOUND", "Resource not found.")
	ErrBadRequest     = New(http.StatusBadRequest, "BAD_REQUEST", "Invalid request.")
	ErrConflict       = New(http.StatusConflict, "CONFLICT", "Resource already exists or conflict.")
	ErrInternal       = New(http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
)

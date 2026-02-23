package middleware

import (
	"log"
	"net/http"

	"ncvms/internal/response"

	"github.com/gin-gonic/gin"
)

// Recovery recovers from panics, logs the stack, and returns 500 with a standard error body.
// Use as the first middleware so it wraps all handlers.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %v", rec)
				// Optionally log stack: debug.PrintStack() or log stack trace
				c.Abort()
				response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
			}
		}()
		c.Next()
	}
}

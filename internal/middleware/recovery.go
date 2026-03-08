package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"ncvms/internal/response"

	"github.com/gin-gonic/gin"
)

// Recovery recovers from panics, logs the stack, and returns 500 with a standard error body.
// Use as the first middleware so it wraps all handlers.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] method=%s path=%s panic=%v\n%s", c.Request.Method, c.Request.URL.Path, rec, debug.Stack())
				c.Abort()
				response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
			}
		}()

		c.Next()
	}
}

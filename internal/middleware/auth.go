package middleware

import (
	"strings"

	"ncvms/internal/auth"
	"ncvms/internal/errors"
	"ncvms/internal/response"

	"github.com/gin-gonic/gin"
)

const UserClaimsKey = "userClaims"

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.AbortWithError(c, errors.New(errors.ErrUnauthorized.Status, "UNAUTHORIZED", "Missing or invalid authorization header"))
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.AbortWithError(c, errors.New(errors.ErrUnauthorized.Status, "UNAUTHORIZED", "Invalid authorization format"))
			return
		}
		claims, err := auth.ParseToken(parts[1], jwtSecret)
		if err != nil {
			response.AbortWithError(c, errors.ErrUnauthorized)
			return
		}
		c.Set(UserClaimsKey, claims)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(UserClaimsKey)
		if !exists {
			response.AbortWithError(c, errors.ErrUnauthorized)
			return
		}
		claims := val.(*auth.Claims)
		for _, r := range roles {
			if claims.Role == r {
				c.Next()
				return
			}
		}
		response.AbortWithError(c, errors.ErrForbidden)
		return
	}
}

func GetClaims(c *gin.Context) *auth.Claims {
	val, _ := c.Get(UserClaimsKey)
	if val == nil {
		return nil
	}
	return val.(*auth.Claims)
}

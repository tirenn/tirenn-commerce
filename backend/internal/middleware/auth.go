package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tirenn/commerce/backend/internal/config"
	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/domain/auth"
	"github.com/tirenn/commerce/backend/internal/response"
	"github.com/tirenn/commerce/backend/internal/security"
)

// JWTAuth verifies the Bearer token in the Authorization header
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, "Authorization header is missing", domain.ErrUnauthorized)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Error(c, "Invalid Authorization header format. Expected 'Bearer <token>'", domain.ErrUnauthorized)
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := security.ValidateJWT(tokenStr, cfg.JWTSecret)
		if err != nil {
			response.Error(c, err.Error(), domain.ErrUnauthorized)
			c.Abort()
			return
		}

		// Set context variables
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		c.Set("userName", claims.Name)

		c.Next()
	}
}

// RequireRole enforces specific RBAC roles (e.g. ADMIN)
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("userRole")
		if !exists {
			response.Error(c, "Authentication required", domain.ErrUnauthorized)
			c.Abort()
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			response.Error(c, "Invalid role claims", domain.ErrForbidden)
			c.Abort()
			return
		}

		allowed := false
		for _, r := range allowedRoles {
			if strings.EqualFold(userRole, r) {
				allowed = true
				break
			}
		}

		if !allowed {
			response.Error(c, "Access denied: insufficient permissions", domain.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin is a shortcut middleware for admin routes
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(auth.RoleAdmin)
}

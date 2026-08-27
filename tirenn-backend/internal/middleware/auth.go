package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/config"
	"gocommerce-backend/internal/domain/auth"
	"gocommerce-backend/internal/utils"
)

// JWTAuth verifies the Bearer token in the Authorization header
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Authorization header is missing")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Unauthorized(c, "Invalid Authorization header format. Expected 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := utils.ValidateJWT(tokenStr, cfg.JWTSecret)
		if err != nil {
			utils.Unauthorized(c, err.Error())
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
			utils.Unauthorized(c, "Authentication required")
			c.Abort()
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			utils.Forbidden(c, "Invalid role claims")
			c.Abort()
			return
		}

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		utils.Forbidden(c, "Access denied: insufficient permissions for this resource")
		c.Abort()
	}
}

// OptionalJWT parses JWT if present, but does not abort if not present
func OptionalJWT(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				if claims, err := utils.ValidateJWT(parts[1], cfg.JWTSecret); err == nil {
					c.Set("userID", claims.UserID)
					c.Set("userEmail", claims.Email)
					c.Set("userRole", claims.Role)
					c.Set("userName", claims.Name)
				}
			}
		}
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole(auth.RoleAdmin)
}

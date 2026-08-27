package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/logger"
)

// generateRequestID produces a unique tracing ID
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req-%s", hex.EncodeToString(bytes))
}

// RequestID middleware injects a unique X-Request-ID into every HTTP request context and response header
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}

		// Set in Gin context
		c.Set(string(logger.RequestIDKey), reqID)
		c.Set("request_id", reqID)

		// Set in standard request context
		ctx := logger.WithRequestID(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)

		// Return header to client
		c.Writer.Header().Set("X-Request-ID", reqID)

		c.Next()
	}
}

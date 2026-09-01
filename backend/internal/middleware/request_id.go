package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tirenn/commerce/backend/internal/logger"
)

// generateRandomHex produces random hex string
func generateRandomHex(byteCount int) string {
	bytes := make([]byte, byteCount)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// RequestID middleware injects Distributed Tracing IDs (X-Trace-ID, X-Span-ID, X-Request-ID) into every HTTP context and response header
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 1. Trace ID
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = c.GetHeader("traceparent")
		}
		if traceID == "" {
			traceID = fmt.Sprintf("trace-%s", generateRandomHex(8))
		}

		// 2. Request ID
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("req-%s", generateRandomHex(6))
		}

		// 3. Span ID
		spanID := fmt.Sprintf("span-%s", generateRandomHex(4))

		// Set in Gin context
		c.Set(string(logger.RequestIDKey), reqID)
		c.Set("request_id", reqID)
		c.Set(string(logger.TraceIDKey), traceID)
		c.Set("trace_id", traceID)
		c.Set(string(logger.SpanIDKey), spanID)
		c.Set("span_id", spanID)

		// Set in standard request context
		ctx := logger.WithRequestID(c.Request.Context(), reqID)
		ctx = logger.WithTraceID(ctx, traceID)
		ctx = logger.WithSpanID(ctx, spanID)
		c.Request = c.Request.WithContext(ctx)

		// Return headers to client
		c.Writer.Header().Set("X-Trace-ID", traceID)
		c.Writer.Header().Set("X-Span-ID", spanID)
		c.Writer.Header().Set("X-Request-ID", reqID)

		c.Next()

		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		c.Writer.Header().Set("X-Response-Time-Ms", fmt.Sprintf("%.2f", latencyMs))
	}
}

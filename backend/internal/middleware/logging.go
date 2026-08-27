package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type StructuredLog struct {
	Timestamp  string  `json:"timestamp"`
	Service    string  `json:"service"`
	ClientIP   string  `json:"client_ip"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	StatusCode int     `json:"status_code"`
	LatencyMs  float64 `json:"latency_ms"`
	UserAgent  string  `json:"user_agent"`
	ErrorMsg   string  `json:"error_msg,omitempty"`
}

// StructuredLogger logs HTTP access requests in JSON format to stdout for Loki ingestion
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}

		logEntry := StructuredLog{
			Timestamp:  start.UTC().Format(time.RFC3339),
			Service:    "tirenn-backend",
			ClientIP:   c.ClientIP(),
			Method:     c.Request.Method,
			Path:       path,
			StatusCode: c.Writer.Status(),
			LatencyMs:  float64(latency.Microseconds()) / 1000.0,
			UserAgent:  c.Request.UserAgent(),
			ErrorMsg:   c.Errors.ByType(gin.ErrorTypePrivate).String(),
		}

		jsonBytes, err := json.Marshal(logEntry)
		if err == nil {
			fmt.Printf("HTTP_ACCESS_LOG: %s\n", string(jsonBytes))
		}
	}
}

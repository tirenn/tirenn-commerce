package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

type ContextKey string

const RequestIDKey ContextKey = "request_id"

type StructuredLog struct {
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Level     string `json:"level"`
	RequestID string `json:"request_id,omitempty"`
	Layer     string `json:"layer,omitempty"`
	Caller    string `json:"caller,omitempty"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return reqID
	}
	if reqID, ok := ctx.Value("request_id").(string); ok && reqID != "" {
		return reqID
	}
	return ""
}

// WithRequestID wraps context with request ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func logMessage(ctx context.Context, level, layer, msg string, err error) {
	entry := StructuredLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Service:   "tirenn-backend",
		Level:     level,
		RequestID: GetRequestID(ctx),
		Layer:     layer,
		Caller:    getCaller(3),
		Message:   msg,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	bytes, _ := json.Marshal(entry)
	fmt.Printf("APP_LOG: %s\n", string(bytes))
}

// Error logs error level events with layer, caller trace, and error details
func Error(ctx context.Context, layer, msg string, err error) {
	logMessage(ctx, "ERROR", layer, msg, err)
}

// Warn logs warning level events
func Warn(ctx context.Context, layer, msg string, err error) {
	logMessage(ctx, "WARN", layer, msg, err)
}

// Info logs information level events
func Info(ctx context.Context, layer, msg string) {
	logMessage(ctx, "INFO", layer, msg, nil)
}

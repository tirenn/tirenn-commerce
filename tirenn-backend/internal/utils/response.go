package utils

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gocommerce-backend/internal/logger"
)

type APIResponse struct {
	Success   bool        `json:"success"`
	RequestID string      `json:"request_id,omitempty"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     interface{} `json:"error,omitempty"`
	Meta      interface{} `json:"meta,omitempty"`
}

type PaginationMeta struct {
	Page      int   `json:"page"`
	Limit     int   `json:"limit"`
	TotalRows int64 `json:"total_rows"`
	TotalPage int   `json:"total_pages"`
}

func getReqID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	reqID := logger.GetRequestID(c.Request.Context())
	if reqID == "" {
		if id, exists := c.Get("request_id"); exists {
			return id.(string)
		}
	}
	return reqID
}

// Success sends a 200/201 structured JSON response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success:   true,
		RequestID: getReqID(c),
		Message:   message,
		Data:      data,
	})
}

// SuccessWithMeta sends a structured JSON response with pagination metadata
func SuccessWithMeta(c *gin.Context, statusCode int, message string, data interface{}, meta interface{}) {
	c.JSON(statusCode, APIResponse{
		Success:   true,
		RequestID: getReqID(c),
		Message:   message,
		Data:      data,
		Meta:      meta,
	})
}

// Error sends an error JSON response and records structured error logs
func Error(c *gin.Context, statusCode int, message string, errDetails interface{}) {
	var errObj error
	if errDetails != nil {
		if e, ok := errDetails.(error); ok {
			errObj = e
		} else {
			errObj = fmt.Errorf("%v", errDetails)
		}
	}

	if statusCode >= 500 {
		logger.Error(c.Request.Context(), "handler", message, errObj)
	} else if statusCode >= 400 {
		logger.Warn(c.Request.Context(), "handler", message, errObj)
	}

	c.JSON(statusCode, APIResponse{
		Success:   false,
		RequestID: getReqID(c),
		Message:   message,
		Error:     errDetails,
	})
}

// BadRequest helper
func BadRequest(c *gin.Context, message string, errDetails interface{}) {
	Error(c, http.StatusBadRequest, message, errDetails)
}

// Unauthorized helper
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, nil)
}

// Forbidden helper
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, nil)
}

// NotFound helper
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message, nil)
}

// InternalServerError helper
func InternalServerError(c *gin.Context, message string, errDetails interface{}) {
	Error(c, http.StatusInternalServerError, message, errDetails)
}

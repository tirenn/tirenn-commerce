package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tirenn/commerce/backend/internal/domain"
	"github.com/tirenn/commerce/backend/internal/logger"
)

// PaginationMeta standard structure for paginated lists
type PaginationMeta struct {
	Page      int   `json:"page"`
	Limit     int   `json:"limit"`
	TotalRows int64 `json:"total_rows"`
	TotalPage int   `json:"total_page"`
}

// APIResponse standard structure for all API JSON outputs
type APIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    interface{}     `json:"data,omitempty"`
	Meta    *PaginationMeta `json:"meta,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Success sends an HTTP 200 OK JSON response
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta sends a paginated list response with pagination metadata
func SuccessWithMeta(c *gin.Context, statusCode int, message string, data interface{}, meta PaginationMeta) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    &meta,
	})
}

// OK sends an HTTP 200 JSON response with standard "Success" message
func OK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, "Success", data)
}

// Created sends an HTTP 201 Created JSON response
func Created(c *gin.Context, message string, data interface{}) {
	Success(c, http.StatusCreated, message, data)
}

// ErrorResponse sends a formatted error response with explicit status code and logs the error
func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
		_ = c.Error(err)
		
		// Log structured error
		if statusCode >= 500 {
			logger.Error(c.Request.Context(), "http.error", message, err)
		} else {
			logger.Warn(c.Request.Context(), "http.warn", message, err)
		}
	}
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
		Error:   errStr,
	})
}

// Error sends an intelligent error JSON response automatically mapping Domain Errors to HTTP status codes
func Error(c *gin.Context, message string, err error) {
	if err == nil {
		ErrorResponse(c, http.StatusInternalServerError, message, errors.New("unknown error"))
		return
	}

	switch {
	case errors.Is(err, domain.ErrNotFound):
		ErrorResponse(c, http.StatusNotFound, message, err)
	case errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrInvalidCredentials):
		ErrorResponse(c, http.StatusUnauthorized, message, err)
	case errors.Is(err, domain.ErrForbidden):
		ErrorResponse(c, http.StatusForbidden, message, err)
	case errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrEmailAlreadyExists):
		ErrorResponse(c, http.StatusConflict, message, err)
	case errors.Is(err, domain.ErrBadRequest) || errors.Is(err, domain.ErrInsufficientStock) || errors.Is(err, domain.ErrInvalidAdjustment):
		ErrorResponse(c, http.StatusBadRequest, message, err)
	default:
		ErrorResponse(c, http.StatusInternalServerError, message, err)
	}
}

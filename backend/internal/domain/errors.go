package domain

import "errors"

// Standard Sentinel Domain Errors
var (
	ErrNotFound            = errors.New("resource not found")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrForbidden           = errors.New("forbidden resource")
	ErrBadRequest         = errors.New("invalid request parameters")
	ErrConflict            = errors.New("resource conflict or already exists")
	ErrInsufficientStock   = errors.New("insufficient product inventory stock")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrEmailAlreadyExists  = errors.New("email address is already registered")
	ErrInvalidAdjustment   = errors.New("invalid stock adjustment type")
	ErrInternalServerError = errors.New("internal server error")
)

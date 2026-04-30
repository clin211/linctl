// Package errno defines application-level error codes and error variables.
package errno

import (
	"errors"
	"fmt"
	"net/http"
)

// BizError represents a business-level error with a code, reason, and message.
type BizError struct {
	// HTTP status code
	HTTPCode int
	// Machine-readable error code (e.g. 100001)
	Code int
	// Human-readable reason (e.g. "User.NotFound")
	Reason string
	// Human-readable message
	Message string
	// Optional details
	Details string
}

// Error implements the error interface.
func (e *BizError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s (%s)", e.Code, e.Reason, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Reason, e.Message)
}

// WithDetails returns a copy of the error with additional details.
func (e *BizError) WithDetails(details string) *BizError {
	cp := *e
	cp.Details = details
	return &cp
}

// New creates a new BizError.
func New(httpCode, code int, reason, message string) *BizError {
	return &BizError{
		HTTPCode: httpCode,
		Code:     code,
		Reason:   reason,
		Message:  message,
	}
}

// FromError converts a standard error into a BizError (using ErrInternal as fallback).
func FromError(err error) *BizError {
	if err == nil {
		return nil
	}
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return ErrInternal.WithDetails(err.Error())
}

// Standard errors.
var (
	ErrInternal          = New(http.StatusInternalServerError, 500001, "Internal.ServerError", "Internal server error.")
	ErrNotFound          = New(http.StatusNotFound, 404001, "NotFound.Resource", "Resource not found.")
	ErrBind              = New(http.StatusBadRequest, 400001, "Request.BindError", "Failed to bind request.")
	ErrInvalidArgument   = New(http.StatusBadRequest, 400002, "Request.InvalidArgument", "Invalid argument.")
	ErrUnauthenticated   = New(http.StatusUnauthorized, 401001, "Auth.Unauthenticated", "Unauthenticated.")
	ErrPermissionDenied  = New(http.StatusForbidden, 403001, "Auth.PermissionDenied", "Permission denied.")
)

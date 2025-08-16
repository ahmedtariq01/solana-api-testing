package models

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Authentication errors
	ErrorCodeMissingAPIKey  ErrorCode = "MISSING_API_KEY"
	ErrorCodeInvalidAPIKey  ErrorCode = "INVALID_API_KEY"
	ErrorCodeInactiveAPIKey ErrorCode = "INACTIVE_API_KEY"

	// Rate limiting errors
	ErrorCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"

	// Validation errors
	ErrorCodeInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrorCodeInvalidWallet    ErrorCode = "INVALID_WALLET_ADDRESS"
	ErrorCodeEmptyWalletArray ErrorCode = "EMPTY_WALLET_ARRAY"
	ErrorCodeMalformedJSON    ErrorCode = "MALFORMED_JSON"

	// RPC errors
	ErrorCodeRPCUnavailable     ErrorCode = "RPC_UNAVAILABLE"
	ErrorCodeRPCTimeout         ErrorCode = "RPC_TIMEOUT"
	ErrorCodeInvalidRPCResponse ErrorCode = "INVALID_RPC_RESPONSE"

	// Internal errors
	ErrorCodeDatabaseError ErrorCode = "DATABASE_ERROR"
	ErrorCodeCacheError    ErrorCode = "CACHE_ERROR"
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
)

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// ErrorResponse represents the standardized error response format
type ErrorResponse struct {
	Error     ErrorDetail `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
}

// HTTPStatusCode returns the appropriate HTTP status code for each error type
func (e ErrorCode) HTTPStatusCode() int {
	switch e {
	case ErrorCodeMissingAPIKey, ErrorCodeInvalidAPIKey, ErrorCodeInactiveAPIKey:
		return http.StatusUnauthorized
	case ErrorCodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case ErrorCodeInvalidRequest, ErrorCodeInvalidWallet, ErrorCodeEmptyWalletArray, ErrorCodeMalformedJSON:
		return http.StatusBadRequest
	case ErrorCodeRPCUnavailable, ErrorCodeRPCTimeout, ErrorCodeInvalidRPCResponse:
		return http.StatusBadGateway
	case ErrorCodeDatabaseError, ErrorCodeCacheError, ErrorCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewErrorResponse creates a new error response with timestamp
func NewErrorResponse(code ErrorCode, message, details string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC(),
	}
}

// NewErrorResponseWithCorrelation creates a new error response with correlation ID
func NewErrorResponseWithCorrelation(code ErrorCode, message, details, correlationID string) *ErrorResponseWithCorrelation {
	return &ErrorResponseWithCorrelation{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp:     time.Now().UTC(),
		CorrelationID: correlationID,
	}
}

// ErrorResponseWithCorrelation represents error response with correlation ID
type ErrorResponseWithCorrelation struct {
	Error         ErrorDetail `json:"error"`
	Timestamp     time.Time   `json:"timestamp"`
	CorrelationID string      `json:"correlation_id"`
}

// AppError represents an application error with context
type AppError struct {
	Code       ErrorCode
	Message    string
	Details    string
	Cause      error
	Context    map[string]interface{}
	StatusCode int
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: code.HTTPStatusCode(),
		Context:    make(map[string]interface{}),
	}
}

// NewAppErrorWithCause creates a new application error with underlying cause
func NewAppErrorWithCause(code ErrorCode, message string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Cause:      cause,
		StatusCode: code.HTTPStatusCode(),
		Context:    make(map[string]interface{}),
	}
}

// NewAppErrorWithDetails creates a new application error with details
func NewAppErrorWithDetails(code ErrorCode, message, details string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		StatusCode: code.HTTPStatusCode(),
		Context:    make(map[string]interface{}),
	}
}

// HandleError handles application errors and sends appropriate HTTP response
func HandleError(c *gin.Context, err error, logger interface{}) {
	var appErr *AppError
	var correlationID string

	// Extract correlation ID from context
	if ctx := c.Request.Context(); ctx != nil {
		if cid := ctx.Value("correlation_id"); cid != nil {
			correlationID = cid.(string)
		}
	}
	if correlationID == "" {
		if cid := c.GetString("correlation_id"); cid != "" {
			correlationID = cid
		}
	}

	// Convert error to AppError if needed
	if appError, ok := err.(*AppError); ok {
		appErr = appError
	} else {
		// Create generic internal error
		appErr = NewAppErrorWithCause(ErrorCodeInternalError, "Internal server error", err)
	}

	// Add request context to error
	appErr.WithContext("method", c.Request.Method).
		WithContext("path", c.Request.URL.Path).
		WithContext("client_ip", c.ClientIP())

	// Log the error with appropriate level
	if l, ok := logger.(interface {
		WithContext(context.Context) interface {
			Error(string, ...zap.Field)
			Warn(string, ...zap.Field)
		}
	}); ok {
		contextLogger := l.WithContext(c.Request.Context())

		logFields := []zap.Field{
			zap.String("error_code", string(appErr.Code)),
			zap.String("error_message", appErr.Message),
			zap.Any("error_context", appErr.Context),
		}

		if appErr.Cause != nil {
			logFields = append(logFields, zap.Error(appErr.Cause))
		}

		if appErr.StatusCode >= 500 {
			contextLogger.Error("Application error", logFields...)
		} else {
			contextLogger.Warn("Client error", logFields...)
		}
	}

	// Create error response
	var response interface{}
	if correlationID != "" {
		response = NewErrorResponseWithCorrelation(
			appErr.Code,
			appErr.Message,
			appErr.Details,
			correlationID,
		)
	} else {
		response = NewErrorResponse(
			appErr.Code,
			appErr.Message,
			appErr.Details,
		)
	}

	// Send HTTP response
	c.JSON(appErr.StatusCode, response)
}

// Common error constructors for specific scenarios

// NewValidationError creates a validation error
func NewValidationError(message, details string) *AppError {
	return NewAppErrorWithDetails(ErrorCodeInvalidRequest, message, details)
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string) *AppError {
	return NewAppError(ErrorCodeInvalidAPIKey, message)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError() *AppError {
	return NewAppError(ErrorCodeRateLimitExceeded, "Rate limit exceeded")
}

// NewRPCError creates an RPC error
func NewRPCError(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeRPCUnavailable, message, cause)
}

// NewDatabaseError creates a database error
func NewDatabaseError(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeDatabaseError, message, cause)
}

// NewCacheError creates a cache error
func NewCacheError(message string, cause error) *AppError {
	return NewAppErrorWithCause(ErrorCodeCacheError, message, cause)
}

package errors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeExternal   ErrorType = "external_api"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypePermission ErrorType = "permission"
	ErrorTypeRateLimit  ErrorType = "rate_limit"
	ErrorTypeTimeout    ErrorType = "timeout"
)

// AppError represents an application error with additional context
type AppError struct {
	Type     ErrorType
	Message  string
	Code     string
	Internal error
	Context  map[string]interface{}
	Source   string
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Type, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the internal error
func (e *AppError) Unwrap() error {
	return e.Internal
}

// Is checks if the error matches the target
func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return errors.Is(e.Internal, target)
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// LogFields returns structured logging fields
func (e *AppError) LogFields() []interface{} {
	fields := []interface{}{
		"error_type", e.Type,
		"error_code", e.Code,
		"error_message", e.Message,
		"source", e.Source,
	}

	if e.Internal != nil {
		fields = append(fields, "internal_error", e.Internal.Error())
	}

	for k, v := range e.Context {
		fields = append(fields, k, v)
	}

	return fields
}

// New creates a new AppError
func New(errorType ErrorType, code, message string) *AppError {
	_, file, line, _ := runtime.Caller(1)
	source := fmt.Sprintf("%s:%d", file, line)

	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Source:  source,
		Context: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error into AppError
func Wrap(err error, errorType ErrorType, code, message string) *AppError {
	_, file, line, _ := runtime.Caller(1)
	source := fmt.Sprintf("%s:%d", file, line)

	return &AppError{
		Type:     errorType,
		Code:     code,
		Message:  message,
		Internal: err,
		Source:   source,
		Context:  make(map[string]interface{}),
	}
}

// Handler provides error handling strategies
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a new error handler
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

// Handle processes an error according to its type
func (h *Handler) Handle(ctx context.Context, err error) {
	if err == nil {
		return
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		h.handleAppError(ctx, appErr)
	} else {
		h.handleGenericError(ctx, err)
	}
}

// handleAppError handles AppError instances
func (h *Handler) handleAppError(ctx context.Context, err *AppError) {
	switch err.Type {
	case ErrorTypeValidation:
		h.logger.WarnContext(ctx, "Validation error", err.LogFields()...)
	case ErrorTypePermission:
		h.logger.WarnContext(ctx, "Permission error", err.LogFields()...)
	case ErrorTypeRateLimit:
		h.logger.WarnContext(ctx, "Rate limit error", err.LogFields()...)
	case ErrorTypeDatabase, ErrorTypeExternal, ErrorTypeInternal, ErrorTypeTimeout:
		h.logger.ErrorContext(ctx, "Critical error", err.LogFields()...)
	default:
		h.logger.ErrorContext(ctx, "Unknown error type", err.LogFields()...)
	}
}

// handleGenericError handles generic errors
func (h *Handler) handleGenericError(ctx context.Context, err error) {
	h.logger.ErrorContext(ctx, "Unhandled error", "error", err.Error())
}

// LogAndReturn logs an error and returns it
func (h *Handler) LogAndReturn(ctx context.Context, err error) error {
	h.Handle(ctx, err)
	return err
}

// Predefined errors
var (
	ErrInvalidInput      = New(ErrorTypeValidation, "INVALID_INPUT", "Invalid input provided")
	ErrUserNotFound      = New(ErrorTypeDatabase, "USER_NOT_FOUND", "User not found")
	ErrDatabaseError     = New(ErrorTypeDatabase, "DB_ERROR", "Database operation failed")
	ErrExternalAPI       = New(ErrorTypeExternal, "EXTERNAL_API", "External API error")
	ErrUnauthorized      = New(ErrorTypePermission, "UNAUTHORIZED", "Unauthorized access")
	ErrRateLimitExceeded = New(ErrorTypeRateLimit, "RATE_LIMIT", "Rate limit exceeded")
	ErrTimeout           = New(ErrorTypeTimeout, "TIMEOUT", "Operation timed out")
	ErrInternalServer    = New(ErrorTypeInternal, "INTERNAL", "Internal server error")
)

// Convenience functions for common errors
func NewValidationError(message string) *AppError {
	return New(ErrorTypeValidation, "VALIDATION", message)
}

func NewDatabaseError(err error) *AppError {
	return Wrap(err, ErrorTypeDatabase, "DB_ERROR", "Database operation failed")
}

func NewExternalAPIError(err error, api string) *AppError {
	return Wrap(err, ErrorTypeExternal, "EXTERNAL_API", fmt.Sprintf("%s API error", api)).
		WithContext("api", api)
}

func NewTimeoutError(operation string) *AppError {
	return New(ErrorTypeTimeout, "TIMEOUT", fmt.Sprintf("%s operation timed out", operation)).
		WithContext("operation", operation)
}

func NewInternalError(err error) *AppError {
	return Wrap(err, ErrorTypeInternal, "INTERNAL", "Internal server error")
}

package api

import "net/http"

// Standard HubSpot error categories.
const (
	CategoryValidationError = "VALIDATION_ERROR"
	CategoryObjectNotFound  = "OBJECT_NOT_FOUND"
	CategoryConflict        = "CONFLICT"
	CategoryRateLimits      = "RATE_LIMITS"
)

// Error represents a HubSpot-compatible error response.
type Error struct {
	Status        string        `json:"status"`
	Message       string        `json:"message"`
	CorrelationID string        `json:"correlationId"`
	Category      string        `json:"category"`
	SubCategory   string        `json:"subCategory,omitempty"`
	Errors        []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail represents a single error within an Error.
type ErrorDetail struct {
	Message     string              `json:"message"`
	Code        string              `json:"code,omitempty"`
	In          string              `json:"in,omitempty"`
	Context     map[string][]string `json:"context,omitempty"`
	SubCategory string              `json:"subCategory,omitempty"`
}

// NewNotFoundError creates a 404 error with the OBJECT_NOT_FOUND category.
func NewNotFoundError(message, correlationID string) *Error {
	return &Error{
		Status:        "error",
		Message:       message,
		CorrelationID: correlationID,
		Category:      CategoryObjectNotFound,
	}
}

// NewValidationError creates a 400 error with the VALIDATION_ERROR category.
func NewValidationError(message, correlationID string, details []ErrorDetail) *Error {
	return &Error{
		Status:        "error",
		Message:       message,
		CorrelationID: correlationID,
		Category:      CategoryValidationError,
		Errors:        details,
	}
}

// NewConflictError creates a 409 error with the CONFLICT category.
func NewConflictError(message, correlationID string) *Error {
	return &Error{
		Status:        "error",
		Message:       message,
		CorrelationID: correlationID,
		Category:      CategoryConflict,
	}
}

// WriteError writes an Error as a JSON response with the given HTTP status code.
func WriteError(w http.ResponseWriter, statusCode int, apiErr *Error) {
	WriteJSON(w, statusCode, apiErr)
}

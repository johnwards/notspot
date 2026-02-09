package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
)

func TestNewNotFoundError(t *testing.T) {
	err := api.NewNotFoundError("object not found", "abc-123")

	if err.Status != "error" {
		t.Errorf("Status = %q, want %q", err.Status, "error")
	}
	if err.Category != api.CategoryObjectNotFound {
		t.Errorf("Category = %q, want %q", err.Category, api.CategoryObjectNotFound)
	}
	if err.CorrelationID != "abc-123" {
		t.Errorf("CorrelationID = %q, want %q", err.CorrelationID, "abc-123")
	}
	if err.Message != "object not found" {
		t.Errorf("Message = %q, want %q", err.Message, "object not found")
	}
}

func TestNewValidationError(t *testing.T) {
	details := []api.ErrorDetail{
		{Message: "field is required", Code: "REQUIRED"},
	}
	err := api.NewValidationError("invalid input", "def-456", details)

	if err.Category != api.CategoryValidationError {
		t.Errorf("Category = %q, want %q", err.Category, api.CategoryValidationError)
	}
	if len(err.Errors) != 1 {
		t.Fatalf("Errors length = %d, want 1", len(err.Errors))
	}
	if err.Errors[0].Code != "REQUIRED" {
		t.Errorf("Errors[0].Code = %q, want %q", err.Errors[0].Code, "REQUIRED")
	}
}

func TestNewConflictError(t *testing.T) {
	err := api.NewConflictError("already exists", "ghi-789")

	if err.Category != api.CategoryConflict {
		t.Errorf("Category = %q, want %q", err.Category, api.CategoryConflict)
	}
}

func TestWriteErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	apiErr := api.NewNotFoundError("not found", "test-id")

	api.WriteError(rec, http.StatusNotFound, apiErr)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusNotFound)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var result api.Error
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if result.CorrelationID != "test-id" {
		t.Errorf("correlationId = %q, want %q", result.CorrelationID, "test-id")
	}
}

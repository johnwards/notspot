package exports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/exports"
	"github.com/johnwards/hubspot/internal/api/objects"
	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	s := store.New(db)
	mux := http.NewServeMux()
	exports.RegisterRoutes(mux, s)
	objects.RegisterRoutes(mux, s)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

func createContact(t *testing.T, srv *httptest.Server, email string) {
	t.Helper()
	body := `{"properties":{"email":"` + email + `","firstname":"Test"}}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	_ = resp.Body.Close()
}

type exportStatusResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result *struct {
		RecordCount int `json:"recordCount"`
	} `json:"result,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func TestStartExport(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createContact(t, srv, "export1@example.com")
	createContact(t, srv, "export2@example.com")

	body := `{"exportType":"VIEW","exportName":"Test Export","objectType":"contacts","objectProperties":["email","firstname"]}`
	resp, err := http.Post(srv.URL+"/crm/v3/exports/export/async", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	var result exportStatusResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
	if result.Status != "COMPLETE" {
		t.Errorf("expected status=COMPLETE, got %s", result.Status)
	}
	if result.Result == nil || result.Result.RecordCount != 2 {
		t.Errorf("expected recordCount=2, got %+v", result.Result)
	}
}

func TestGetExportStatus(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createContact(t, srv, "status@example.com")

	body := `{"exportType":"VIEW","objectType":"contacts","objectProperties":["email"]}`
	resp, err := http.Post(srv.URL+"/crm/v3/exports/export/async", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	var created exportStatusResp
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	statusResp, err := http.Get(srv.URL + "/crm/v3/exports/export/async/tasks/" + created.ID + "/status")
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	defer func() { _ = statusResp.Body.Close() }()

	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusResp.StatusCode)
	}

	var result exportStatusResp
	if err := json.NewDecoder(statusResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.Status != "COMPLETE" {
		t.Errorf("expected status=COMPLETE, got %s", result.Status)
	}
}

func TestGetExportStatusNotFound(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/exports/export/async/tasks/999/status")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestStartExportValidation(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	// Missing objectType.
	body := `{"objectProperties":["email"]}`
	resp, err := http.Post(srv.URL+"/crm/v3/exports/export/async", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	var apiErr api.Error
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if apiErr.Category != "VALIDATION_ERROR" {
		t.Errorf("expected category=VALIDATION_ERROR, got %s", apiErr.Category)
	}
}

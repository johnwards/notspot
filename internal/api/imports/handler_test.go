package imports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/imports"
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
	imports.RegisterRoutes(mux, s)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

func createImport(t *testing.T, srv *httptest.Server, importReqJSON, csvContent string) *http.Response {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add importRequest field.
	if err := writer.WriteField("importRequest", importReqJSON); err != nil {
		t.Fatalf("write field: %v", err)
	}

	// Add CSV file.
	part, err := writer.CreateFormFile("files", "contacts.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte(csvContent)); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	resp, err := http.Post(srv.URL+"/crm/v3/imports", writer.FormDataContentType(), &buf)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func TestStartImport(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	importReq := `{
		"name": "Test Import",
		"importOperations": {"0-1": {"objectTypeId": "0-1", "importOperationType": "CREATE"}},
		"files": [{
			"fileName": "contacts.csv",
			"fileFormat": "CSV",
			"fileImportPage": {
				"hasHeader": true,
				"columnMappings": [
					{"columnObjectTypeId": "0-1", "columnName": "Email", "propertyName": "email"},
					{"columnObjectTypeId": "0-1", "columnName": "First Name", "propertyName": "firstname"}
				]
			}
		}]
	}`

	csv := "Email,First Name\nalice@example.com,Alice\nbob@example.com,Bob\n"

	resp := createImport(t, srv, importReq, csv)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result store.ImportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.ID == "" {
		t.Error("expected non-empty ID")
	}
	if result.State != "DONE" {
		t.Errorf("expected state=DONE, got %s", result.State)
	}
	if result.Name != "Test Import" {
		t.Errorf("expected name=Test Import, got %s", result.Name)
	}
}

func TestListImports(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	importReq := `{
		"name": "List Test",
		"importOperations": {"0-1": {"objectTypeId": "0-1", "importOperationType": "CREATE"}},
		"files": [{"fileName": "contacts.csv", "fileFormat": "CSV", "fileImportPage": {"hasHeader": true, "columnMappings": [{"columnObjectTypeId": "0-1", "columnName": "Email", "propertyName": "email"}]}}]
	}`

	resp := createImport(t, srv, importReq, "Email\ntest@example.com\n")
	_ = resp.Body.Close()

	listResp, err := http.Get(srv.URL + "/crm/v3/imports")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = listResp.Body.Close() }()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.StatusCode)
	}

	var result api.CollectionResponse
	if err := json.NewDecoder(listResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result.Results) != 1 {
		t.Errorf("expected 1 import, got %d", len(result.Results))
	}
}

func TestGetImport(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	importReq := `{
		"name": "Get Test",
		"importOperations": {"0-1": {"objectTypeId": "0-1", "importOperationType": "CREATE"}},
		"files": [{"fileName": "c.csv", "fileFormat": "CSV", "fileImportPage": {"hasHeader": true, "columnMappings": [{"columnObjectTypeId": "0-1", "columnName": "Email", "propertyName": "email"}]}}]
	}`

	resp := createImport(t, srv, importReq, "Email\nget@example.com\n")
	var created store.ImportResponse
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	getResp, err := http.Get(srv.URL + "/crm/v3/imports/" + created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = getResp.Body.Close() }()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
}

func TestCancelImport(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	importReq := `{
		"name": "Cancel Test",
		"importOperations": {"0-1": {"objectTypeId": "0-1", "importOperationType": "CREATE"}},
		"files": [{"fileName": "c.csv", "fileFormat": "CSV", "fileImportPage": {"hasHeader": true, "columnMappings": [{"columnObjectTypeId": "0-1", "columnName": "Email", "propertyName": "email"}]}}]
	}`

	resp := createImport(t, srv, importReq, "Email\ncancel@example.com\n")
	var created store.ImportResponse
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	cancelResp, err := http.Post(srv.URL+"/crm/v3/imports/"+created.ID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = cancelResp.Body.Close() }()

	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", cancelResp.StatusCode)
	}

	var result store.ImportResponse
	if err := json.NewDecoder(cancelResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if result.State != "CANCELED" {
		t.Errorf("expected state=CANCELED, got %s", result.State)
	}
}

func TestGetImportErrors(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	// Import with a bad object type to generate errors.
	importReq := `{
		"name": "Error Test",
		"importOperations": {"bad-type": {"objectTypeId": "bad-type", "importOperationType": "CREATE"}},
		"files": [{"fileName": "c.csv", "fileFormat": "CSV", "fileImportPage": {"hasHeader": true, "columnMappings": [{"columnObjectTypeId": "bad-type", "columnName": "Email", "propertyName": "email"}]}}]
	}`

	resp := createImport(t, srv, importReq, "Email\nerror@example.com\n")
	var created store.ImportResponse
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	errResp, err := http.Get(srv.URL + "/crm/v3/imports/" + created.ID + "/errors")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = errResp.Body.Close() }()

	if errResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", errResp.StatusCode)
	}

	var result api.CollectionResponse
	if err := json.NewDecoder(errResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result.Results) == 0 {
		t.Error("expected at least one import error")
	}
}

func TestGetImportNotFound(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/imports/999")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

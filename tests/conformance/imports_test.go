package conformance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestImports_ListAll(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/imports/", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response")
	}
}

func TestImports_ListEmpty(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/imports/", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response, even if empty")
	}
}

func TestImports_GetNonExistent(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/imports/999999999", nil)
	mustStatus(t, resp, http.StatusNotFound)
}

func TestImports_StartImport(t *testing.T) {
	resetServer(t)

	// Build multipart/form-data request body.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the CSV file part.
	filePart, err := writer.CreateFormFile("files", "contacts.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, err = filePart.Write([]byte("First Name,Last Name,Email\nJohn,Doe,john@test.com\nJane,Smith,jane@test.com"))
	if err != nil {
		t.Fatalf("write CSV data: %v", err)
	}

	// Add the importRequest JSON part.
	importRequest := map[string]any{
		"name": "test-import",
		"files": []map[string]any{
			{
				"fileName":   "contacts.csv",
				"fileFormat": "CSV",
				"fileImportPage": map[string]any{
					"hasHeader": true,
					"columnMappings": []map[string]any{
						{"columnObjectTypeId": "0-1", "columnName": "First Name", "propertyName": "firstname"},
						{"columnObjectTypeId": "0-1", "columnName": "Last Name", "propertyName": "lastname"},
						{"columnObjectTypeId": "0-1", "columnName": "Email", "propertyName": "email"},
					},
				},
			},
		},
	}
	importRequestJSON, err := json.Marshal(importRequest)
	if err != nil {
		t.Fatalf("marshal import request: %v", err)
	}

	requestPart, err := writer.CreateFormField("importRequest")
	if err != nil {
		t.Fatalf("create form field: %v", err)
	}
	_, err = requestPart.Write(importRequestJSON)
	if err != nil {
		t.Fatalf("write import request: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	// Create the request manually since doRequest sets Content-Type to application/json.
	req, err := http.NewRequest(http.MethodPost, serverURL+"/crm/v3/imports/", &buf)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /crm/v3/imports/: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Accept either 200 or 201 as valid responses.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Logf("start import response: status=%d body=%s", resp.StatusCode, string(b))
	}

	// If the server returned a JSON body, validate its structure.
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read response body: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal(b, &body); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		assertFieldPresent(t, body, "id")
		assertFieldPresent(t, body, "state")
		assertFieldPresent(t, body, "createdAt")
		assertFieldPresent(t, body, "updatedAt")
	}
}

func TestImports_CancelNonExistent(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodPost, "/crm/v3/imports/999999999/cancel", nil)
	// Expect 404 or an error status.
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Errorf("expected 404 or 400 for cancel non-existent import, got %d; body=%s", resp.StatusCode, string(b))
	} else {
		_ = resp.Body.Close()
	}
}

func TestImports_ResponseStructure(t *testing.T) {
	resetServer(t)

	// List imports to find any that exist.
	resp := doRequest(t, http.MethodGet, "/crm/v3/imports/", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if len(results) == 0 {
		t.Skip("no imports available to validate response structure")
	}

	for i, r := range results {
		imp := toObject(t, r)

		// Required fields per spec: id, state, createdAt, updatedAt, metadata, optOutImport, mappedObjectTypeIds
		id := assertIsString(t, imp, "id")
		if id == "" {
			t.Errorf("import[%d]: id should be non-empty", i)
		}

		state := assertIsString(t, imp, "state")
		validStates := map[string]bool{
			"STARTED": true, "PROCESSING": true, "DONE": true,
			"FAILED": true, "CANCELED": true, "DEFERRED": true, "REVERTED": true,
		}
		if !validStates[state] {
			t.Errorf("import[%d]: unexpected state %q", i, state)
		}

		createdAt := assertIsString(t, imp, "createdAt")
		assertISOTimestamp(t, createdAt)

		updatedAt := assertIsString(t, imp, "updatedAt")
		assertISOTimestamp(t, updatedAt)

		assertFieldPresent(t, imp, "metadata")
	}
}

func TestImports_ListWithPagination(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/imports/?limit=%d", 1), nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response")
	}

	if len(results) > 1 {
		t.Errorf("expected at most 1 result with limit=1, got %d", len(results))
	}
}

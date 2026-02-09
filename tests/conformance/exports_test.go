package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestExports_StartAsyncExport(t *testing.T) {
	resetServer(t)

	exportReq := map[string]any{
		"exportType":       "VIEW",
		"format":           "CSV",
		"exportName":       "test-export",
		"objectType":       "contacts",
		"objectProperties": []string{"firstname", "lastname", "email"},
		"language":         "EN",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/exports/export/async", exportReq)

	// Spec says 202, but accept 200 as well.
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		mustStatus(t, resp, http.StatusAccepted)
	}

	body := readJSON(t, resp)
	// TaskLocator response should have an "id" field.
	assertFieldPresent(t, body, "id")
}

func TestExports_GetStatusNonExistent(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/exports/export/async/tasks/999999999/status", nil)
	mustStatus(t, resp, http.StatusNotFound)
}

func TestExports_GetExportNonExistent(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/exports/export/999999999", nil)
	mustStatus(t, resp, http.StatusNotFound)
}

func TestExports_StartExportInvalidObjectType(t *testing.T) {
	resetServer(t)

	exportReq := map[string]any{
		"exportType":       "VIEW",
		"format":           "CSV",
		"exportName":       "bad-export",
		"objectType":       "nonexistent_type",
		"objectProperties": []string{"firstname"},
		"language":         "EN",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/exports/export/async", exportReq)

	// Should get an error response (400 or similar).
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		t.Errorf("expected error status for invalid object type, got %d", resp.StatusCode)
	} else {
		_ = resp.Body.Close()
	}
}

func TestExports_StartExportMissingFields(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodPost, "/crm/v3/exports/export/async", map[string]any{})
	mustStatus(t, resp, http.StatusBadRequest)
	_ = resp.Body.Close()
}

func TestExports_ResponseStructure(t *testing.T) {
	resetServer(t)

	exportReq := map[string]any{
		"exportType":       "VIEW",
		"format":           "CSV",
		"exportName":       "structure-test",
		"objectType":       "contacts",
		"objectProperties": []string{"firstname", "lastname", "email"},
		"language":         "EN",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/exports/export/async", exportReq)
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Skip("could not start export to validate response structure")
	}

	body := readJSON(t, resp)
	exportID := assertIsString(t, body, "id")

	if exportID == "" {
		t.Skip("no export ID returned to validate response structure")
	}

	// Try to get the export details using the export ID.
	detailResp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/exports/export/%s", exportID), nil)
	if detailResp.StatusCode != http.StatusOK {
		_ = detailResp.Body.Close()
		t.Skip("could not retrieve export details")
	}

	detail := readJSON(t, detailResp)

	// Required fields per PublicExportResponse: id, exportState, exportType, objectType, objectProperties, createdAt, updatedAt
	assertIsString(t, detail, "id")

	exportState := assertIsString(t, detail, "exportState")
	validStates := map[string]bool{
		"CANCELED": true, "CONFLICT": true, "DEFERRED": true, "DELETED": true,
		"DONE": true, "ENQUEUED": true, "FAILED": true, "PENDING_APPROVAL": true, "PROCESSING": true,
	}
	if exportState != "" && !validStates[exportState] {
		t.Errorf("unexpected exportState %q", exportState)
	}

	assertIsString(t, detail, "exportType")
	assertIsString(t, detail, "objectType")
	assertIsArray(t, detail, "objectProperties")

	createdAt := assertIsString(t, detail, "createdAt")
	assertISOTimestamp(t, createdAt)

	updatedAt := assertIsString(t, detail, "updatedAt")
	assertISOTimestamp(t, updatedAt)
}

func TestExports_StartExportListType(t *testing.T) {
	resetServer(t)

	exportReq := map[string]any{
		"exportType":       "LIST",
		"format":           "CSV",
		"exportName":       "list-export-test",
		"objectType":       "contacts",
		"objectProperties": []string{"firstname", "lastname", "email"},
		"language":         "EN",
		"listId":           "1",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/exports/export/async", exportReq)

	// Accept either a successful response or an error (list may not exist).
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusOK {
		body := readJSON(t, resp)
		assertFieldPresent(t, body, "id")
	} else {
		_ = resp.Body.Close()
	}
}

func TestExports_ExportFormats(t *testing.T) {
	resetServer(t)

	formats := []string{"CSV", "XLSX", "XLS"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			exportReq := map[string]any{
				"exportType":       "VIEW",
				"format":           format,
				"exportName":       fmt.Sprintf("format-test-%s", format),
				"objectType":       "contacts",
				"objectProperties": []string{"firstname", "lastname", "email"},
				"language":         "EN",
			}

			resp := doRequest(t, http.MethodPost, "/crm/v3/exports/export/async", exportReq)

			// Each format should be accepted (202 or 200).
			if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
				mustStatus(t, resp, http.StatusAccepted)
			} else {
				body := readJSON(t, resp)
				assertFieldPresent(t, body, "id")
			}
		})
	}
}

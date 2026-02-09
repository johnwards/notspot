package conformance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// TestError_InvalidObjectType verifies that requesting a non-existent object type
// returns an error in HubSpot error format.
func TestError_InvalidObjectType(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/nonexistent_type", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 or 404, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "")
}

// TestError_CreateWithEmptyBody verifies that creating a contact with an empty
// JSON object ({}) returns 400.
func TestError_CreateWithEmptyBody(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts", map[string]any{})
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_CreateWithNullBody verifies that creating a contact with a nil/no body
// returns 400.
func TestError_CreateWithNullBody(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "")
}

// TestError_InvalidJSON verifies that sending malformed JSON returns 400.
func TestError_InvalidJSON(t *testing.T) {
	resetServer(t)

	req, err := http.NewRequest(http.MethodPost, serverURL+"/crm/v3/objects/contacts",
		bytes.NewReader([]byte("{invalid json")))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "")
}

// TestError_InvalidFilterOperator verifies that searching with an invalid filter
// operator returns 400.
func TestError_InvalidFilterOperator(t *testing.T) {
	resetServer(t)

	searchBody := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "INVALID_OP",
						"value":        "test",
					},
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/search", searchBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_ExceedBatchLimit verifies that sending more than 100 items in a
// batch create returns 400.
func TestError_ExceedBatchLimit(t *testing.T) {
	resetServer(t)

	inputs := make([]any, 101)
	for i := range inputs {
		inputs[i] = map[string]any{
			"properties": map[string]string{
				"firstname": fmt.Sprintf("BatchUser%d", i),
				"lastname":  "Test",
				"email":     fmt.Sprintf("batch%d@example.com", i),
			},
		}
	}

	batchBody := map[string]any{
		"inputs": inputs,
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", batchBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_ExceedFilterGroupLimit verifies that searching with more than 5
// filter groups returns 400.
func TestError_ExceedFilterGroupLimit(t *testing.T) {
	resetServer(t)

	filterGroups := make([]any, 6)
	for i := range filterGroups {
		filterGroups[i] = map[string]any{
			"filters": []any{
				map[string]any{
					"propertyName": "firstname",
					"operator":     "EQ",
					"value":        fmt.Sprintf("name%d", i),
				},
			},
		}
	}

	searchBody := map[string]any{
		"filterGroups": filterGroups,
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/search", searchBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_ExceedFiltersPerGroupLimit verifies that searching with more than 6
// filters in a single group returns 400.
func TestError_ExceedFiltersPerGroupLimit(t *testing.T) {
	resetServer(t)

	filters := make([]any, 7)
	for i := range filters {
		filters[i] = map[string]any{
			"propertyName": "firstname",
			"operator":     "EQ",
			"value":        fmt.Sprintf("name%d", i),
		}
	}

	searchBody := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": filters,
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/search", searchBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_DuplicatePropertyName verifies that creating a property with a name
// that already exists returns 409 CONFLICT.
func TestError_DuplicatePropertyName(t *testing.T) {
	resetServer(t)

	propBody := map[string]any{
		"name":      "custom_test_prop",
		"label":     "Custom Test Prop",
		"type":      "string",
		"fieldType": "text",
		"groupName": "contactinformation",
	}

	// Create the property the first time — should succeed.
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", propBody)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("first create property: expected 200 or 201, got %d; body=%s", resp.StatusCode, string(b))
	}
	_ = resp.Body.Close()

	// Create the same property again — should return 409.
	resp2 := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", propBody)
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("expected 409, got %d; body=%s", resp2.StatusCode, string(b))
	}

	body := readJSON(t, resp2)
	assertHubSpotError(t, body, "CONFLICT")
}

// TestError_UpdateNonExistentObject verifies that updating a non-existent contact
// returns 404.
func TestError_UpdateNonExistentObject(t *testing.T) {
	resetServer(t)

	updateBody := map[string]any{
		"properties": map[string]string{
			"firstname": "test",
		},
	}

	resp := doRequest(t, http.MethodPatch, "/crm/v3/objects/contacts/999999999", updateBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "OBJECT_NOT_FOUND")
}

// TestError_DeleteNonExistentObject verifies that deleting a non-existent contact
// returns 404.
func TestError_DeleteNonExistentObject(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodDelete, "/crm/v3/objects/contacts/999999999", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "OBJECT_NOT_FOUND")
}

// TestError_GetNonExistentObject verifies that getting a non-existent contact
// returns 404.
func TestError_GetNonExistentObject(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/999999999", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "OBJECT_NOT_FOUND")
}

// TestError_AssociateNonExistentObjects verifies that associating objects that
// don't exist returns an error.
func TestError_AssociateNonExistentObjects(t *testing.T) {
	resetServer(t)

	assocBody := []any{
		map[string]any{
			"associationCategory": "HUBSPOT_DEFINED",
			"associationTypeId":   1,
		},
	}

	resp := doRequest(t, http.MethodPut, "/crm/v4/objects/contacts/999999999/associations/companies/999999998", assocBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 or 404, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "")
}

// TestError_InvalidPipelineObjectType verifies that creating a pipeline for an
// object type that doesn't support pipelines (e.g. contacts) returns an error.
func TestError_InvalidPipelineObjectType(t *testing.T) {
	resetServer(t)

	pipelineBody := map[string]any{
		"label":        "Test Pipeline",
		"displayOrder": 0,
		"stages": []any{
			map[string]any{
				"label":        "Stage 1",
				"displayOrder": 0,
				"metadata": map[string]any{
					"probability": "0.5",
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/pipelines/contacts", pipelineBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400 or 404, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "")
}

// TestError_DuplicatePipelineStageLabel verifies behavior when creating a pipeline
// with two stages that have the same label but different displayOrder.
func TestError_DuplicatePipelineStageLabel(t *testing.T) {
	resetServer(t)

	pipelineBody := map[string]any{
		"label":        "Dup Stage Pipeline",
		"displayOrder": 0,
		"stages": []any{
			map[string]any{
				"label":        "Same Label",
				"displayOrder": 0,
				"metadata": map[string]any{
					"probability": "0.2",
				},
			},
			map[string]any{
				"label":        "Same Label",
				"displayOrder": 1,
				"metadata": map[string]any{
					"probability": "0.5",
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/pipelines/deals", pipelineBody)
	defer func() { _ = resp.Body.Close() }()

	// HubSpot may reject duplicate stage labels or may allow them — record behavior.
	b, _ := io.ReadAll(resp.Body)
	t.Logf("duplicate stage label response: status=%d body=%s", resp.StatusCode, string(b))

	// If it returns an error, validate the error format.
	if resp.StatusCode >= 400 {
		var body map[string]any
		if err := json.Unmarshal(b, &body); err == nil {
			assertHubSpotError(t, body, "")
		}
	}
}

// TestError_PropertyTypeValidation verifies that setting a non-numeric value on
// a number-type property returns an error.
func TestError_PropertyTypeValidation(t *testing.T) {
	resetServer(t)

	// Create a number-type property.
	propBody := map[string]any{
		"name":      "custom_number_prop",
		"label":     "Custom Number Prop",
		"type":      "number",
		"fieldType": "number",
		"groupName": "contactinformation",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", propBody)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create number property: expected 200 or 201, got %d; body=%s", resp.StatusCode, string(b))
	}
	_ = resp.Body.Close()

	// Try to create a contact with a non-numeric value for that property.
	contactBody := map[string]any{
		"properties": map[string]string{
			"firstname":          "NumTest",
			"custom_number_prop": "not_a_number",
		},
	}

	resp2 := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts", contactBody)
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp2.Body)
		t.Fatalf("expected 400 for invalid number value, got %d; body=%s", resp2.StatusCode, string(b))
	}

	body := readJSON(t, resp2)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_BatchCreateInvalidInputs verifies that batch creating contacts with
// invalid input data (missing properties field) returns an error.
func TestError_BatchCreateInvalidInputs(t *testing.T) {
	resetServer(t)

	batchBody := map[string]any{
		"inputs": []any{
			map[string]any{
				// Missing "properties" field entirely.
				"email": "bad@example.com",
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", batchBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected error status (>=400), got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "")
}

// TestError_SearchWithInvalidSort verifies behavior when searching with a sort
// on a non-existent property.
func TestError_SearchWithInvalidSort(t *testing.T) {
	resetServer(t)

	searchBody := map[string]any{
		"sorts": []any{
			map[string]any{
				"propertyName": "completely_nonexistent_property_xyz",
				"direction":    "ASCENDING",
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/search", searchBody)
	defer func() { _ = resp.Body.Close() }()

	// Log the response — HubSpot may return 400 or may just ignore invalid sorts.
	b, _ := io.ReadAll(resp.Body)
	t.Logf("search with invalid sort response: status=%d body=%s", resp.StatusCode, string(b))

	if resp.StatusCode >= 400 {
		var body map[string]any
		if err := json.Unmarshal(b, &body); err == nil {
			assertHubSpotError(t, body, "")
		}
	}
}

// TestError_CreatePropertyInvalidType verifies that creating a property with an
// invalid type string returns 400.
func TestError_CreatePropertyInvalidType(t *testing.T) {
	resetServer(t)

	propBody := map[string]any{
		"name":      "bad_type_prop",
		"label":     "Bad Type Prop",
		"type":      "invalid_type",
		"fieldType": "text",
		"groupName": "contactinformation",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", propBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

// TestError_CreatePropertyMissingRequiredFields verifies that creating a property
// with only a name (missing type, fieldType, etc.) returns 400.
func TestError_CreatePropertyMissingRequiredFields(t *testing.T) {
	resetServer(t)

	propBody := map[string]any{
		"name": "test",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", propBody)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d; body=%s", resp.StatusCode, string(b))
	}

	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

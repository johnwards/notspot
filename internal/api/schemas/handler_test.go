package schemas_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/schemas"
	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	mux := http.NewServeMux()
	schemas.RegisterRoutes(mux, db)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

func postJSON(t *testing.T, url string, v any) *http.Response {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func doRequest(t *testing.T, method, url string, v any) *http.Response {
	t.Helper()
	var body *bytes.Reader
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		body = bytes.NewReader(b)
	}
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, body)
	} else {
		req, err = http.NewRequest(method, url, http.NoBody)
	}
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func createSchema(t *testing.T, baseURL, name string) map[string]any {
	t.Helper()
	input := map[string]any{
		"name":                   name,
		"labels":                 map[string]any{"singular": name, "plural": name + "s"},
		"primaryDisplayProperty": "hs_object_id",
	}
	resp := postJSON(t, baseURL+"/crm/v3/schemas", input)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create schema status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	var result map[string]any
	decodeJSON(t, resp, &result)
	return result
}

func TestListSchemas_Empty(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/schemas")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.CollectionResponse
	decodeJSON(t, resp, &body)
	if len(body.Results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(body.Results))
	}
}

func TestCreateAndGetSchema(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	created := createSchema(t, srv.URL, "cars")

	if created["name"] != "cars" {
		t.Errorf("name = %v, want cars", created["name"])
	}
	if created["id"] != "2-1" {
		t.Errorf("id = %v, want 2-1", created["id"])
	}
	if created["fullyQualifiedName"] != "p0_cars" {
		t.Errorf("fqn = %v, want p0_cars", created["fullyQualifiedName"])
	}

	// Get by name
	resp, err := http.Get(srv.URL + "/crm/v3/schemas/cars")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("get status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got map[string]any
	decodeJSON(t, resp, &got)
	if got["name"] != "cars" {
		t.Errorf("name = %v, want cars", got["name"])
	}

	// Check default properties exist
	props, ok := got["properties"].([]any)
	if !ok {
		t.Fatal("properties is not an array")
	}
	if len(props) != 3 {
		t.Errorf("len(properties) = %d, want 3", len(props))
	}
}

func TestGetSchema_ByID(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")

	resp, err := http.Get(srv.URL + "/crm/v3/schemas/2-1")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got map[string]any
	decodeJSON(t, resp, &got)
	if got["name"] != "cars" {
		t.Errorf("name = %v, want cars", got["name"])
	}
}

func TestGetSchema_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/schemas/nonexistent")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestUpdateSchema(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")

	resp := doRequest(t, http.MethodPatch, srv.URL+"/crm/v3/schemas/cars",
		map[string]any{"labels": map[string]any{"singular": "Automobile", "plural": "Automobiles"}})
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("update status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated map[string]any
	decodeJSON(t, resp, &updated)
	labels, ok := updated["labels"].(map[string]any)
	if !ok {
		t.Fatal("labels is not a map")
	}
	if labels["singular"] != "Automobile" {
		t.Errorf("singular = %v, want Automobile", labels["singular"])
	}
}

func TestArchiveSchema(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")

	resp := doRequest(t, http.MethodDelete, srv.URL+"/crm/v3/schemas/cars", nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("archive status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	// Should no longer appear in list
	resp2, err := http.Get(srv.URL + "/crm/v3/schemas")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var list api.CollectionResponse
	decodeJSON(t, resp2, &list)
	if len(list.Results) != 0 {
		t.Errorf("len(results) = %d, want 0 after archive", len(list.Results))
	}
}

func TestCreateSchema_MissingFields(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	// Missing labels
	resp := postJSON(t, srv.URL+"/crm/v3/schemas", map[string]any{"name": "cars"})
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Missing name
	resp2 := postJSON(t, srv.URL+"/crm/v3/schemas", map[string]any{
		"labels": map[string]any{"singular": "Car", "plural": "Cars"},
	})
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp2.StatusCode, http.StatusBadRequest)
	}
}

func TestCreateSchema_Duplicate(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")

	resp := postJSON(t, srv.URL+"/crm/v3/schemas", map[string]any{
		"name":   "cars",
		"labels": map[string]any{"singular": "Car", "plural": "Cars"},
	})
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestSchemaAssociation_CreateAndDelete(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")
	createSchema(t, srv.URL, "drivers")

	// Create association
	resp := postJSON(t, srv.URL+"/crm/v3/schemas/cars/associations", map[string]any{
		"toObjectTypeId": "2-2",
		"name":           "car_to_driver",
	})
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create assoc status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var assoc map[string]any
	decodeJSON(t, resp, &assoc)
	assocID, ok := assoc["id"].(string)
	if !ok {
		t.Fatal("association id is not a string")
	}

	// Verify association appears on schema
	resp2, err := http.Get(srv.URL + "/crm/v3/schemas/cars")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var schema map[string]any
	decodeJSON(t, resp2, &schema)
	assocs, ok := schema["associations"].([]any)
	if !ok {
		t.Fatal("associations is not an array")
	}
	if len(assocs) != 1 {
		t.Fatalf("len(associations) = %d, want 1", len(assocs))
	}

	// Delete association
	resp3 := doRequest(t, http.MethodDelete, srv.URL+"/crm/v3/schemas/cars/associations/"+assocID, nil)
	_ = resp3.Body.Close()
	if resp3.StatusCode != http.StatusNoContent {
		t.Fatalf("delete assoc status = %d, want %d", resp3.StatusCode, http.StatusNoContent)
	}
}

func TestSchemaAssociation_MissingTarget(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")

	resp := postJSON(t, srv.URL+"/crm/v3/schemas/cars/associations", map[string]any{
		"name": "car_to_nothing",
	})
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestAlternatePathPrefix(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	// Create via /crm-object-schemas/v3/schemas
	input := map[string]any{
		"name":                   "cars",
		"labels":                 map[string]any{"singular": "Car", "plural": "Cars"},
		"primaryDisplayProperty": "hs_object_id",
	}
	resp := postJSON(t, srv.URL+"/crm-object-schemas/v3/schemas", input)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	_ = resp.Body.Close()

	// Get via /crm-object-schemas/v3/schemas/{name}
	resp2, err := http.Get(srv.URL + "/crm-object-schemas/v3/schemas/cars")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		_ = resp2.Body.Close()
		t.Fatalf("get status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	var got map[string]any
	decodeJSON(t, resp2, &got)
	if got["name"] != "cars" {
		t.Errorf("name = %v, want cars", got["name"])
	}

	// List via /crm/v3/schemas (cross-prefix)
	resp3, err := http.Get(srv.URL + "/crm/v3/schemas")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var list api.CollectionResponse
	decodeJSON(t, resp3, &list)
	if len(list.Results) != 1 {
		t.Errorf("len(results) = %d, want 1", len(list.Results))
	}
}

func TestListSchemas_MultipleWithProperties(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	createSchema(t, srv.URL, "cars")
	createSchema(t, srv.URL, "trucks")

	resp, err := http.Get(srv.URL + "/crm/v3/schemas")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}

	var list api.CollectionResponse
	decodeJSON(t, resp, &list)
	if len(list.Results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(list.Results))
	}

	// Each schema should have 3 default properties
	for _, r := range list.Results {
		schema, ok := r.(map[string]any)
		if !ok {
			t.Fatal("result is not a map")
		}
		props, ok := schema["properties"].([]any)
		if !ok {
			t.Fatal("properties is not an array")
		}
		if len(props) != 3 {
			t.Errorf("schema %v has %d properties, want 3", schema["name"], len(props))
		}
	}
}

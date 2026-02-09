package properties_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/properties"
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

	_, err := db.ExecContext(ctx,
		`INSERT INTO object_types (id, name, label_singular, label_plural, primary_display_property, is_custom, created_at, updated_at)
		 VALUES ('0-1', 'contacts', 'Contact', 'Contacts', 'email', FALSE, '2024-01-01T00:00:00.000Z', '2024-01-01T00:00:00.000Z')`)
	if err != nil {
		t.Fatalf("seed object type: %v", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO property_groups (object_type_id, name, label, display_order, archived)
		 VALUES ('0-1', 'contactinformation', 'Contact Information', 0, FALSE)`)
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}

	mux := http.NewServeMux()
	properties.RegisterRoutes(mux, db)

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

func TestListProperties_Empty(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/properties/contacts")
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

func TestCreateAndGetProperty(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	input := map[string]any{
		"name": "email", "label": "Email", "type": "string",
		"fieldType": "text", "groupName": "contactinformation",
	}
	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts", input)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var created map[string]any
	decodeJSON(t, resp, &created)
	if created["name"] != "email" {
		t.Errorf("name = %v, want email", created["name"])
	}

	resp2, err := http.Get(srv.URL + "/crm/v3/properties/contacts/email")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		_ = resp2.Body.Close()
		t.Fatalf("get status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	var got map[string]any
	decodeJSON(t, resp2, &got)
	if got["label"] != "Email" {
		t.Errorf("label = %v, want Email", got["label"])
	}
}

func TestUpdateProperty(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	input := map[string]any{
		"name": "email", "label": "Email", "type": "string",
		"fieldType": "text", "groupName": "contactinformation",
	}
	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts", input)
	_ = resp.Body.Close()

	resp2 := doRequest(t, http.MethodPatch, srv.URL+"/crm/v3/properties/contacts/email",
		map[string]any{"label": "Email Address"})
	if resp2.StatusCode != http.StatusOK {
		_ = resp2.Body.Close()
		t.Fatalf("update status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}

	var updated map[string]any
	decodeJSON(t, resp2, &updated)
	if updated["label"] != "Email Address" {
		t.Errorf("label = %v, want Email Address", updated["label"])
	}
}

func TestArchiveProperty(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	input := map[string]any{
		"name": "email", "label": "Email", "type": "string",
		"fieldType": "text", "groupName": "contactinformation",
	}
	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts", input)
	_ = resp.Body.Close()

	resp2 := doRequest(t, http.MethodDelete, srv.URL+"/crm/v3/properties/contacts/email", nil)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("archive status = %d, want %d", resp2.StatusCode, http.StatusNoContent)
	}

	resp3, err := http.Get(srv.URL + "/crm/v3/properties/contacts")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}

	var list api.CollectionResponse
	decodeJSON(t, resp3, &list)
	if len(list.Results) != 0 {
		t.Errorf("len(results) = %d, want 0 after archive", len(list.Results))
	}
}

func TestGetProperty_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/properties/contacts/nonexistent")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestCreateProperty_MissingFields(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts", map[string]any{"name": "email"})
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestBatchCreate(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	input := map[string]any{
		"inputs": []map[string]any{
			{"name": "email", "label": "Email", "type": "string", "fieldType": "text", "groupName": "contactinformation"},
			{"name": "phone", "label": "Phone", "type": "string", "fieldType": "phonenumber", "groupName": "contactinformation"},
		},
	}
	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts/batch/create", input)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var result api.CollectionResponse
	decodeJSON(t, resp, &result)
	if len(result.Results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(result.Results))
	}
}

func TestBatchRead(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	for _, name := range []string{"email", "phone"} {
		resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts", map[string]any{
			"name": name, "label": name, "type": "string", "fieldType": "text", "groupName": "contactinformation",
		})
		_ = resp.Body.Close()
	}

	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts/batch/read", map[string]any{
		"inputs": []map[string]any{{"name": "email"}, {"name": "phone"}},
	})
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result api.CollectionResponse
	decodeJSON(t, resp, &result)
	if len(result.Results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(result.Results))
	}
}

func TestBatchArchive(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	for _, name := range []string{"email", "phone"} {
		resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts", map[string]any{
			"name": name, "label": name, "type": "string", "fieldType": "text", "groupName": "contactinformation",
		})
		_ = resp.Body.Close()
	}

	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts/batch/archive", map[string]any{
		"inputs": []map[string]any{{"name": "email"}, {"name": "phone"}},
	})
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestListGroups(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/properties/contacts/groups")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body api.CollectionResponse
	decodeJSON(t, resp, &body)
	if len(body.Results) != 1 {
		t.Errorf("len(results) = %d, want 1", len(body.Results))
	}
}

func TestCreateAndGetGroup(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp := postJSON(t, srv.URL+"/crm/v3/properties/contacts/groups",
		map[string]any{"name": "customgroup", "label": "Custom Group"})
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	resp2, err := http.Get(srv.URL + "/crm/v3/properties/contacts/groups/customgroup")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}
}

func TestUpdateGroup(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp := doRequest(t, http.MethodPatch, srv.URL+"/crm/v3/properties/contacts/groups/contactinformation",
		map[string]any{"label": "Updated Contact Info"})
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("update status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var updated map[string]any
	decodeJSON(t, resp, &updated)
	if updated["label"] != "Updated Contact Info" {
		t.Errorf("label = %v, want Updated Contact Info", updated["label"])
	}
}

func TestArchiveGroup(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp := doRequest(t, http.MethodDelete, srv.URL+"/crm/v3/properties/contacts/groups/contactinformation", nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("archive status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestGetGroup_NotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/properties/contacts/groups/nonexistent")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestInvalidObjectType(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/properties/invalid_type")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

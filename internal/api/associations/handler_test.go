package associations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/associations"
	"github.com/johnwards/hubspot/internal/api/objects"
	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := seed.AssociationTypes(ctx, db); err != nil {
		t.Fatalf("seed assoc types: %v", err)
	}

	s := store.New(db)
	mux := http.NewServeMux()
	objects.RegisterRoutes(mux, s)
	associations.RegisterRoutes(mux, db)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

func createObject(t *testing.T, serverURL, objectType string) string {
	t.Helper()
	body := map[string]any{"properties": map[string]string{"test": "val"}}
	b, _ := json.Marshal(body)
	resp, err := http.Post(fmt.Sprintf("%s/crm/v3/objects/%s", serverURL, objectType), "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("create %s: %v", objectType, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %s: status %d", objectType, resp.StatusCode)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result.ID
}

func doRequest(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reqBody)
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

func TestAssociateDefaultEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	contactID := createObject(t, srv.URL, "contacts")
	companyID := createObject(t, srv.URL, "companies")

	resp := doRequest(t, "PUT",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/default/companies/%s", srv.URL, contactID, companyID), nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			ToObjectID string `json:"toObjectId"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].ToObjectID != companyID {
		t.Errorf("expected toObjectId %s, got %s", companyID, result.Results[0].ToObjectID)
	}
}

func TestAssociateWithLabelsEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	contactID := createObject(t, srv.URL, "contacts")
	companyID := createObject(t, srv.URL, "companies")

	body := []map[string]any{
		{"associationCategory": "HUBSPOT_DEFINED", "associationTypeId": 279},
	}
	resp := doRequest(t, "PUT",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/companies/%s", srv.URL, contactID, companyID), body)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetAssociationsEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	contactID := createObject(t, srv.URL, "contacts")
	companyID := createObject(t, srv.URL, "companies")

	// Create association.
	doRequest(t, "PUT",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/default/companies/%s", srv.URL, contactID, companyID), nil)

	// Get.
	resp := doRequest(t, "GET",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/companies", srv.URL, contactID), nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			ToObjectID string `json:"toObjectId"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
}

func TestRemoveAssociationsEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	contactID := createObject(t, srv.URL, "contacts")
	companyID := createObject(t, srv.URL, "companies")

	doRequest(t, "PUT",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/default/companies/%s", srv.URL, contactID, companyID), nil)

	resp := doRequest(t, "DELETE",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/companies/%s", srv.URL, contactID, companyID), nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify removed.
	resp2 := doRequest(t, "GET",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/companies", srv.URL, contactID), nil)
	defer func() { _ = resp2.Body.Close() }()

	var result struct {
		Results []any `json:"results"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 0 {
		t.Fatalf("expected 0 results after remove, got %d", len(result.Results))
	}
}

func TestListLabelsEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp := doRequest(t, "GET",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/labels", srv.URL), nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Results []struct {
			TypeID   int    `json:"typeId"`
			Category string `json:"category"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) < 2 {
		t.Fatalf("expected at least 2 labels, got %d", len(result.Results))
	}
}

func TestCreateLabelEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	body := map[string]string{"label": "Custom Label", "associationCategory": "USER_DEFINED"}
	resp := doRequest(t, "POST",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/labels", srv.URL), body)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result struct {
		TypeID   int    `json:"typeId"`
		Category string `json:"category"`
		Label    string `json:"label"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Label != "Custom Label" {
		t.Errorf("expected label 'Custom Label', got %q", result.Label)
	}
}

func TestDeleteLabelEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	// Create a label to delete.
	createBody := map[string]string{"label": "ToDelete", "associationCategory": "USER_DEFINED"}
	createResp := doRequest(t, "POST",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/labels", srv.URL), createBody)
	defer func() { _ = createResp.Body.Close() }()

	var created struct {
		TypeID int `json:"typeId"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&created)

	resp := doRequest(t, "DELETE",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/labels/%d", srv.URL, created.TypeID), nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestBatchAssociateDefaultEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	c1 := createObject(t, srv.URL, "contacts")
	co1 := createObject(t, srv.URL, "companies")

	body := map[string]any{
		"inputs": []map[string]any{
			{"from": map[string]string{"id": c1}, "to": map[string]string{"id": co1}},
		},
	}
	resp := doRequest(t, "POST",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/batch/associate/default", srv.URL), body)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Status  string `json:"status"`
		Results []any  `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != "COMPLETE" {
		t.Errorf("expected status COMPLETE, got %s", result.Status)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
}

func TestBatchReadEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	contactID := createObject(t, srv.URL, "contacts")
	companyID := createObject(t, srv.URL, "companies")

	// Create association first.
	doRequest(t, "PUT",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/default/companies/%s", srv.URL, contactID, companyID), nil)

	body := map[string]any{
		"inputs": []map[string]string{{"id": contactID}},
	}
	resp := doRequest(t, "POST",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/batch/read", srv.URL), body)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBatchArchiveEndpoint(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	contactID := createObject(t, srv.URL, "contacts")
	companyID := createObject(t, srv.URL, "companies")

	doRequest(t, "PUT",
		fmt.Sprintf("%s/crm/v4/objects/contacts/%s/associations/default/companies/%s", srv.URL, contactID, companyID), nil)

	body := map[string]any{
		"inputs": []map[string]any{
			{"from": map[string]string{"id": contactID}, "to": map[string]string{"id": companyID}},
		},
	}
	resp := doRequest(t, "POST",
		fmt.Sprintf("%s/crm/v4/associations/contacts/companies/batch/archive", srv.URL), body)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

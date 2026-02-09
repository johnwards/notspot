package objects_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/objects"
	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
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
	objects.RegisterRoutes(mux, s)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

// createContact is a test helper that creates a contact and returns the decoded object.
func createContact(t *testing.T, srv *httptest.Server, props string) domain.Object {
	t.Helper()
	body := `{"properties":` + props + `}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var obj domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return obj
}

func TestCreateEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := `{"properties":{"email":"create@example.com","firstname":"Test"}}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var obj domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if obj.ID == "" {
		t.Error("expected non-empty ID")
	}
	if obj.Properties["email"] != "create@example.com" {
		t.Errorf("expected email=create@example.com, got %s", obj.Properties["email"])
	}
}

func TestGetEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	created := createContact(t, srv, `{"email":"get@example.com","firstname":"Get"}`)

	resp, err := http.Get(srv.URL + "/crm/v3/objects/contacts/" + created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var obj domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if obj.ID != created.ID {
		t.Errorf("expected ID=%s, got %s", created.ID, obj.ID)
	}
}

func TestGetWithProperties(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	created := createContact(t, srv, `{"email":"props@example.com","firstname":"Props"}`)

	resp, err := http.Get(srv.URL + "/crm/v3/objects/contacts/" + created.ID + "?properties=email,firstname")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var obj domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if obj.Properties["email"] != "props@example.com" {
		t.Errorf("expected email in response")
	}
	if obj.Properties["firstname"] != "Props" {
		t.Errorf("expected firstname in response")
	}
}

func TestListEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	for _, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		createContact(t, srv, `{"email":"`+email+`"}`)
	}

	resp, err := http.Get(srv.URL + "/crm/v3/objects/contacts?limit=2")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var listResp api.CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(listResp.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(listResp.Results))
	}
	if listResp.Paging == nil || listResp.Paging.Next == nil {
		t.Fatal("expected paging.next")
	}

	// Page 2.
	resp2, err := http.Get(srv.URL + "/crm/v3/objects/contacts?limit=2&after=" + listResp.Paging.Next.After)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	var listResp2 api.CollectionResponse
	if err := json.NewDecoder(resp2.Body).Decode(&listResp2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(listResp2.Results) != 1 {
		t.Fatalf("expected 1 result on page 2, got %d", len(listResp2.Results))
	}
}

func TestUpdateEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	created := createContact(t, srv, `{"email":"upd@example.com","firstname":"Old"}`)

	patchBody := `{"properties":{"firstname":"New"}}`
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/crm/v3/objects/contacts/"+created.ID, bytes.NewBufferString(patchBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("patch: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestArchiveEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	created := createContact(t, srv, `{"email":"del@example.com"}`)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/crm/v3/objects/contacts/"+created.ID, http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestNotFoundObjectType(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/objects/nonexistent")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestBatchCreateEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := `{"inputs":[{"properties":{"email":"b1@example.com"}},{"properties":{"email":"b2@example.com"}}]}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/batch/create", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("batch create: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result domain.BatchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
}

func TestMergeEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	primary := createContact(t, srv, `{"email":"primary@example.com","firstname":"Primary"}`)
	merged := createContact(t, srv, `{"email":"merged@example.com","lastname":"Merged"}`)

	mergeBody := `{"primaryObjectId":"` + primary.ID + `","objectIdToMerge":"` + merged.ID + `"}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/merge", "application/json", bytes.NewBufferString(mergeBody))
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.ID != primary.ID {
		t.Errorf("expected primary ID %s, got %s", primary.ID, result.ID)
	}
}

func TestCreateWithObjectTypeID(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := `{"properties":{"email":"byid@example.com"}}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/0-1", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create by type id: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

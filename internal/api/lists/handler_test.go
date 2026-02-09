package lists_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/lists"
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
	lists.RegisterRoutes(mux, s)
	objects.RegisterRoutes(mux, s)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

func createList(t *testing.T, srv *httptest.Server, body string) domain.List {
	t.Helper()
	resp, err := http.Post(srv.URL+"/crm/v3/lists", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create list: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var list domain.List
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return list
}

func createContact(t *testing.T, srv *httptest.Server, email string) string {
	t.Helper()
	body := `{"properties":{"email":"` + email + `"}}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create contact: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var obj domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return obj.ID
}

func TestCreateListEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := `{"name":"Test List","objectTypeId":"0-1","processingType":"MANUAL"}`
	list := createList(t, srv, body)

	if list.ListID == "" {
		t.Error("expected non-empty listId")
	}
	if list.Name != "Test List" {
		t.Errorf("expected name=Test List, got %s", list.Name)
	}
	if list.ProcessingType != "MANUAL" {
		t.Errorf("expected processingType=MANUAL, got %s", list.ProcessingType)
	}
}

func TestGetListEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Get Test","objectTypeId":"0-1","processingType":"MANUAL"}`)

	resp, err := http.Get(srv.URL + "/crm/v3/lists/" + list.ListID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var got domain.List
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Get Test" {
		t.Errorf("expected name=Get Test, got %s", got.Name)
	}
}

func TestDeleteListEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Delete Test","objectTypeId":"0-1","processingType":"MANUAL"}`)

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/crm/v3/lists/"+list.ListID, http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify it's gone.
	resp2, err := http.Get(srv.URL + "/crm/v3/lists/" + list.ListID)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp2.StatusCode)
	}
}

func TestRestoreListEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Restore Test","objectTypeId":"0-1","processingType":"MANUAL"}`)

	// Delete.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/crm/v3/lists/"+list.ListID, http.NoBody)
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Restore.
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/restore", http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify it's back.
	resp2, _ := http.Get(srv.URL + "/crm/v3/lists/" + list.ListID)
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after restore, got %d", resp2.StatusCode)
	}
}

func TestUpdateListNameEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Old Name","objectTypeId":"0-1","processingType":"MANUAL"}`)

	body := `{"name":"New Name"}`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/update-list-name", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update name: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var updated domain.List
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name=New Name, got %s", updated.Name)
	}
}

func TestSearchListsEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createList(t, srv, `{"name":"Alpha List","objectTypeId":"0-1","processingType":"MANUAL"}`)
	createList(t, srv, `{"name":"Beta List","objectTypeId":"0-1","processingType":"MANUAL"}`)

	body := `{"query":"Alpha","count":10}`
	resp, err := http.Post(srv.URL+"/crm/v3/lists/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Lists   []domain.List `json:"lists"`
		Total   int           `json:"total"`
		HasMore bool          `json:"hasMore"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Lists) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Lists))
	}
	if result.Lists[0].Name != "Alpha List" {
		t.Errorf("expected Alpha List, got %s", result.Lists[0].Name)
	}
}

func TestAddMembersEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Members Test","objectTypeId":"0-1","processingType":"MANUAL"}`)
	contactID := createContact(t, srv, "member@test.com")

	body, _ := json.Marshal([]string{contactID})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("add members: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result["recordIdsAdded"]) != 1 {
		t.Errorf("expected 1 recordIdsAdded, got %d", len(result["recordIdsAdded"]))
	}
	// Verify HubSpot typo field is also present.
	if len(result["recordsIdsAdded"]) != 1 {
		t.Errorf("expected 1 recordsIdsAdded (typo field), got %d", len(result["recordsIdsAdded"]))
	}
}

func TestAddAndRemoveMembersEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Add Remove Test","objectTypeId":"0-1","processingType":"MANUAL"}`)
	c1 := createContact(t, srv, "c1@test.com")
	c2 := createContact(t, srv, "c2@test.com")

	// Add c1 first.
	body, _ := json.Marshal([]string{c1})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Add c2 and remove c1.
	addRemoveBody := `{"recordIdsToAdd":["` + c2 + `"],"recordIdsToRemove":["` + c1 + `"]}`
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships/add-and-remove", bytes.NewBufferString(addRemoveBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("add-and-remove: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string][]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result["recordIdsAdded"]) != 1 {
		t.Errorf("expected 1 added, got %d", len(result["recordIdsAdded"]))
	}
	if len(result["recordIdsRemoved"]) != 1 {
		t.Errorf("expected 1 removed, got %d", len(result["recordIdsRemoved"]))
	}
}

func TestDynamicListMembershipEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Dynamic Test","objectTypeId":"0-1","processingType":"DYNAMIC"}`)

	body := `["1"]`
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships/add", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for dynamic list mutation, got %d", resp.StatusCode)
	}
}

func TestDuplicateListNameEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createList(t, srv, `{"name":"Same Name","objectTypeId":"0-1","processingType":"MANUAL"}`)

	resp, err := http.Post(srv.URL+"/crm/v3/lists", "application/json", bytes.NewBufferString(`{"name":"Same Name","objectTypeId":"0-1","processingType":"MANUAL"}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestGetMembershipsEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Memberships Test","objectTypeId":"0-1","processingType":"MANUAL"}`)
	c1 := createContact(t, srv, "m1@test.com")
	c2 := createContact(t, srv, "m2@test.com")

	// Add both.
	body, _ := json.Marshal([]string{c1, c2})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Get memberships.
	resp2, err := http.Get(srv.URL + "/crm/v3/lists/" + list.ListID + "/memberships")
	if err != nil {
		t.Fatalf("get memberships: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}

	var result struct {
		Results []domain.ListMembership `json:"results"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 memberships, got %d", len(result.Results))
	}
}

func TestRemoveAllMembersEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	list := createList(t, srv, `{"name":"Remove All Test","objectTypeId":"0-1","processingType":"MANUAL"}`)
	c1 := createContact(t, srv, "ra1@test.com")

	// Add member.
	body, _ := json.Marshal([]string{c1})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	_ = resp.Body.Close()

	// Remove all.
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/crm/v3/lists/"+list.ListID+"/memberships", http.NoBody)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("remove all: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// Verify list is empty.
	resp2, _ := http.Get(srv.URL + "/crm/v3/lists/" + list.ListID)
	defer func() { _ = resp2.Body.Close() }()
	var got domain.List
	_ = json.NewDecoder(resp2.Body).Decode(&got)
	if got.Size != 0 {
		t.Errorf("expected size=0, got %d", got.Size)
	}
}

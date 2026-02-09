package objects_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/johnwards/hubspot/internal/domain"
)

func TestSearchEndpoint(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	// Create test contacts.
	createContact(t, srv, `{"email":"search1@example.com","firstname":"Alice"}`)
	createContact(t, srv, `{"email":"search2@example.com","firstname":"Bob"}`)
	createContact(t, srv, `{"email":"search3@other.com","firstname":"Charlie"}`)

	// Search with EQ filter.
	body := `{"filterGroups":[{"filters":[{"propertyName":"email","operator":"EQ","value":"search1@example.com"}]}]}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result domain.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
}

func TestSearchEndpointNoFilters(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createContact(t, srv, `{"email":"all1@example.com"}`)
	createContact(t, srv, `{"email":"all2@example.com"}`)

	body := `{}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result domain.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
}

func TestSearchEndpointPagination(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	for _, email := range []string{"p1@example.com", "p2@example.com", "p3@example.com"} {
		createContact(t, srv, `{"email":"`+email+`"}`)
	}

	body := `{"limit":2}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result domain.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 3 {
		t.Errorf("expected total=3, got %d", result.Total)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Paging == nil {
		t.Fatal("expected paging info")
	}

	// Second page.
	body2 := `{"limit":2,"after":"` + result.Paging.Next.After + `"}`
	resp2, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body2))
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	var result2 domain.SearchResult
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result2.Results) != 1 {
		t.Errorf("expected 1 result on page 2, got %d", len(result2.Results))
	}
}

func TestSearchEndpointWithQuery(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createContact(t, srv, `{"email":"findme@example.com","firstname":"Find"}`)
	createContact(t, srv, `{"email":"skip@other.com","firstname":"Skip"}`)

	body := `{"query":"findme"}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result domain.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
}

func TestSearchEndpointValidationError(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	// Invalid operator.
	body := `{"filterGroups":[{"filters":[{"propertyName":"email","operator":"INVALID","value":"test"}]}]}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSearchEndpointNotFoundType(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := `{}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/nonexistent/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSearchEndpointWithProperties(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createContact(t, srv, `{"email":"props@example.com","firstname":"Props"}`)

	body := `{"properties":["email","firstname"]}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result domain.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0].Properties["email"] != "props@example.com" {
		t.Errorf("expected email in response properties")
	}
	if result.Results[0].Properties["firstname"] != "Props" {
		t.Errorf("expected firstname in response properties")
	}
}

func TestSearchEndpointWithSort(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	createContact(t, srv, `{"email":"charlie@example.com","firstname":"Charlie"}`)
	createContact(t, srv, `{"email":"alice@example.com","firstname":"Alice"}`)
	createContact(t, srv, `{"email":"bob@example.com","firstname":"Bob"}`)

	body := `{"sorts":[{"propertyName":"firstname","direction":"ASCENDING"}],"properties":["firstname"]}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts/search", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result domain.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}
	if result.Results[0].Properties["firstname"] != "Alice" {
		t.Errorf("expected first=Alice, got %s", result.Results[0].Properties["firstname"])
	}
	if result.Results[2].Properties["firstname"] != "Charlie" {
		t.Errorf("expected last=Charlie, got %s", result.Results[2].Properties["firstname"])
	}
}

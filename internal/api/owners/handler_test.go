package owners_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/owners"
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
	if err := seed.Owners(ctx, db); err != nil {
		t.Fatalf("seed owners: %v", err)
	}

	s := store.New(db)
	mux := http.NewServeMux()
	owners.RegisterRoutes(mux, s)

	handler := api.Chain(mux, api.RequestID())
	return httptest.NewServer(handler)
}

func TestListOwners(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/owners")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result api.CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 owners, got %d", len(result.Results))
	}
}

func TestGetOwner(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/owners/1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var owner store.Owner
	if err := json.NewDecoder(resp.Body).Decode(&owner); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if owner.Email != "admin@example.com" {
		t.Errorf("expected email=admin@example.com, got %s", owner.Email)
	}
}

func TestGetOwnerNotFound(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/owners/999")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestListOwnersPagination(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/crm/v3/owners?limit=2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result api.CollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 owners, got %d", len(result.Results))
	}
	if result.Paging == nil || result.Paging.Next == nil {
		t.Fatal("expected paging info")
	}

	// Fetch next page.
	resp2, err := http.Get(srv.URL + "/crm/v3/owners?limit=2&after=" + result.Paging.Next.After)
	if err != nil {
		t.Fatalf("get page 2: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	var result2 api.CollectionResponse
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(result2.Results) != 1 {
		t.Errorf("expected 1 owner on page 2, got %d", len(result2.Results))
	}
}

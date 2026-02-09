package store_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

var _ store.ImportStore = (*store.SQLiteImportStore)(nil)

func setupImportStore(t *testing.T) *store.SQLiteImportStore {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return store.NewSQLiteImportStore(db)
}

func TestImportCreate(t *testing.T) {
	s := setupImportStore(t)
	ctx := context.Background()

	reqJSON := json.RawMessage(`{"name":"Test Import"}`)
	imp, err := s.Create(ctx, "Test Import", reqJSON)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if imp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if imp.Name != "Test Import" {
		t.Errorf("expected name=Test Import, got %s", imp.Name)
	}
	if imp.State != "STARTED" {
		t.Errorf("expected state=STARTED, got %s", imp.State)
	}
}

func TestImportGet(t *testing.T) {
	s := setupImportStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, "Get Test", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Name != "Get Test" {
		t.Errorf("expected name=Get Test, got %s", got.Name)
	}
}

func TestImportGetNotFound(t *testing.T) {
	s := setupImportStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "999")
	if err == nil {
		t.Fatal("expected error for nonexistent import")
	}
}

func TestImportList(t *testing.T) {
	s := setupImportStore(t)
	ctx := context.Background()

	for i := range 3 {
		_, err := s.Create(ctx, "Import "+string(rune('0'+i)), json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	imports, hasMore, _, err := s.List(ctx, 100, "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(imports))
	}
	if hasMore {
		t.Error("expected hasMore=false")
	}
}

func TestImportUpdateState(t *testing.T) {
	s := setupImportStore(t)
	ctx := context.Background()

	imp, err := s.Create(ctx, "State Test", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	meta := json.RawMessage(`{"objectLists":[]}`)
	if err := s.UpdateState(ctx, imp.ID, "DONE", meta); err != nil {
		t.Fatalf("update state: %v", err)
	}

	got, err := s.Get(ctx, imp.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != "DONE" {
		t.Errorf("expected state=DONE, got %s", got.State)
	}
}

func TestImportAddAndGetErrors(t *testing.T) {
	s := setupImportStore(t)
	ctx := context.Background()

	imp, err := s.Create(ctx, "Error Test", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.AddError(ctx, imp.ID, "INVALID_ROW", "bad data", "xyz", "0-1", 5); err != nil {
		t.Fatalf("add error: %v", err)
	}
	if err := s.AddError(ctx, imp.ID, "OBJECT_CREATE_ERROR", "duplicate", "", "0-1", 10); err != nil {
		t.Fatalf("add error: %v", err)
	}

	errs, err := s.GetErrors(ctx, imp.ID)
	if err != nil {
		t.Fatalf("get errors: %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
	if errs[0].ErrorType != "INVALID_ROW" {
		t.Errorf("expected first error type=INVALID_ROW, got %s", errs[0].ErrorType)
	}
	if errs[0].InvalidValue != "xyz" {
		t.Errorf("expected invalidValue=xyz, got %s", errs[0].InvalidValue)
	}
	if errs[0].LineNumber != 5 {
		t.Errorf("expected lineNumber=5, got %d", errs[0].LineNumber)
	}
}

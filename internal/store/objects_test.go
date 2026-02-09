package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

// Verify interface compliance at compile time.
var _ store.ObjectStore = (*store.SQLiteObjectStore)(nil)

func setupStore(t *testing.T) *store.SQLiteObjectStore {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	return store.NewSQLiteObjectStore(db)
}

func TestCreateAndGet(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	obj, err := s.Create(ctx, "contacts", map[string]string{
		"email":     "test@example.com",
		"firstname": "Test",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if obj.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if obj.Properties["email"] != "test@example.com" {
		t.Errorf("expected email=test@example.com, got %s", obj.Properties["email"])
	}
	if obj.Properties["hs_object_id"] != obj.ID {
		t.Errorf("expected hs_object_id=%s, got %s", obj.ID, obj.Properties["hs_object_id"])
	}

	// Get with default props only.
	got, err := s.Get(ctx, "contacts", obj.ID, nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Properties["hs_object_id"] != obj.ID {
		t.Errorf("expected hs_object_id in default props")
	}
	// email should not be in defaults.
	if _, ok := got.Properties["email"]; ok {
		t.Error("expected email NOT in default properties")
	}

	// Get with specific props.
	got, err = s.Get(ctx, "contacts", obj.ID, []string{"email", "firstname"})
	if err != nil {
		t.Fatalf("get with props: %v", err)
	}
	if got.Properties["email"] != "test@example.com" {
		t.Errorf("expected email in requested properties")
	}
}

func TestResolveObjectTypeByIDViaCreate(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	// Creating via type ID "0-1" should work just like "contacts".
	obj, err := s.Create(ctx, "0-1", map[string]string{"email": "resolve@example.com"})
	if err != nil {
		t.Fatalf("create via type id: %v", err)
	}
	if obj.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	// Verify the object can be retrieved via "contacts" too.
	got, err := s.Get(ctx, "contacts", obj.ID, []string{"email"})
	if err != nil {
		t.Fatalf("get via name: %v", err)
	}
	if got.Properties["email"] != "resolve@example.com" {
		t.Errorf("expected email=resolve@example.com, got %s", got.Properties["email"])
	}
}

func TestCreateNonexistentType(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	_, err := s.Create(ctx, "nonexistent", map[string]string{})
	if err == nil {
		t.Fatal("expected error for nonexistent type")
	}
}

func TestList(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	// Create 3 contacts.
	for i := range 3 {
		_, err := s.Create(ctx, "contacts", map[string]string{
			"email": "user" + string(rune('0'+i)) + "@example.com",
		})
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	// List with limit 2.
	page, err := s.List(ctx, "contacts", domain.ListOpts{Limit: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(page.Results))
	}
	if !page.HasMore {
		t.Error("expected HasMore=true")
	}

	// Next page.
	page2, err := s.List(ctx, "contacts", domain.ListOpts{Limit: 2, After: page.After})
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(page2.Results) != 1 {
		t.Fatalf("expected 1 result on page 2, got %d", len(page2.Results))
	}
	if page2.HasMore {
		t.Error("expected HasMore=false on last page")
	}
}

func TestUpdate(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	obj, err := s.Create(ctx, "contacts", map[string]string{
		"email":     "old@example.com",
		"firstname": "Old",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := s.Update(ctx, "contacts", obj.ID, map[string]string{
		"firstname": "New",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify new value.
	got, err := s.Get(ctx, "contacts", updated.ID, []string{"firstname", "email"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Properties["firstname"] != "New" {
		t.Errorf("expected firstname=New, got %s", got.Properties["firstname"])
	}
	if got.Properties["email"] != "old@example.com" {
		t.Errorf("expected email unchanged, got %s", got.Properties["email"])
	}
}

func TestArchive(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	obj, err := s.Create(ctx, "contacts", map[string]string{"email": "del@example.com"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.Archive(ctx, "contacts", obj.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	// Should not appear in non-archived list.
	page, err := s.List(ctx, "contacts", domain.ListOpts{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Results) != 0 {
		t.Errorf("expected 0 non-archived, got %d", len(page.Results))
	}

	// Should appear in archived list.
	page, err = s.List(ctx, "contacts", domain.ListOpts{Archived: true})
	if err != nil {
		t.Fatalf("list archived: %v", err)
	}
	if len(page.Results) != 1 {
		t.Errorf("expected 1 archived, got %d", len(page.Results))
	}
}

func TestGetByProperty(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	_, err := s.Create(ctx, "contacts", map[string]string{
		"email":     "lookup@example.com",
		"firstname": "Lookup",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.GetByProperty(ctx, "contacts", "email", "lookup@example.com", []string{"firstname"})
	if err != nil {
		t.Fatalf("get by property: %v", err)
	}
	if got.Properties["firstname"] != "Lookup" {
		t.Errorf("expected firstname=Lookup, got %s", got.Properties["firstname"])
	}
}

func TestBatchCreate(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	inputs := []domain.CreateInput{
		{Properties: map[string]string{"email": "batch1@example.com"}},
		{Properties: map[string]string{"email": "batch2@example.com"}},
	}

	result, err := s.BatchCreate(ctx, "contacts", inputs)
	if err != nil {
		t.Fatalf("batch create: %v", err)
	}
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Status != "COMPLETE" {
		t.Errorf("expected status COMPLETE, got %s", result.Status)
	}
}

func TestBatchRead(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	obj1, _ := s.Create(ctx, "contacts", map[string]string{"email": "r1@example.com"})
	obj2, _ := s.Create(ctx, "contacts", map[string]string{"email": "r2@example.com"})

	result, err := s.BatchRead(ctx, "contacts", []string{obj1.ID, obj2.ID, "999"}, nil, "")
	if err != nil {
		t.Fatalf("batch read: %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.NumErrors != 1 {
		t.Errorf("expected 1 error (missing id), got %d", result.NumErrors)
	}
}

func TestBatchUpdate(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	obj, _ := s.Create(ctx, "contacts", map[string]string{"email": "upd@example.com", "firstname": "Old"})

	result, err := s.BatchUpdate(ctx, "contacts", []domain.UpdateInput{
		{ID: obj.ID, Properties: map[string]string{"firstname": "New"}},
	})
	if err != nil {
		t.Fatalf("batch update: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
}

func TestBatchArchive(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	obj1, _ := s.Create(ctx, "contacts", map[string]string{"email": "a1@example.com"})
	obj2, _ := s.Create(ctx, "contacts", map[string]string{"email": "a2@example.com"})

	err := s.BatchArchive(ctx, "contacts", []string{obj1.ID, obj2.ID})
	if err != nil {
		t.Fatalf("batch archive: %v", err)
	}

	page, _ := s.List(ctx, "contacts", domain.ListOpts{})
	if len(page.Results) != 0 {
		t.Errorf("expected 0 non-archived, got %d", len(page.Results))
	}
}

func TestMerge(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	primary, _ := s.Create(ctx, "contacts", map[string]string{
		"email":     "primary@example.com",
		"firstname": "Primary",
	})
	merged, _ := s.Create(ctx, "contacts", map[string]string{
		"email":    "merged@example.com",
		"lastname": "Merged",
	})

	result, err := s.Merge(ctx, "contacts", primary.ID, merged.ID)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	// Primary should have the merged object's unique properties.
	got, err := s.Get(ctx, "contacts", result.ID, []string{"email", "firstname", "lastname", "hs_merged_object_ids"})
	if err != nil {
		t.Fatalf("get merged: %v", err)
	}
	if got.Properties["firstname"] != "Primary" {
		t.Errorf("expected primary's firstname, got %s", got.Properties["firstname"])
	}
	if got.Properties["lastname"] != "Merged" {
		t.Errorf("expected merged's lastname, got %s", got.Properties["lastname"])
	}
	if got.Properties["hs_merged_object_ids"] != merged.ID {
		t.Errorf("expected hs_merged_object_ids=%s, got %s", merged.ID, got.Properties["hs_merged_object_ids"])
	}

	// Merged object should be archived.
	mergedObj, err := s.Get(ctx, "contacts", merged.ID, nil)
	if err != nil {
		t.Fatalf("get archived merged: %v", err)
	}
	if !mergedObj.Archived {
		t.Error("expected merged object to be archived")
	}
}

func TestCreateWithObjectTypeID(t *testing.T) {
	s := setupStore(t)
	ctx := context.Background()

	// Create using type ID instead of name.
	obj, err := s.Create(ctx, "0-1", map[string]string{
		"email": "byid@example.com",
	})
	if err != nil {
		t.Fatalf("create by type id: %v", err)
	}
	if obj.ID == "" {
		t.Fatal("expected non-empty ID")
	}
}

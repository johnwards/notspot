package store_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

// Verify interface compliance at compile time.
var _ store.ListStore = (*store.SQLiteListStore)(nil)

func setupListStore(t *testing.T) (*store.SQLiteListStore, *store.SQLiteObjectStore) {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}

	return store.NewSQLiteListStore(db), store.NewSQLiteObjectStore(db)
}

func TestListCreateAndGet(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	list, err := s.Create(ctx, "My Test List", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if list.ListID == "" {
		t.Fatal("expected non-empty ListID")
	}
	if list.Name != "My Test List" {
		t.Errorf("expected name=My Test List, got %s", list.Name)
	}
	if list.ObjectTypeId != "0-1" {
		t.Errorf("expected objectTypeId=0-1, got %s", list.ObjectTypeId)
	}
	if list.ProcessingType != "MANUAL" {
		t.Errorf("expected processingType=MANUAL, got %s", list.ProcessingType)
	}
	if list.ProcessingStatus != "COMPLETE" {
		t.Errorf("expected processingStatus=COMPLETE, got %s", list.ProcessingStatus)
	}
	if list.ListVersion != 1 {
		t.Errorf("expected listVersion=1, got %d", list.ListVersion)
	}

	got, err := s.Get(ctx, list.ListID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "My Test List" {
		t.Errorf("expected name=My Test List, got %s", got.Name)
	}
}

func TestListCreateWithFilterBranch(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	fb := json.RawMessage(`{"filterBranchType":"AND","filters":[]}`)
	list, err := s.Create(ctx, "Dynamic List", "0-1", "DYNAMIC", fb)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.Get(ctx, list.ListID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if string(got.FilterBranch) != string(fb) {
		t.Errorf("expected filterBranch=%s, got %s", fb, got.FilterBranch)
	}
}

func TestListCreateDuplicateName(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	_, err := s.Create(ctx, "Unique Name", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}

	_, err = s.Create(ctx, "Unique Name", "0-1", "MANUAL", nil)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestListDelete(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	list, err := s.Create(ctx, "Delete Me", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.Delete(ctx, list.ListID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = s.Get(ctx, list.ListID)
	if err == nil {
		t.Fatal("expected error for deleted list")
	}
}

func TestListRestore(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	list, err := s.Create(ctx, "Restore Me", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.Delete(ctx, list.ListID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if err := s.Restore(ctx, list.ListID); err != nil {
		t.Fatalf("restore: %v", err)
	}

	got, err := s.Get(ctx, list.ListID)
	if err != nil {
		t.Fatalf("get after restore: %v", err)
	}
	if got.Name != "Restore Me" {
		t.Errorf("expected name=Restore Me, got %s", got.Name)
	}
}

func TestListUpdateName(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	list, err := s.Create(ctx, "Old Name", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := s.UpdateName(ctx, list.ListID, "New Name")
	if err != nil {
		t.Fatalf("update name: %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("expected name=New Name, got %s", updated.Name)
	}
	if updated.ListVersion != 2 {
		t.Errorf("expected listVersion=2, got %d", updated.ListVersion)
	}
}

func TestListUpdateFilters(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	list, err := s.Create(ctx, "Filter List", "0-1", "DYNAMIC", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	fb := json.RawMessage(`{"filterBranchType":"OR","filters":[{"type":"string"}]}`)
	updated, err := s.UpdateFilters(ctx, list.ListID, fb)
	if err != nil {
		t.Fatalf("update filters: %v", err)
	}

	if string(updated.FilterBranch) != string(fb) {
		t.Errorf("expected filterBranch=%s, got %s", fb, updated.FilterBranch)
	}
	if updated.ListVersion != 2 {
		t.Errorf("expected listVersion=2, got %d", updated.ListVersion)
	}
}

func TestListSearch(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		name := "Search List " + string(rune('A'+i))
		_, err := s.Create(ctx, name, "0-1", "MANUAL", nil)
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	page, err := s.Search(ctx, domain.ListSearchOpts{Query: "Search List", Limit: 3})
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(page.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(page.Results))
	}
	if !page.HasMore {
		t.Error("expected hasMore=true")
	}
	if page.TotalCount != 5 {
		t.Errorf("expected totalCount=5, got %d", page.TotalCount)
	}
}

func TestListGetMultiple(t *testing.T) {
	s, _ := setupListStore(t)
	ctx := context.Background()

	l1, _ := s.Create(ctx, "Multi 1", "0-1", "MANUAL", nil)
	l2, _ := s.Create(ctx, "Multi 2", "0-1", "MANUAL", nil)
	_, _ = s.Create(ctx, "Multi 3", "0-1", "MANUAL", nil)

	lists, err := s.GetMultiple(ctx, []string{l1.ListID, l2.ListID})
	if err != nil {
		t.Fatalf("get multiple: %v", err)
	}
	if len(lists) != 2 {
		t.Errorf("expected 2 lists, got %d", len(lists))
	}
}

func TestListMemberships(t *testing.T) {
	ls, os := setupListStore(t)
	ctx := context.Background()

	list, err := ls.Create(ctx, "Members List", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create list: %v", err)
	}

	// Create some objects to be members.
	obj1, err := os.Create(ctx, "contacts", map[string]string{"email": "a@test.com"})
	if err != nil {
		t.Fatalf("create obj1: %v", err)
	}
	obj2, err := os.Create(ctx, "contacts", map[string]string{"email": "b@test.com"})
	if err != nil {
		t.Fatalf("create obj2: %v", err)
	}

	// Add members.
	added, err := ls.AddMembers(ctx, list.ListID, []string{obj1.ID, obj2.ID})
	if err != nil {
		t.Fatalf("add members: %v", err)
	}
	if len(added) != 2 {
		t.Errorf("expected 2 added, got %d", len(added))
	}

	// Verify size.
	got, err := ls.Get(ctx, list.ListID)
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if got.Size != 2 {
		t.Errorf("expected size=2, got %d", got.Size)
	}

	// Get memberships.
	page, err := ls.GetMemberships(ctx, list.ListID, "", 100)
	if err != nil {
		t.Fatalf("get memberships: %v", err)
	}
	if len(page.Results) != 2 {
		t.Errorf("expected 2 memberships, got %d", len(page.Results))
	}

	// Remove one member.
	removed, err := ls.RemoveMembers(ctx, list.ListID, []string{obj1.ID})
	if err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if len(removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(removed))
	}

	// Check size again.
	got, err = ls.Get(ctx, list.ListID)
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if got.Size != 1 {
		t.Errorf("expected size=1, got %d", got.Size)
	}

	// Remove all members.
	if err := ls.RemoveAllMembers(ctx, list.ListID); err != nil {
		t.Fatalf("remove all: %v", err)
	}

	got, err = ls.Get(ctx, list.ListID)
	if err != nil {
		t.Fatalf("get list: %v", err)
	}
	if got.Size != 0 {
		t.Errorf("expected size=0, got %d", got.Size)
	}
}

func TestDynamicListMembershipMutation(t *testing.T) {
	ls, _ := setupListStore(t)
	ctx := context.Background()

	list, err := ls.Create(ctx, "Dynamic List", "0-1", "DYNAMIC", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = ls.AddMembers(ctx, list.ListID, []string{"1"})
	if err == nil {
		t.Fatal("expected error for adding members to dynamic list")
	}

	_, err = ls.RemoveMembers(ctx, list.ListID, []string{"1"})
	if err == nil {
		t.Fatal("expected error for removing members from dynamic list")
	}

	err = ls.RemoveAllMembers(ctx, list.ListID)
	if err == nil {
		t.Fatal("expected error for removing all members from dynamic list")
	}
}

func TestMembershipPagination(t *testing.T) {
	ls, os := setupListStore(t)
	ctx := context.Background()

	list, err := ls.Create(ctx, "Paginated List", "0-1", "MANUAL", nil)
	if err != nil {
		t.Fatalf("create list: %v", err)
	}

	// Create 5 objects and add them.
	var ids []string
	for i := 0; i < 5; i++ {
		obj, err := os.Create(ctx, "contacts", map[string]string{"email": string(rune('a'+i)) + "@test.com"})
		if err != nil {
			t.Fatalf("create obj %d: %v", i, err)
		}
		ids = append(ids, obj.ID)
	}

	_, err = ls.AddMembers(ctx, list.ListID, ids)
	if err != nil {
		t.Fatalf("add members: %v", err)
	}

	// Get first page of 2.
	page, err := ls.GetMemberships(ctx, list.ListID, "", 2)
	if err != nil {
		t.Fatalf("get page 1: %v", err)
	}
	if len(page.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(page.Results))
	}
	if !page.HasMore {
		t.Error("expected hasMore=true")
	}
	if page.After == "" {
		t.Error("expected non-empty after cursor")
	}

	// Get second page.
	page2, err := ls.GetMemberships(ctx, list.ListID, page.After, 2)
	if err != nil {
		t.Fatalf("get page 2: %v", err)
	}
	if len(page2.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(page2.Results))
	}
	if !page2.HasMore {
		t.Error("expected hasMore=true for page 2")
	}

	// Get last page.
	page3, err := ls.GetMemberships(ctx, list.ListID, page2.After, 2)
	if err != nil {
		t.Fatalf("get page 3: %v", err)
	}
	if len(page3.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(page3.Results))
	}
	if page3.HasMore {
		t.Error("expected hasMore=false for last page")
	}
}

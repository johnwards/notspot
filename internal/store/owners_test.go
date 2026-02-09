package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

var _ store.OwnerStore = (*store.SQLiteOwnerStore)(nil)

func setupOwnerStore(t *testing.T) *store.SQLiteOwnerStore {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return store.NewSQLiteOwnerStore(db)
}

func TestOwnerCreate(t *testing.T) {
	s := setupOwnerStore(t)
	ctx := context.Background()

	owner, err := s.Create(ctx, "test@example.com", "Test", "User", 1001)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if owner.ID == "" {
		t.Error("expected non-empty ID")
	}
	if owner.Email != "test@example.com" {
		t.Errorf("expected email=test@example.com, got %s", owner.Email)
	}
	if owner.FirstName != "Test" {
		t.Errorf("expected firstName=Test, got %s", owner.FirstName)
	}
	if owner.LastName != "User" {
		t.Errorf("expected lastName=User, got %s", owner.LastName)
	}
	if owner.UserID != 1001 {
		t.Errorf("expected userId=1001, got %d", owner.UserID)
	}
}

func TestOwnerGet(t *testing.T) {
	s := setupOwnerStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, "get@example.com", "Get", "Owner", 2001)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Email != "get@example.com" {
		t.Errorf("expected email=get@example.com, got %s", got.Email)
	}
}

func TestOwnerGetNotFound(t *testing.T) {
	s := setupOwnerStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "999")
	if err == nil {
		t.Fatal("expected error for nonexistent owner")
	}
}

func TestOwnerList(t *testing.T) {
	s := setupOwnerStore(t)
	ctx := context.Background()

	for i := range 3 {
		_, err := s.Create(ctx, "owner"+string(rune('0'+i))+"@example.com", "Owner", "Test", 3000+i)
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	// List all.
	owners, hasMore, _, err := s.List(ctx, 100, "", "", false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(owners) != 3 {
		t.Fatalf("expected 3 owners, got %d", len(owners))
	}
	if hasMore {
		t.Error("expected hasMore=false")
	}

	// Paginated list.
	owners, hasMore, nextAfter, err := s.List(ctx, 2, "", "", false)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(owners) != 2 {
		t.Fatalf("expected 2 owners, got %d", len(owners))
	}
	if !hasMore {
		t.Error("expected hasMore=true")
	}

	owners2, hasMore2, _, err := s.List(ctx, 2, nextAfter, "", false)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(owners2) != 1 {
		t.Fatalf("expected 1 owner on page 2, got %d", len(owners2))
	}
	if hasMore2 {
		t.Error("expected hasMore=false on last page")
	}
}

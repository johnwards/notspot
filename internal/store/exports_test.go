package store_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

var _ store.ExportStore = (*store.SQLiteExportStore)(nil)

func setupExportStore(t *testing.T) *store.SQLiteExportStore {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return store.NewSQLiteExportStore(db)
}

func TestExportCreate(t *testing.T) {
	s := setupExportStore(t)
	ctx := context.Background()

	exp, err := s.Create(ctx, "Test Export", "VIEW", "contacts", []string{"email", "firstname"}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if exp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if exp.State != "ENQUEUED" {
		t.Errorf("expected state=ENQUEUED, got %s", exp.State)
	}
	if exp.ExportType != "VIEW" {
		t.Errorf("expected exportType=VIEW, got %s", exp.ExportType)
	}
}

func TestExportGetAndComplete(t *testing.T) {
	s := setupExportStore(t)
	ctx := context.Background()

	exp, err := s.Create(ctx, "Complete Test", "VIEW", "contacts", []string{"email"}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	csvData := []byte("hs_object_id,email\n1,test@example.com\n")
	if err := s.Complete(ctx, exp.ID, csvData, 1); err != nil {
		t.Fatalf("complete: %v", err)
	}

	got, err := s.Get(ctx, exp.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != "COMPLETE" {
		t.Errorf("expected state=COMPLETE, got %s", got.State)
	}
	if got.RecordCount != 1 {
		t.Errorf("expected recordCount=1, got %d", got.RecordCount)
	}
	if !bytes.Equal(got.ResultData, csvData) {
		t.Errorf("expected result data to match")
	}
}

func TestExportGetNotFound(t *testing.T) {
	s := setupExportStore(t)
	ctx := context.Background()

	_, err := s.Get(ctx, "999")
	if err == nil {
		t.Fatal("expected error for nonexistent export")
	}
}

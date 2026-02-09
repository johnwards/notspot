package database_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func TestOpen(t *testing.T) {
	db := testhelpers.NewTestDB(t)

	if err := db.Ping(); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	// Verify WAL mode is set.
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	// In-memory databases may report "memory" instead of "wal".
	if journalMode != "wal" && journalMode != "memory" {
		t.Errorf("journal_mode = %q, want wal or memory", journalMode)
	}

	// Verify foreign keys are enabled.
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestMigrate(t *testing.T) {
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Verify schema_migrations table exists and is queryable.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	// Run migrations twice — should not error.
	for i := 0; i < 2; i++ {
		if err := database.Migrate(ctx, db); err != nil {
			t.Fatalf("migrate (run %d): %v", i+1, err)
		}
	}
}

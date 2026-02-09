package database_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func TestMigrationsCreateAllTables(t *testing.T) {
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	tables := []string{
		"schema_migrations",
		"object_types",
		"property_definitions",
		"property_groups",
		"objects",
		"property_values",
		"property_value_history",
		"association_types",
		"associations",
		"pipelines",
		"pipeline_stages",
		"lists",
		"list_memberships",
		"imports",
		"import_errors",
		"exports",
		"owners",
		"request_log",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := database.Migrate(ctx, db); err != nil {
			t.Fatalf("migrate (run %d): %v", i+1, err)
		}
	}

	// Verify version was recorded.
	var version int
	err := db.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
	if err != nil {
		t.Fatalf("query version: %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
}

func TestMigrationsIndexes(t *testing.T) {
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	indexes := []string{
		"idx_objects_type",
		"idx_objects_type_created",
		"idx_property_values_value",
		"idx_prop_history",
		"idx_assoc_from",
		"idx_assoc_to",
		"idx_request_log_time",
	}

	for _, idx := range indexes {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name)
		if err != nil {
			t.Errorf("index %q not found: %v", idx, err)
		}
	}
}

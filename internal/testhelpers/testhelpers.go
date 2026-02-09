package testhelpers

import (
	"database/sql"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
)

// NewTestDB returns an in-memory SQLite database configured the same way as
// production. The database is automatically closed when the test completes.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

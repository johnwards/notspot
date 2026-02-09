package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// now returns the current UTC time formatted as a HubSpot-compatible timestamp.
func now() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// ResolveObjectType resolves an object type path parameter (name like "contacts"
// or ID like "0-1") to the internal type ID used in the database.
func ResolveObjectType(ctx context.Context, db *sql.DB, objectType string) (string, error) {
	var typeID string
	err := db.QueryRowContext(ctx,
		`SELECT id FROM object_types WHERE name = ? OR id = ?`,
		objectType, objectType,
	).Scan(&typeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("object type %q not found", objectType)
		}
		return "", fmt.Errorf("resolve object type: %w", err)
	}
	return typeID, nil
}

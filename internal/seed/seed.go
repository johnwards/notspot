package seed

import (
	"context"
	"database/sql"
	"fmt"
)

// Seed inserts all standard seed data into the database. It is idempotent —
// existing rows are left untouched. Call order matters: object types first,
// then properties, pipelines, associations, and owners.
func Seed(ctx context.Context, db *sql.DB) error {
	if err := Properties(ctx, db); err != nil {
		return fmt.Errorf("seed properties: %w", err)
	}
	if err := Pipelines(ctx, db); err != nil {
		return fmt.Errorf("seed pipelines: %w", err)
	}
	if err := AssociationTypes(ctx, db); err != nil {
		return fmt.Errorf("seed association types: %w", err)
	}
	if err := Owners(ctx, db); err != nil {
		return fmt.Errorf("seed owners: %w", err)
	}
	return nil
}

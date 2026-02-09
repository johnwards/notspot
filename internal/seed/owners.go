package seed

import (
	"context"
	"database/sql"
	"fmt"
)

type ownerDef struct {
	email     string
	firstName string
	lastName  string
	userID    int
}

var defaultOwners = []ownerDef{
	{email: "admin@example.com", firstName: "Admin", lastName: "User", userID: 1001},
	{email: "sales@example.com", firstName: "Sales", lastName: "Rep", userID: 1002},
	{email: "support@example.com", firstName: "Support", lastName: "Agent", userID: 1003},
}

// Owners inserts default test owners if none exist yet.
func Owners(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM owners`).Scan(&count); err != nil {
		return fmt.Errorf("count owners: %w", err)
	}
	if count > 0 {
		return nil
	}

	ts := "2024-01-01T00:00:00.000Z"
	for _, od := range defaultOwners {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO owners (email, first_name, last_name, user_id, archived, created_at, updated_at)
			 VALUES (?, ?, ?, ?, FALSE, ?, ?)`,
			od.email, od.firstName, od.lastName, od.userID, ts, ts,
		); err != nil {
			return fmt.Errorf("insert owner %s: %w", od.email, err)
		}
	}

	return nil
}

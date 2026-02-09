package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// Owner represents a HubSpot owner.
type Owner struct {
	ID                      string `json:"id"`
	Email                   string `json:"email"`
	FirstName               string `json:"firstName"`
	LastName                string `json:"lastName"`
	UserID                  int    `json:"userId"`
	UserIDIncludingInactive int    `json:"userIdIncludingInactive"`
	Type                    string `json:"type"`
	Archived                bool   `json:"archived"`
	Teams                   []any  `json:"teams"`
	CreatedAt               string `json:"createdAt"`
	UpdatedAt               string `json:"updatedAt"`
}

// OwnerStore defines the interface for owner persistence.
type OwnerStore interface {
	List(ctx context.Context, limit int, after string, email string, archived bool) ([]*Owner, bool, string, error)
	Get(ctx context.Context, id string) (*Owner, error)
	Create(ctx context.Context, email, firstName, lastName string, userID int) (*Owner, error)
}

// SQLiteOwnerStore implements OwnerStore backed by SQLite.
type SQLiteOwnerStore struct {
	db *sql.DB
}

// NewSQLiteOwnerStore creates a new SQLiteOwnerStore.
func NewSQLiteOwnerStore(db *sql.DB) *SQLiteOwnerStore {
	return &SQLiteOwnerStore{db: db}
}

// Create inserts a new owner.
func (s *SQLiteOwnerStore) Create(ctx context.Context, email, firstName, lastName string, userID int) (*Owner, error) {
	ts := now()

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO owners (email, first_name, last_name, user_id, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, FALSE, ?, ?)`,
		email, firstName, lastName, userID, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("insert owner: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return &Owner{
		ID:                      strconv.FormatInt(id, 10),
		Email:                   email,
		FirstName:               firstName,
		LastName:                lastName,
		UserID:                  userID,
		UserIDIncludingInactive: userID,
		Type:                    "PERSON",
		Teams:                   []any{},
		CreatedAt:               ts,
		UpdatedAt:               ts,
	}, nil
}

// List returns a paginated list of owners.
//
//nolint:gocritic // named results provide clarity for multiple return values
func (s *SQLiteOwnerStore) List(ctx context.Context, limit int, after string, email string, archived bool) ([]*Owner, bool, string, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, email, first_name, last_name, user_id, archived, created_at, updated_at FROM owners WHERE archived = ?`
	args := []any{archived}

	if email != "" {
		query += ` AND email = ?`
		args = append(args, email)
	}

	if after != "" {
		query += ` AND id > ?`
		args = append(args, after)
	}

	query += ` ORDER BY id ASC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, "", fmt.Errorf("list owners: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var owners []*Owner
	for rows.Next() {
		var o Owner
		var id int64
		if err := rows.Scan(&id, &o.Email, &o.FirstName, &o.LastName, &o.UserID, &o.Archived, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, false, "", fmt.Errorf("scan owner: %w", err)
		}
		o.ID = strconv.FormatInt(id, 10)
		o.UserIDIncludingInactive = o.UserID
		o.Type = "PERSON"
		o.Teams = []any{}
		owners = append(owners, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", fmt.Errorf("rows iteration: %w", err)
	}

	hasMore := false
	nextAfter := ""
	if len(owners) > limit {
		hasMore = true
		nextAfter = owners[limit-1].ID
		owners = owners[:limit]
	}

	return owners, hasMore, nextAfter, nil
}

// Get retrieves a single owner by ID.
func (s *SQLiteOwnerStore) Get(ctx context.Context, id string) (*Owner, error) {
	var o Owner
	var dbID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, first_name, last_name, user_id, archived, created_at, updated_at FROM owners WHERE id = ?`,
		id,
	).Scan(&dbID, &o.Email, &o.FirstName, &o.LastName, &o.UserID, &o.Archived, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get owner: %w", err)
	}
	o.ID = strconv.FormatInt(dbID, 10)
	o.UserIDIncludingInactive = o.UserID
	o.Type = "PERSON"
	o.Teams = []any{}
	return &o, nil
}

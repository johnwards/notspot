package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/johnwards/hubspot/internal/domain"
)

// ListStore defines the interface for list persistence.
type ListStore interface {
	Create(ctx context.Context, name, objectTypeID, processingType string, filterBranch json.RawMessage) (*domain.List, error)
	Get(ctx context.Context, listID string) (*domain.List, error)
	GetMultiple(ctx context.Context, listIDs []string) ([]*domain.List, error)
	Delete(ctx context.Context, listID string) error
	Restore(ctx context.Context, listID string) error
	UpdateName(ctx context.Context, listID, name string) (*domain.List, error)
	UpdateFilters(ctx context.Context, listID string, filterBranch json.RawMessage) (*domain.List, error)
	Search(ctx context.Context, opts domain.ListSearchOpts) (*domain.ListSearchPage, error)
	GetMemberships(ctx context.Context, listID, after string, limit int) (*domain.MembershipPage, error)
	AddMembers(ctx context.Context, listID string, recordIDs []string) ([]string, error)
	RemoveMembers(ctx context.Context, listID string, recordIDs []string) ([]string, error)
	RemoveAllMembers(ctx context.Context, listID string) error
}

// SQLiteListStore implements ListStore backed by SQLite.
type SQLiteListStore struct {
	db *sql.DB
}

// NewSQLiteListStore creates a new SQLiteListStore.
func NewSQLiteListStore(db *sql.DB) *SQLiteListStore {
	return &SQLiteListStore{db: db}
}

// Create inserts a new list.
func (s *SQLiteListStore) Create(ctx context.Context, name, objectTypeID, processingType string, filterBranch json.RawMessage) (*domain.List, error) {
	ts := now()

	var fb *string
	if len(filterBranch) > 0 {
		str := string(filterBranch)
		fb = &str
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO lists (name, object_type_id, processing_type, processing_status, filter_branch, list_version, created_at, updated_at)
		 VALUES (?, ?, ?, 'COMPLETE', ?, 1, ?, ?)`,
		name, objectTypeID, processingType, fb, ts, ts,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("list name %q already exists: %w", name, ErrConflict)
		}
		return nil, fmt.Errorf("create list: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return &domain.List{
		ListID:           strconv.FormatInt(id, 10),
		Name:             name,
		ObjectTypeId:     objectTypeID,
		ProcessingType:   processingType,
		ProcessingStatus: "COMPLETE",
		FilterBranch:     filterBranch,
		ListVersion:      1,
		Size:             0,
		CreatedAt:        ts,
		UpdatedAt:        ts,
	}, nil
}

// Get retrieves a single list by ID.
func (s *SQLiteListStore) Get(ctx context.Context, listID string) (*domain.List, error) {
	return s.scanList(s.db.QueryRowContext(ctx,
		`SELECT l.id, l.name, l.object_type_id, l.processing_type, l.processing_status,
		        l.filter_branch, l.list_version, l.created_at, l.updated_at,
		        (SELECT COUNT(*) FROM list_memberships WHERE list_id = l.id)
		 FROM lists l
		 WHERE l.id = ? AND l.archived = FALSE`,
		listID,
	))
}

// GetMultiple retrieves multiple lists by their IDs.
func (s *SQLiteListStore) GetMultiple(ctx context.Context, listIDs []string) ([]*domain.List, error) {
	if len(listIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(listIDs))
	args := make([]any, len(listIDs))
	for i, id := range listIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT l.id, l.name, l.object_type_id, l.processing_type, l.processing_status,
		        l.filter_branch, l.list_version, l.created_at, l.updated_at,
		        (SELECT COUNT(*) FROM list_memberships WHERE list_id = l.id)
		 FROM lists l
		 WHERE l.id IN (`+strings.Join(placeholders, ",")+`) AND l.archived = FALSE`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get multiple lists: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var lists []*domain.List
	for rows.Next() {
		l, err := s.scanListFromRow(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	return lists, rows.Err()
}

// Delete soft-deletes a list by ID.
func (s *SQLiteListStore) Delete(ctx context.Context, listID string) error {
	ts := now()
	result, err := s.db.ExecContext(ctx,
		`UPDATE lists SET archived = TRUE, deleted_at = ?, updated_at = ? WHERE id = ? AND archived = FALSE`,
		ts, ts, listID,
	)
	if err != nil {
		return fmt.Errorf("delete list: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list %s: %w", listID, ErrNotFound)
	}
	return nil
}

// Restore un-deletes a previously deleted list.
func (s *SQLiteListStore) Restore(ctx context.Context, listID string) error {
	ts := now()
	result, err := s.db.ExecContext(ctx,
		`UPDATE lists SET archived = FALSE, deleted_at = NULL, updated_at = ? WHERE id = ? AND archived = TRUE`,
		ts, listID,
	)
	if err != nil {
		return fmt.Errorf("restore list: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("list %s: %w", listID, ErrNotFound)
	}
	return nil
}

// UpdateName renames a list.
func (s *SQLiteListStore) UpdateName(ctx context.Context, listID, name string) (*domain.List, error) {
	ts := now()
	result, err := s.db.ExecContext(ctx,
		`UPDATE lists SET name = ?, list_version = list_version + 1, updated_at = ? WHERE id = ? AND archived = FALSE`,
		name, ts, listID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("list name %q already exists: %w", name, ErrConflict)
		}
		return nil, fmt.Errorf("update list name: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("list %s: %w", listID, ErrNotFound)
	}
	return s.Get(ctx, listID)
}

// UpdateFilters replaces the filter branch JSON on a list.
func (s *SQLiteListStore) UpdateFilters(ctx context.Context, listID string, filterBranch json.RawMessage) (*domain.List, error) {
	ts := now()
	var fb *string
	if len(filterBranch) > 0 {
		str := string(filterBranch)
		fb = &str
	}

	result, err := s.db.ExecContext(ctx,
		`UPDATE lists SET filter_branch = ?, list_version = list_version + 1, updated_at = ? WHERE id = ? AND archived = FALSE`,
		fb, ts, listID,
	)
	if err != nil {
		return nil, fmt.Errorf("update list filters: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("list %s: %w", listID, ErrNotFound)
	}
	return s.Get(ctx, listID)
}

// Search finds lists matching a query with offset-based pagination.
func (s *SQLiteListStore) Search(ctx context.Context, opts domain.ListSearchOpts) (*domain.ListSearchPage, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	var where string
	var args []any
	if opts.Query != "" {
		where = " AND l.name LIKE ?"
		args = append(args, "%"+opts.Query+"%")
	}

	var total int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM lists l WHERE l.archived = FALSE`+where,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count lists: %w", err)
	}

	queryArgs := make([]any, len(args))
	copy(queryArgs, args)
	queryArgs = append(queryArgs, limit+1, opts.Offset)

	rows, err := s.db.QueryContext(ctx,
		`SELECT l.id, l.name, l.object_type_id, l.processing_type, l.processing_status,
		        l.filter_branch, l.list_version, l.created_at, l.updated_at,
		        (SELECT COUNT(*) FROM list_memberships WHERE list_id = l.id)
		 FROM lists l
		 WHERE l.archived = FALSE`+where+`
		 ORDER BY l.id
		 LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("search lists: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var lists []*domain.List
	for rows.Next() {
		l, err := s.scanListFromRow(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(lists) > limit
	if hasMore {
		lists = lists[:limit]
	}

	return &domain.ListSearchPage{
		Results:    lists,
		Offset:     opts.Offset + len(lists),
		HasMore:    hasMore,
		TotalCount: total,
	}, nil
}

// GetMemberships returns a paginated list of members for a given list.
func (s *SQLiteListStore) GetMemberships(ctx context.Context, listID, after string, limit int) (*domain.MembershipPage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 250 {
		limit = 250
	}

	// Verify list exists.
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM lists WHERE id = ? AND archived = FALSE`, listID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", listID, ErrNotFound)
	}

	var args []any
	whereAfter := ""
	if after != "" {
		whereAfter = " AND lm.object_id > ?"
		args = append(args, after)
	}
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx,
		`SELECT lm.object_id, lm.list_id, lm.added_at
		 FROM list_memberships lm
		 WHERE lm.list_id = ?`+whereAfter+`
		 ORDER BY lm.object_id
		 LIMIT ?`,
		append([]any{listID}, args...)...,
	)
	if err != nil {
		return nil, fmt.Errorf("get memberships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var members []*domain.ListMembership
	for rows.Next() {
		m := &domain.ListMembership{}
		var objectID int64
		var lID int64
		if err := rows.Scan(&objectID, &lID, &m.AddedAt); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		m.RecordID = strconv.FormatInt(objectID, 10)
		m.ListID = strconv.FormatInt(lID, 10)
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(members) > limit
	afterCursor := ""
	if hasMore {
		members = members[:limit]
	}
	if len(members) > 0 && hasMore {
		afterCursor = members[len(members)-1].RecordID
	}

	return &domain.MembershipPage{
		Results: members,
		After:   afterCursor,
		HasMore: hasMore,
	}, nil
}

// AddMembers adds record IDs to a list's membership.
func (s *SQLiteListStore) AddMembers(ctx context.Context, listID string, recordIDs []string) ([]string, error) {
	if err := s.checkManualOrSnapshot(ctx, listID); err != nil {
		return nil, err
	}

	ts := now()
	var added []string
	for _, rid := range recordIDs {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO list_memberships (list_id, object_id, added_at) VALUES (?, ?, ?)`,
			listID, rid, ts,
		)
		if err != nil {
			return nil, fmt.Errorf("add member %s: %w", rid, err)
		}
		added = append(added, rid)
	}
	return added, nil
}

// RemoveMembers removes record IDs from a list's membership.
func (s *SQLiteListStore) RemoveMembers(ctx context.Context, listID string, recordIDs []string) ([]string, error) {
	if err := s.checkManualOrSnapshot(ctx, listID); err != nil {
		return nil, err
	}

	var removed []string
	for _, rid := range recordIDs {
		result, err := s.db.ExecContext(ctx,
			`DELETE FROM list_memberships WHERE list_id = ? AND object_id = ?`,
			listID, rid,
		)
		if err != nil {
			return nil, fmt.Errorf("remove member %s: %w", rid, err)
		}
		n, _ := result.RowsAffected()
		if n > 0 {
			removed = append(removed, rid)
		}
	}
	return removed, nil
}

// RemoveAllMembers removes all memberships from a list.
func (s *SQLiteListStore) RemoveAllMembers(ctx context.Context, listID string) error {
	if err := s.checkManualOrSnapshot(ctx, listID); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, `DELETE FROM list_memberships WHERE list_id = ?`, listID)
	if err != nil {
		return fmt.Errorf("remove all members: %w", err)
	}
	return nil
}

// checkManualOrSnapshot verifies the list exists and allows membership mutation.
func (s *SQLiteListStore) checkManualOrSnapshot(ctx context.Context, listID string) error {
	var processingType string
	err := s.db.QueryRowContext(ctx,
		`SELECT processing_type FROM lists WHERE id = ? AND archived = FALSE`,
		listID,
	).Scan(&processingType)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("list %s: %w", listID, ErrNotFound)
		}
		return fmt.Errorf("check list type: %w", err)
	}
	if processingType != "MANUAL" && processingType != "SNAPSHOT" {
		return ErrDynamicListMutation
	}
	return nil
}

func (s *SQLiteListStore) scanList(row *sql.Row) (*domain.List, error) {
	l := &domain.List{}
	var fb sql.NullString
	var id int64
	err := row.Scan(
		&id, &l.Name, &l.ObjectTypeId, &l.ProcessingType, &l.ProcessingStatus,
		&fb, &l.ListVersion, &l.CreatedAt, &l.UpdatedAt, &l.Size,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("list not found: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("scan list: %w", err)
	}
	l.ListID = strconv.FormatInt(id, 10)
	if fb.Valid {
		l.FilterBranch = json.RawMessage(fb.String)
	}
	return l, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *SQLiteListStore) scanListFromRow(row scanner) (*domain.List, error) {
	l := &domain.List{}
	var fb sql.NullString
	var id int64
	err := row.Scan(
		&id, &l.Name, &l.ObjectTypeId, &l.ProcessingType, &l.ProcessingStatus,
		&fb, &l.ListVersion, &l.CreatedAt, &l.UpdatedAt, &l.Size,
	)
	if err != nil {
		return nil, fmt.Errorf("scan list: %w", err)
	}
	l.ListID = strconv.FormatInt(id, 10)
	if fb.Valid {
		l.FilterBranch = json.RawMessage(fb.String)
	}
	return l, nil
}

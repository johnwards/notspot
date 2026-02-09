package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/johnwards/hubspot/internal/domain"
)

// ObjectStore defines the interface for CRM object persistence.
type ObjectStore interface {
	Create(ctx context.Context, objectType string, properties map[string]string) (*domain.Object, error)
	Get(ctx context.Context, objectType, id string, props []string) (*domain.Object, error)
	GetByProperty(ctx context.Context, objectType, propName, propValue string, props []string) (*domain.Object, error)
	List(ctx context.Context, objectType string, opts domain.ListOpts) (*domain.ObjectPage, error)
	Update(ctx context.Context, objectType, id string, properties map[string]string) (*domain.Object, error)
	Archive(ctx context.Context, objectType, id string) error
	BatchCreate(ctx context.Context, objectType string, inputs []domain.CreateInput) (*domain.BatchResult, error)
	BatchRead(ctx context.Context, objectType string, ids, props []string, idProperty string) (*domain.BatchResult, error)
	BatchUpdate(ctx context.Context, objectType string, inputs []domain.UpdateInput) (*domain.BatchResult, error)
	BatchUpsert(ctx context.Context, objectType string, inputs []domain.UpsertInput, idProperty string) (*domain.BatchResult, error)
	BatchArchive(ctx context.Context, objectType string, ids []string) error
	Merge(ctx context.Context, objectType, primaryID, mergeID string) (*domain.Object, error)
}

// ErrNotFound is returned when a requested object does not exist.
var ErrNotFound = fmt.Errorf("object not found")

// SQLiteObjectStore implements ObjectStore backed by SQLite.
type SQLiteObjectStore struct {
	db *sql.DB
}

// NewSQLiteObjectStore creates a new SQLiteObjectStore.
func NewSQLiteObjectStore(db *sql.DB) *SQLiteObjectStore {
	return &SQLiteObjectStore{db: db}
}

// defaultProps are always returned even when no properties are requested.
var defaultProps = []string{"hs_object_id", "createdate", "lastmodifieddate"}

func (s *SQLiteObjectStore) resolveType(ctx context.Context, objectType string) (string, error) {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return "", fmt.Errorf("%s: %w", err.Error(), ErrNotFound)
	}
	return typeID, nil
}

// Create inserts a new CRM object with the given properties.
func (s *SQLiteObjectStore) Create(ctx context.Context, objectType string, properties map[string]string) (*domain.Object, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	ts := now()

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO objects (object_type_id, created_at, updated_at) VALUES (?, ?, ?)`,
		typeID, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("insert object: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	idStr := strconv.FormatInt(id, 10)

	// Auto-set system properties.
	sysProps := map[string]string{
		"hs_object_id":           idStr,
		"hs_createdate":          ts,
		"hs_lastmodifieddate":    ts,
		"createdate":             ts,
		"lastmodifieddate":       ts,
		"hs_object_source":       "API",
		"hs_object_source_id":    "",
		"hs_object_source_label": "",
	}
	for k, v := range properties {
		sysProps[k] = v
	}

	if err := s.setProperties(ctx, id, sysProps, ts); err != nil {
		return nil, err
	}

	return s.getWithAllProps(ctx, objectType, idStr)
}

// Get retrieves a single object by ID, optionally filtering properties.
func (s *SQLiteObjectStore) Get(ctx context.Context, objectType, id string, props []string) (*domain.Object, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	var obj domain.Object
	var archivedAt sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT id, archived, archived_at, created_at, updated_at FROM objects WHERE id = ? AND object_type_id = ?`,
		id, typeID,
	).Scan(&obj.ID, &obj.Archived, &archivedAt, &obj.CreatedAt, &obj.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", id, ErrNotFound)
	}
	if archivedAt.Valid {
		obj.ArchivedAt = archivedAt.String
	}
	obj.ID = id // Ensure string form.

	obj.Properties, err = s.getProperties(ctx, id, props)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

// GetByProperty looks up a single object by a property value.
func (s *SQLiteObjectStore) GetByProperty(ctx context.Context, objectType, propName, propValue string, props []string) (*domain.Object, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	var objID string
	err = s.db.QueryRowContext(ctx,
		`SELECT o.id FROM objects o
		 JOIN property_values pv ON pv.object_id = o.id
		 WHERE o.object_type_id = ? AND pv.property_name = ? AND pv.value = ? AND o.archived = FALSE`,
		typeID, propName, propValue,
	).Scan(&objID)
	if err != nil {
		return nil, fmt.Errorf("get object by %s=%s: %w", propName, propValue, ErrNotFound)
	}

	return s.Get(ctx, objectType, objID, props)
}

// List returns a paginated list of objects.
func (s *SQLiteObjectStore) List(ctx context.Context, objectType string, opts domain.ListOpts) (*domain.ObjectPage, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	query := `SELECT id, archived, archived_at, created_at, updated_at FROM objects WHERE object_type_id = ? AND archived = ?`
	args := []any{typeID, opts.Archived}

	if opts.After != "" {
		query += ` AND id > ?`
		args = append(args, opts.After)
	}

	// Fetch one extra to determine if there is a next page.
	query += ` ORDER BY id ASC LIMIT ?`
	args = append(args, opts.Limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list objects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	page := &domain.ObjectPage{}
	for rows.Next() {
		var obj domain.Object
		var archivedAt sql.NullString
		if err := rows.Scan(&obj.ID, &obj.Archived, &archivedAt, &obj.CreatedAt, &obj.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan object: %w", err)
		}
		if archivedAt.Valid {
			obj.ArchivedAt = archivedAt.String
		}
		page.Results = append(page.Results, &obj)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if len(page.Results) > opts.Limit {
		page.HasMore = true
		page.After = page.Results[opts.Limit-1].ID
		page.Results = page.Results[:opts.Limit]
	}

	// Fetch properties for each object.
	for _, obj := range page.Results {
		obj.Properties, err = s.getProperties(ctx, obj.ID, opts.Properties)
		if err != nil {
			return nil, err
		}
	}

	return page, nil
}

// Update merges the given properties into an existing object.
func (s *SQLiteObjectStore) Update(ctx context.Context, objectType, id string, properties map[string]string) (*domain.Object, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	// Verify exists.
	var exists int
	err = s.db.QueryRowContext(ctx,
		`SELECT 1 FROM objects WHERE id = ? AND object_type_id = ? AND archived = FALSE`, id, typeID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("object %s not found: %w", id, ErrNotFound)
	}

	ts := now()

	// Add system property update.
	properties["hs_lastmodifieddate"] = ts
	properties["lastmodifieddate"] = ts

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid object id: %w", err)
	}

	if err := s.setProperties(ctx, idInt, properties, ts); err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `UPDATE objects SET updated_at = ? WHERE id = ?`, ts, id)
	if err != nil {
		return nil, fmt.Errorf("update object timestamp: %w", err)
	}

	return s.getWithAllProps(ctx, objectType, id)
}

// Archive soft-deletes an object.
func (s *SQLiteObjectStore) Archive(ctx context.Context, objectType, id string) error {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return err
	}

	ts := now()
	res, err := s.db.ExecContext(ctx,
		`UPDATE objects SET archived = TRUE, archived_at = ?, updated_at = ? WHERE id = ? AND object_type_id = ? AND archived = FALSE`,
		ts, ts, id, typeID,
	)
	if err != nil {
		return fmt.Errorf("archive object: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("object %s: %w", id, ErrNotFound)
	}

	// Clean up associations for the archived object.
	_, _ = s.db.ExecContext(ctx,
		`DELETE FROM associations WHERE from_object_id = ? OR to_object_id = ?`,
		id, id,
	)

	return nil
}

// BatchCreate creates multiple objects in a single operation.
func (s *SQLiteObjectStore) BatchCreate(ctx context.Context, objectType string, inputs []domain.CreateInput) (*domain.BatchResult, error) {
	startedAt := now()
	result := &domain.BatchResult{Status: "COMPLETE", StartedAt: startedAt, Results: []*domain.Object{}}
	for _, input := range inputs {
		obj, err := s.Create(ctx, objectType, input.Properties)
		if err != nil {
			return nil, err
		}
		result.Results = append(result.Results, obj)
	}
	result.CompletedAt = now()
	return result, nil
}

// BatchRead reads multiple objects by ID.
func (s *SQLiteObjectStore) BatchRead(ctx context.Context, objectType string, ids, props []string, idProperty string) (*domain.BatchResult, error) {
	startedAt := now()
	result := &domain.BatchResult{Status: "COMPLETE", StartedAt: startedAt}
	for _, id := range ids {
		var obj *domain.Object
		var err error
		if idProperty != "" && idProperty != "hs_object_id" {
			obj, err = s.GetByProperty(ctx, objectType, idProperty, id, props)
		} else {
			obj, err = s.Get(ctx, objectType, id, props)
		}
		if err != nil {
			result.NumErrors++
			continue
		}
		result.Results = append(result.Results, obj)
	}
	result.CompletedAt = now()
	return result, nil
}

// BatchUpdate updates multiple objects.
func (s *SQLiteObjectStore) BatchUpdate(ctx context.Context, objectType string, inputs []domain.UpdateInput) (*domain.BatchResult, error) {
	startedAt := now()
	result := &domain.BatchResult{Status: "COMPLETE", StartedAt: startedAt}
	for _, input := range inputs {
		obj, err := s.Update(ctx, objectType, input.ID, input.Properties)
		if err != nil {
			return nil, err
		}
		result.Results = append(result.Results, obj)
	}
	result.CompletedAt = now()
	return result, nil
}

// BatchUpsert creates or updates objects based on a matching property.
func (s *SQLiteObjectStore) BatchUpsert(ctx context.Context, objectType string, inputs []domain.UpsertInput, idProperty string) (*domain.BatchResult, error) {
	if idProperty == "" {
		idProperty = "hs_object_id"
	}

	startedAt := now()
	result := &domain.BatchResult{Status: "COMPLETE", StartedAt: startedAt}
	for _, input := range inputs {
		lookupValue := input.ID
		if lookupValue == "" {
			lookupValue = input.Properties[idProperty]
		}

		existing, err := s.GetByProperty(ctx, objectType, idProperty, lookupValue, nil)
		if err != nil {
			// Not found — create.
			obj, createErr := s.Create(ctx, objectType, input.Properties)
			if createErr != nil {
				return nil, createErr
			}
			result.Results = append(result.Results, obj)
		} else {
			// Found — update.
			obj, updateErr := s.Update(ctx, objectType, existing.ID, input.Properties)
			if updateErr != nil {
				return nil, updateErr
			}
			result.Results = append(result.Results, obj)
		}
	}
	result.CompletedAt = now()
	return result, nil
}

// BatchArchive archives multiple objects.
func (s *SQLiteObjectStore) BatchArchive(ctx context.Context, objectType string, ids []string) error {
	for _, id := range ids {
		if err := s.Archive(ctx, objectType, id); err != nil {
			return err
		}
	}
	return nil
}

// Merge merges one object into another. The primary survives; the merged
// object is archived.
func (s *SQLiteObjectStore) Merge(ctx context.Context, objectType, primaryID, mergeID string) (*domain.Object, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	// Verify both exist.
	for _, id := range []string{primaryID, mergeID} {
		var exists int
		err := s.db.QueryRowContext(ctx,
			`SELECT 1 FROM objects WHERE id = ? AND object_type_id = ? AND archived = FALSE`, id, typeID,
		).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("object %s: %w", id, ErrNotFound)
		}
	}

	// Get ALL properties from the merged object.
	mergedProps, err := s.getAllProperties(ctx, mergeID)
	if err != nil {
		return nil, err
	}

	// Get ALL properties from the primary object.
	primaryProps, err := s.getAllProperties(ctx, primaryID)
	if err != nil {
		return nil, err
	}

	// Copy unique properties from merged to primary (don't overwrite existing).
	ts := now()
	propsToSet := map[string]string{}
	for k, v := range mergedProps {
		if _, has := primaryProps[k]; !has {
			propsToSet[k] = v
		}
	}

	// Record merged IDs.
	existing := primaryProps["hs_merged_object_ids"]
	if existing != "" {
		propsToSet["hs_merged_object_ids"] = existing + ";" + mergeID
	} else {
		propsToSet["hs_merged_object_ids"] = mergeID
	}
	propsToSet["hs_lastmodifieddate"] = ts
	propsToSet["lastmodifieddate"] = ts

	primaryIDInt, err := strconv.ParseInt(primaryID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid primary id: %w", err)
	}

	if err := s.setProperties(ctx, primaryIDInt, propsToSet, ts); err != nil {
		return nil, err
	}

	// Update primary timestamp.
	if _, err := s.db.ExecContext(ctx, `UPDATE objects SET updated_at = ? WHERE id = ?`, ts, primaryID); err != nil {
		return nil, fmt.Errorf("update primary: %w", err)
	}

	// Archive the merged object and set merged_into_id.
	if _, err := s.db.ExecContext(ctx,
		`UPDATE objects SET archived = TRUE, archived_at = ?, updated_at = ?, merged_into_id = ? WHERE id = ?`,
		ts, ts, primaryID, mergeID,
	); err != nil {
		return nil, fmt.Errorf("archive merged: %w", err)
	}

	return s.getWithAllProps(ctx, objectType, primaryID)
}

// getWithAllProps retrieves an object with ALL its properties (used by
// Create, Update, Merge where the response includes everything).
func (s *SQLiteObjectStore) getWithAllProps(ctx context.Context, objectType, id string) (*domain.Object, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	var obj domain.Object
	var archivedAt sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT id, archived, archived_at, created_at, updated_at FROM objects WHERE id = ? AND object_type_id = ?`,
		id, typeID,
	).Scan(&obj.ID, &obj.Archived, &archivedAt, &obj.CreatedAt, &obj.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", id, ErrNotFound)
	}
	if archivedAt.Valid {
		obj.ArchivedAt = archivedAt.String
	}

	// Fetch ALL properties.
	rows, err := s.db.QueryContext(ctx,
		`SELECT property_name, value FROM property_values WHERE object_id = ?`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("get all properties: %w", err)
	}
	defer func() { _ = rows.Close() }()

	obj.Properties = make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		obj.Properties[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return &obj, nil
}

// getAllProperties fetches every property value for an object.
func (s *SQLiteObjectStore) getAllProperties(ctx context.Context, objectID string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT property_name, value FROM property_values WHERE object_id = ?`, objectID,
	)
	if err != nil {
		return nil, fmt.Errorf("get all properties: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		result[name] = value
	}
	return result, rows.Err()
}

// setProperties upserts property values and records history.
func (s *SQLiteObjectStore) setProperties(ctx context.Context, objectID int64, props map[string]string, ts string) error {
	for name, value := range props {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO property_values (object_id, property_name, value, updated_at) VALUES (?, ?, ?, ?)
			 ON CONFLICT(object_id, property_name) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
			objectID, name, value, ts,
		)
		if err != nil {
			return fmt.Errorf("set property %s: %w", name, err)
		}

		_, err = s.db.ExecContext(ctx,
			`INSERT INTO property_value_history (object_id, property_name, value, timestamp) VALUES (?, ?, ?, ?)`,
			objectID, name, value, ts,
		)
		if err != nil {
			return fmt.Errorf("record property history %s: %w", name, err)
		}
	}
	return nil
}

// getProperties fetches property values for an object. If props is nil or
// empty, only default properties are returned. If props contains specific
// names, those plus defaults are returned.
func (s *SQLiteObjectStore) getProperties(ctx context.Context, objectID string, props []string) (map[string]string, error) {
	var rows *sql.Rows
	var err error

	if len(props) == 0 {
		// Return only default properties.
		placeholders := make([]string, len(defaultProps))
		args := make([]any, 0, len(defaultProps)+1)
		args = append(args, objectID)
		for i, p := range defaultProps {
			placeholders[i] = "?"
			args = append(args, p)
		}
		rows, err = s.db.QueryContext(ctx,
			`SELECT property_name, value FROM property_values WHERE object_id = ? AND property_name IN (`+strings.Join(placeholders, ",")+`)`,
			args...,
		)
	} else {
		// Return requested properties plus defaults.
		allProps := make(map[string]bool)
		for _, p := range defaultProps {
			allProps[p] = true
		}
		for _, p := range props {
			allProps[p] = true
		}

		placeholders := make([]string, 0, len(allProps))
		args := make([]any, 0, len(allProps)+1)
		args = append(args, objectID)
		for p := range allProps {
			placeholders = append(placeholders, "?")
			args = append(args, p)
		}
		rows, err = s.db.QueryContext(ctx,
			`SELECT property_name, value FROM property_values WHERE object_id = ? AND property_name IN (`+strings.Join(placeholders, ",")+`)`,
			args...,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("get properties: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		result[name] = value
	}
	return result, rows.Err()
}

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/johnwards/hubspot/internal/domain"
)

// PropertyStore defines operations for property definitions and groups.
type PropertyStore interface {
	List(ctx context.Context, objectType string) ([]domain.Property, error)
	Create(ctx context.Context, objectType string, p *domain.Property) (*domain.Property, error)
	Get(ctx context.Context, objectType string, name string) (*domain.Property, error)
	Update(ctx context.Context, objectType string, name string, p *domain.Property) (*domain.Property, error)
	Archive(ctx context.Context, objectType string, name string) error
	BatchCreate(ctx context.Context, objectType string, props []domain.Property) ([]domain.Property, error)
	BatchRead(ctx context.Context, objectType string, names []string) ([]domain.Property, error)
	BatchArchive(ctx context.Context, objectType string, names []string) error

	ListGroups(ctx context.Context, objectType string) ([]domain.PropertyGroup, error)
	CreateGroup(ctx context.Context, objectType string, g *domain.PropertyGroup) (*domain.PropertyGroup, error)
	GetGroup(ctx context.Context, objectType string, name string) (*domain.PropertyGroup, error)
	UpdateGroup(ctx context.Context, objectType string, name string, g *domain.PropertyGroup) (*domain.PropertyGroup, error)
	ArchiveGroup(ctx context.Context, objectType string, name string) error
}

// SQLitePropertyStore implements PropertyStore using SQLite.
type SQLitePropertyStore struct {
	db *sql.DB
}

// NewSQLitePropertyStore creates a new SQLitePropertyStore.
func NewSQLitePropertyStore(db *sql.DB) *SQLitePropertyStore {
	return &SQLitePropertyStore{db: db}
}

func (s *SQLitePropertyStore) resolveType(ctx context.Context, objectType string) (string, error) {
	return ResolveObjectType(ctx, s.db, objectType)
}

func encodeOptions(opts []domain.Option) (string, error) {
	if opts == nil {
		opts = []domain.Option{}
	}
	b, err := json.Marshal(opts)
	if err != nil {
		return "", fmt.Errorf("encode options: %w", err)
	}
	return string(b), nil
}

func decodeOptions(raw sql.NullString) ([]domain.Option, error) {
	if !raw.Valid || raw.String == "" {
		return []domain.Option{}, nil
	}
	var opts []domain.Option
	if err := json.Unmarshal([]byte(raw.String), &opts); err != nil {
		return nil, fmt.Errorf("decode options: %w", err)
	}
	return opts, nil
}

func scanProperty(row interface{ Scan(dest ...any) error }) (*domain.Property, error) {
	var p domain.Property
	var optionsRaw sql.NullString
	err := row.Scan(
		&p.Name, &p.Label, &p.Type, &p.FieldType,
		&p.GroupName, &p.Description, &p.DisplayOrder,
		&p.HasUniqueValue, &p.Hidden, &p.FormField,
		&p.Calculated, &p.ExternalOptions, &p.HubspotDefined,
		&optionsRaw, &p.Archived, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	opts, err := decodeOptions(optionsRaw)
	if err != nil {
		return nil, err
	}
	p.Options = opts
	if p.HubspotDefined {
		p.ModificationMetadata = &domain.ModificationMetadata{
			ReadOnlyDefinition: true,
			ReadOnlyValue:      false,
			ReadOnlyOptions:    false,
			Archivable:         false,
		}
	}
	return &p, nil
}

const propertyCols = `name, label, type, field_type, group_name, description,
	display_order, has_unique_value, hidden, form_field, calculated,
	external_options, hubspot_defined, options, archived, created_at, updated_at`

// List returns all non-archived properties for the given object type.
func (s *SQLitePropertyStore) List(ctx context.Context, objectType string) ([]domain.Property, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT `+propertyCols+` FROM property_definitions
		 WHERE object_type_id = ? AND archived = FALSE
		 ORDER BY display_order, name`, typeID)
	if err != nil {
		return nil, fmt.Errorf("list properties: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var props []domain.Property
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, fmt.Errorf("scan property: %w", err)
		}
		props = append(props, *p)
	}
	return props, rows.Err()
}

// Create inserts a new property definition.
func (s *SQLitePropertyStore) Create(ctx context.Context, objectType string, p *domain.Property) (*domain.Property, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	ts := now()
	p.CreatedAt = ts
	p.UpdatedAt = ts

	optStr, err := encodeOptions(p.Options)
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO property_definitions (
			object_type_id, name, label, type, field_type, group_name, description,
			display_order, has_unique_value, hidden, form_field, calculated,
			external_options, hubspot_defined, options, archived, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, FALSE, ?, ?)`,
		typeID, p.Name, p.Label, p.Type, p.FieldType, p.GroupName, p.Description,
		p.DisplayOrder, p.HasUniqueValue, p.Hidden, p.FormField, p.Calculated,
		p.ExternalOptions, p.HubspotDefined, optStr, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("create property: %w", err)
	}

	return p, nil
}

// Get retrieves a single property definition by name.
func (s *SQLitePropertyStore) Get(ctx context.Context, objectType, name string) (*domain.Property, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT `+propertyCols+` FROM property_definitions
		 WHERE object_type_id = ? AND name = ?`, typeID, name)

	p, err := scanProperty(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("property %q not found", name)
		}
		return nil, fmt.Errorf("get property: %w", err)
	}
	return p, nil
}

// Update modifies an existing property definition.
func (s *SQLitePropertyStore) Update(ctx context.Context, objectType, name string, p *domain.Property) (*domain.Property, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	ts := now()
	optStr, err := encodeOptions(p.Options)
	if err != nil {
		return nil, err
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE property_definitions SET
			label = ?, description = ?, group_name = ?, field_type = ?,
			display_order = ?, options = ?, hidden = ?, form_field = ?,
			updated_at = ?
		 WHERE object_type_id = ? AND name = ? AND archived = FALSE`,
		p.Label, p.Description, p.GroupName, p.FieldType,
		p.DisplayOrder, optStr, p.Hidden, p.FormField,
		ts, typeID, name,
	)
	if err != nil {
		return nil, fmt.Errorf("update property: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("property %q not found", name)
	}

	return s.Get(ctx, objectType, name)
}

// Archive soft-deletes a property definition.
func (s *SQLitePropertyStore) Archive(ctx context.Context, objectType, name string) error {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE property_definitions SET archived = TRUE, updated_at = ?
		 WHERE object_type_id = ? AND name = ? AND archived = FALSE`,
		now(), typeID, name,
	)
	if err != nil {
		return fmt.Errorf("archive property: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("property %q not found", name)
	}
	return nil
}

// BatchCreate inserts multiple property definitions.
func (s *SQLitePropertyStore) BatchCreate(ctx context.Context, objectType string, props []domain.Property) ([]domain.Property, error) {
	var results []domain.Property
	for i := range props {
		created, err := s.Create(ctx, objectType, &props[i])
		if err != nil {
			return nil, err
		}
		results = append(results, *created)
	}
	return results, nil
}

// BatchRead retrieves multiple property definitions by name.
func (s *SQLitePropertyStore) BatchRead(ctx context.Context, objectType string, names []string) ([]domain.Property, error) {
	var results []domain.Property
	for _, name := range names {
		p, err := s.Get(ctx, objectType, name)
		if err != nil {
			return nil, err
		}
		results = append(results, *p)
	}
	return results, nil
}

// BatchArchive soft-deletes multiple property definitions.
func (s *SQLitePropertyStore) BatchArchive(ctx context.Context, objectType string, names []string) error {
	for _, name := range names {
		if err := s.Archive(ctx, objectType, name); err != nil {
			return err
		}
	}
	return nil
}

// ListGroups returns all non-archived property groups for the given object type.
func (s *SQLitePropertyStore) ListGroups(ctx context.Context, objectType string) ([]domain.PropertyGroup, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT name, label, display_order, archived FROM property_groups
		 WHERE object_type_id = ? AND archived = FALSE
		 ORDER BY display_order, name`, typeID)
	if err != nil {
		return nil, fmt.Errorf("list property groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var groups []domain.PropertyGroup
	for rows.Next() {
		var g domain.PropertyGroup
		if err := rows.Scan(&g.Name, &g.Label, &g.DisplayOrder, &g.Archived); err != nil {
			return nil, fmt.Errorf("scan property group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// CreateGroup inserts a new property group.
func (s *SQLitePropertyStore) CreateGroup(ctx context.Context, objectType string, g *domain.PropertyGroup) (*domain.PropertyGroup, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO property_groups (object_type_id, name, label, display_order, archived)
		 VALUES (?, ?, ?, ?, FALSE)`,
		typeID, g.Name, g.Label, g.DisplayOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("create property group: %w", err)
	}
	return g, nil
}

// GetGroup retrieves a single property group by name.
func (s *SQLitePropertyStore) GetGroup(ctx context.Context, objectType, name string) (*domain.PropertyGroup, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	var g domain.PropertyGroup
	err = s.db.QueryRowContext(ctx,
		`SELECT name, label, display_order, archived FROM property_groups
		 WHERE object_type_id = ? AND name = ?`, typeID, name,
	).Scan(&g.Name, &g.Label, &g.DisplayOrder, &g.Archived)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("property group %q not found", name)
		}
		return nil, fmt.Errorf("get property group: %w", err)
	}
	return &g, nil
}

// UpdateGroup modifies an existing property group.
func (s *SQLitePropertyStore) UpdateGroup(ctx context.Context, objectType, name string, g *domain.PropertyGroup) (*domain.PropertyGroup, error) {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE property_groups SET label = ?, display_order = ?
		 WHERE object_type_id = ? AND name = ? AND archived = FALSE`,
		g.Label, g.DisplayOrder, typeID, name,
	)
	if err != nil {
		return nil, fmt.Errorf("update property group: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("property group %q not found", name)
	}

	return s.GetGroup(ctx, objectType, name)
}

// ArchiveGroup soft-deletes a property group.
func (s *SQLitePropertyStore) ArchiveGroup(ctx context.Context, objectType, name string) error {
	typeID, err := s.resolveType(ctx, objectType)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE property_groups SET archived = TRUE
		 WHERE object_type_id = ? AND name = ? AND archived = FALSE`,
		typeID, name,
	)
	if err != nil {
		return fmt.Errorf("archive property group: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("property group %q not found", name)
	}
	return nil
}

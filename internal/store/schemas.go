package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/johnwards/hubspot/internal/domain"
)

// SchemaStore defines operations for custom object schema definitions.
type SchemaStore interface {
	List(ctx context.Context) ([]domain.ObjectSchema, error)
	Create(ctx context.Context, s *domain.ObjectSchema) (*domain.ObjectSchema, error)
	Get(ctx context.Context, objectType string) (*domain.ObjectSchema, error)
	Update(ctx context.Context, objectType string, s *domain.ObjectSchema) (*domain.ObjectSchema, error)
	Archive(ctx context.Context, objectType string) error
	CreateAssociation(ctx context.Context, objectType string, a *domain.SchemaAssociation) (*domain.SchemaAssociation, error)
	DeleteAssociation(ctx context.Context, objectType string, associationID string) error
}

// SQLiteSchemaStore implements SchemaStore using SQLite.
type SQLiteSchemaStore struct {
	db *sql.DB
}

// NewSQLiteSchemaStore creates a new SQLiteSchemaStore.
func NewSQLiteSchemaStore(db *sql.DB) *SQLiteSchemaStore {
	return &SQLiteSchemaStore{db: db}
}

// resolveSchemaType resolves an object type (name or ID) to the type row, but
// only for custom types (is_custom = TRUE).
func (s *SQLiteSchemaStore) resolveSchemaType(ctx context.Context, objectType string) (typeID, name string, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT id, name FROM object_types WHERE (name = ? OR id = ?) AND is_custom = TRUE`,
		objectType, objectType,
	).Scan(&typeID, &name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("schema %q not found", objectType)
		}
		return "", "", fmt.Errorf("resolve schema type: %w", err)
	}
	return typeID, name, nil
}

// nextCustomTypeID generates the next 2-{n} ID for a custom object type.
func (s *SQLiteSchemaStore) nextCustomTypeID(ctx context.Context) (string, error) {
	var maxNum int
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(CAST(SUBSTR(id, 3) AS INTEGER)), 0)
		 FROM object_types WHERE id LIKE '2-%'`,
	).Scan(&maxNum)
	if err != nil {
		return "", fmt.Errorf("next custom type id: %w", err)
	}
	return fmt.Sprintf("2-%d", maxNum+1), nil
}

// loadSchema builds a full ObjectSchema from the object_types row, including
// properties and associations.
func (s *SQLiteSchemaStore) loadSchema(ctx context.Context, typeID string) (*domain.ObjectSchema, error) {
	var schema domain.ObjectSchema
	var archived bool
	var fqn, pdp sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, label_singular, label_plural, primary_display_property,
		        fully_qualified_name, archived, created_at, updated_at
		 FROM object_types WHERE id = ? AND is_custom = TRUE`, typeID,
	).Scan(&schema.ID, &schema.Name, &schema.Labels.Singular, &schema.Labels.Plural,
		&pdp, &fqn, &archived, &schema.CreatedAt, &schema.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("schema %q not found", typeID)
		}
		return nil, fmt.Errorf("load schema: %w", err)
	}
	schema.Archived = archived
	schema.FullyQualifiedName = fqn.String
	schema.PrimaryDisplayProperty = pdp.String

	props, err := s.loadProperties(ctx, typeID)
	if err != nil {
		return nil, err
	}
	schema.Properties = props

	assocs, err := s.loadAssociations(ctx, typeID)
	if err != nil {
		return nil, err
	}
	schema.Associations = assocs

	return &schema, nil
}

func (s *SQLiteSchemaStore) loadProperties(ctx context.Context, typeID string) ([]domain.Property, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+propertyCols+` FROM property_definitions
		 WHERE object_type_id = ? AND archived = FALSE
		 ORDER BY display_order, name`, typeID)
	if err != nil {
		return nil, fmt.Errorf("load schema properties: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var props []domain.Property
	for rows.Next() {
		p, err := scanProperty(rows)
		if err != nil {
			return nil, fmt.Errorf("scan schema property: %w", err)
		}
		props = append(props, *p)
	}
	if props == nil {
		props = []domain.Property{}
	}
	return props, rows.Err()
}

func (s *SQLiteSchemaStore) loadAssociations(ctx context.Context, typeID string) ([]domain.SchemaAssociation, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, from_object_type, to_object_type, label, inverse_label
		 FROM association_types
		 WHERE from_object_type = ? OR to_object_type = ?
		 ORDER BY id`, typeID, typeID)
	if err != nil {
		return nil, fmt.Errorf("load schema associations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var assocs []domain.SchemaAssociation
	for rows.Next() {
		var a domain.SchemaAssociation
		var idInt int
		var label, inverseLabel sql.NullString
		if err := rows.Scan(&idInt, &a.FromObjectTypeID, &a.ToObjectTypeID, &label, &inverseLabel); err != nil {
			return nil, fmt.Errorf("scan schema association: %w", err)
		}
		a.ID = strconv.Itoa(idInt)
		if label.Valid {
			a.Name = label.String
		}
		assocs = append(assocs, a)
	}
	if assocs == nil {
		assocs = []domain.SchemaAssociation{}
	}
	return assocs, rows.Err()
}

// List returns all custom object schemas.
func (s *SQLiteSchemaStore) List(ctx context.Context) ([]domain.ObjectSchema, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM object_types WHERE is_custom = TRUE AND archived = FALSE ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var schemas []domain.ObjectSchema
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan schema id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		schema, err := s.loadSchema(ctx, id)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, *schema)
	}
	if schemas == nil {
		schemas = []domain.ObjectSchema{}
	}
	return schemas, nil
}

// Create inserts a new custom object schema and registers the type.
func (s *SQLiteSchemaStore) Create(ctx context.Context, schema *domain.ObjectSchema) (*domain.ObjectSchema, error) {
	if schema.Name == "" {
		return nil, fmt.Errorf("schema name is required")
	}

	typeID, err := s.nextCustomTypeID(ctx)
	if err != nil {
		return nil, err
	}

	ts := now()
	fqn := "p0_" + schema.Name

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO object_types (id, name, label_singular, label_plural, primary_display_property,
		 is_custom, fully_qualified_name, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, TRUE, ?, FALSE, ?, ?)`,
		typeID, schema.Name, schema.Labels.Singular, schema.Labels.Plural,
		schema.PrimaryDisplayProperty, fqn, ts, ts,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("schema %q already exists", schema.Name)
		}
		return nil, fmt.Errorf("create schema: %w", err)
	}

	if err := s.createDefaultProperties(ctx, typeID, ts); err != nil {
		return nil, err
	}

	// Auto-register default association types for declared associated objects.
	for _, assocObj := range schema.AssociatedObjects {
		assocTypeID, err := ResolveObjectType(ctx, s.db, assocObj)
		if err != nil {
			continue
		}
		// Create forward association: custom → associated
		_, _ = s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO association_types (from_object_type, to_object_type, category, label)
			 VALUES (?, ?, 'HUBSPOT_DEFINED', NULL)`,
			typeID, assocTypeID,
		)
		// Create reverse association: associated → custom
		_, _ = s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO association_types (from_object_type, to_object_type, category, label)
			 VALUES (?, ?, 'HUBSPOT_DEFINED', NULL)`,
			assocTypeID, typeID,
		)
	}

	return s.loadSchema(ctx, typeID)
}

// createDefaultProperties inserts the standard HubSpot default properties for a
// newly created custom object type.
func (s *SQLiteSchemaStore) createDefaultProperties(ctx context.Context, typeID, ts string) error {
	defaults := []struct {
		name, label, typ, fieldType string
		hubspotDefined              bool
	}{
		{"hs_object_id", "Object ID", "number", "number", true},
		{"hs_createdate", "Create date", "datetime", "date", true},
		{"hs_lastmodifieddate", "Last modified date", "datetime", "date", true},
	}

	for _, d := range defaults {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO property_definitions (
				object_type_id, name, label, type, field_type, group_name,
				description, display_order, has_unique_value, hidden, form_field,
				calculated, external_options, hubspot_defined, options,
				archived, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, 'schemainfo', '', 0, FALSE, FALSE, FALSE, FALSE, FALSE, ?, '[]', FALSE, ?, ?)`,
			typeID, d.name, d.label, d.typ, d.fieldType, d.hubspotDefined, ts, ts,
		)
		if err != nil {
			return fmt.Errorf("create default property %q: %w", d.name, err)
		}
	}
	return nil
}

// Get retrieves a single custom object schema by name or ID.
func (s *SQLiteSchemaStore) Get(ctx context.Context, objectType string) (*domain.ObjectSchema, error) {
	typeID, _, err := s.resolveSchemaType(ctx, objectType)
	if err != nil {
		return nil, err
	}
	return s.loadSchema(ctx, typeID)
}

// Update modifies an existing custom object schema.
func (s *SQLiteSchemaStore) Update(ctx context.Context, objectType string, patch *domain.ObjectSchema) (*domain.ObjectSchema, error) {
	typeID, _, err := s.resolveSchemaType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	ts := now()
	res, err := s.db.ExecContext(ctx,
		`UPDATE object_types SET
			label_singular = COALESCE(NULLIF(?, ''), label_singular),
			label_plural = COALESCE(NULLIF(?, ''), label_plural),
			primary_display_property = COALESCE(NULLIF(?, ''), primary_display_property),
			updated_at = ?
		 WHERE id = ? AND is_custom = TRUE AND archived = FALSE`,
		patch.Labels.Singular, patch.Labels.Plural,
		patch.PrimaryDisplayProperty, ts, typeID,
	)
	if err != nil {
		return nil, fmt.Errorf("update schema: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("schema %q not found", objectType)
	}

	return s.loadSchema(ctx, typeID)
}

// Archive soft-deletes a custom object schema.
func (s *SQLiteSchemaStore) Archive(ctx context.Context, objectType string) error {
	typeID, _, err := s.resolveSchemaType(ctx, objectType)
	if err != nil {
		return err
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE object_types SET archived = TRUE, updated_at = ?
		 WHERE id = ? AND is_custom = TRUE AND archived = FALSE`,
		now(), typeID,
	)
	if err != nil {
		return fmt.Errorf("archive schema: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("schema %q not found", objectType)
	}
	return nil
}

// CreateAssociation creates a new association type linked to the schema.
func (s *SQLiteSchemaStore) CreateAssociation(ctx context.Context, objectType string, a *domain.SchemaAssociation) (*domain.SchemaAssociation, error) {
	typeID, _, err := s.resolveSchemaType(ctx, objectType)
	if err != nil {
		return nil, err
	}

	// Default fromObjectTypeID to the schema's type ID if not specified.
	if a.FromObjectTypeID == "" {
		a.FromObjectTypeID = typeID
	}

	// Validate that the target type exists.
	_, err = ResolveObjectType(ctx, s.db, a.ToObjectTypeID)
	if err != nil {
		return nil, fmt.Errorf("target object type %q not found", a.ToObjectTypeID)
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO association_types (from_object_type, to_object_type, category, label)
		 VALUES (?, ?, 'USER_DEFINED', ?)`,
		a.FromObjectTypeID, a.ToObjectTypeID, a.Name,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("association already exists")
		}
		return nil, fmt.Errorf("create schema association: %w", err)
	}

	id, _ := res.LastInsertId()
	a.ID = strconv.FormatInt(id, 10)
	return a, nil
}

// DeleteAssociation removes an association type by ID.
func (s *SQLiteSchemaStore) DeleteAssociation(ctx context.Context, objectType, associationID string) error {
	typeID, _, err := s.resolveSchemaType(ctx, objectType)
	if err != nil {
		return err
	}

	assocID, err := strconv.Atoi(associationID)
	if err != nil {
		return fmt.Errorf("invalid association ID %q", associationID)
	}

	res, err := s.db.ExecContext(ctx,
		`DELETE FROM association_types WHERE id = ? AND (from_object_type = ? OR to_object_type = ?)`,
		assocID, typeID, typeID,
	)
	if err != nil {
		return fmt.Errorf("delete schema association: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("association %q not found", associationID)
	}
	return nil
}

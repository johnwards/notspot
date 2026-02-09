package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/johnwards/hubspot/internal/domain"
)

// AssociationStore defines the interface for association persistence.
type AssociationStore interface {
	AssociateDefault(ctx context.Context, fromType, fromID, toType, toID string) (*DefaultAssocResult, error)
	AssociateWithLabels(ctx context.Context, fromType, fromID, toType, toID string, types []AssociationInput) (*DefaultAssocResult, error)
	GetAssociations(ctx context.Context, fromType, fromID, toType string) ([]domain.AssociationResult, error)
	RemoveAssociations(ctx context.Context, fromType, fromID, toType, toID string) error
	ListLabels(ctx context.Context, fromType, toType string) ([]domain.AssociationLabel, error)
	CreateLabel(ctx context.Context, fromType, toType, label, category string) (*domain.AssociationLabel, error)
	UpdateLabel(ctx context.Context, fromType, toType string, typeID int, label string) (*domain.AssociationLabel, error)
	DeleteLabel(ctx context.Context, fromType, toType string, typeID int) error
	BatchAssociateDefault(ctx context.Context, fromType, toType string, inputs []BatchAssocInput) ([]BatchDefaultAssocResult, error)
	BatchCreate(ctx context.Context, fromType, toType string, inputs []BatchAssocCreateInput) ([]BatchCreateResult, error)
	BatchRead(ctx context.Context, fromType, toType string, inputs []BatchAssocReadInput) ([]BatchAssocResult, error)
	BatchArchive(ctx context.Context, fromType, toType string, inputs []BatchArchiveInput) error
	BatchArchiveLabels(ctx context.Context, fromType, toType string, inputs []BatchArchiveLabelInput) error
}

// AssociationInput represents a single association type in a create request.
type AssociationInput struct {
	AssociationCategory string `json:"associationCategory"`
	AssociationTypeID   int    `json:"associationTypeId"`
}

// BatchAssocInput is a from/to pair for batch default associate.
type BatchAssocInput struct {
	From ObjectID `json:"from"`
	To   ObjectID `json:"to"`
}

// ObjectID identifies an object in batch operations.
type ObjectID struct {
	ID string `json:"id"`
}

// BatchAssocCreateInput is a from/to pair with association types.
type BatchAssocCreateInput struct {
	From  ObjectID           `json:"from"`
	To    ObjectID           `json:"to"`
	Types []AssociationInput `json:"types"`
}

// BatchAssocReadInput is a single ID for batch read.
type BatchAssocReadInput struct {
	ID string `json:"id"`
}

// DefaultAssocResult is the result of creating a default association.
type DefaultAssocResult struct {
	Category string
	TypeID   int
}

// BatchDefaultAssocResult is the result for a single pair in batch default associate.
type BatchDefaultAssocResult struct {
	FromID   string
	ToID     string
	Category string
	TypeID   int
}

// BatchCreateResult is the result for batch create (LabelsBetweenObjectPair).
type BatchCreateResult struct {
	FromObjectID     string
	FromObjectTypeID string
	ToObjectID       string
	ToObjectTypeID   string
	Labels           []domain.AssociationLabel
}

// BatchAssocResult is the result for a single pair in batch read operations.
type BatchAssocResult struct {
	From string                     `json:"from"`
	To   []domain.AssociationResult `json:"to"`
}

// BatchArchiveInput is a from/to pair for batch archive.
type BatchArchiveInput struct {
	From ObjectID `json:"from"`
	To   ObjectID `json:"to"`
}

// BatchArchiveLabelInput is a from/to pair with specific type IDs to remove.
type BatchArchiveLabelInput struct {
	From  ObjectID           `json:"from"`
	To    ObjectID           `json:"to"`
	Types []AssociationInput `json:"types"`
}

// SQLiteAssociationStore implements AssociationStore backed by SQLite.
type SQLiteAssociationStore struct {
	db *sql.DB
}

// NewSQLiteAssociationStore creates a new SQLiteAssociationStore.
func NewSQLiteAssociationStore(db *sql.DB) *SQLiteAssociationStore {
	return &SQLiteAssociationStore{db: db}
}

// Now returns the current UTC time as a HubSpot-compatible timestamp.
func Now() string { return now() }

func (s *SQLiteAssociationStore) resolveType(ctx context.Context, objectType string) (string, error) {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return "", fmt.Errorf("%s: %w", err.Error(), ErrNotFound)
	}
	return typeID, nil
}

func (s *SQLiteAssociationStore) getDefaultTypeID(ctx context.Context, fromType, toType string) (int, error) {
	var typeID int
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM association_types WHERE from_object_type = ? AND to_object_type = ? AND category = 'HUBSPOT_DEFINED' AND (label IS NULL OR label = '') ORDER BY id ASC LIMIT 1`,
		fromType, toType,
	).Scan(&typeID)
	if err != nil {
		return 0, fmt.Errorf("no default association type for %s→%s: %w", fromType, toType, ErrNotFound)
	}
	return typeID, nil
}

func (s *SQLiteAssociationStore) createReverseAssociation(ctx context.Context, fromTypeID, fromID, toTypeID, toID, ts string) {
	reverseTypeID, err := s.getDefaultTypeID(ctx, fromTypeID, toTypeID)
	if err != nil {
		return
	}
	_, _ = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
		fromID, toID, reverseTypeID, ts,
	)
}

func (s *SQLiteAssociationStore) objectExists(ctx context.Context, objectID string) bool {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM objects WHERE id = ? AND archived = FALSE`, objectID).Scan(&exists)
	return err == nil
}

// AssociateDefault creates a default (unlabeled) association between two objects.
func (s *SQLiteAssociationStore) AssociateDefault(ctx context.Context, fromType, fromID, toType, toID string) (*DefaultAssocResult, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	if !s.objectExists(ctx, fromID) {
		return nil, fmt.Errorf("object %s not found: %w", fromID, ErrNotFound)
	}
	if !s.objectExists(ctx, toID) {
		return nil, fmt.Errorf("object %s not found: %w", toID, ErrNotFound)
	}
	assocTypeID, err := s.getDefaultTypeID(ctx, fromTypeID, toTypeID)
	if err != nil {
		return nil, err
	}
	ts := now()
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
		fromID, toID, assocTypeID, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("create default association: %w", err)
	}
	s.createReverseAssociation(ctx, toTypeID, toID, fromTypeID, fromID, ts)
	return &DefaultAssocResult{Category: "HUBSPOT_DEFINED", TypeID: assocTypeID}, nil
}

// AssociateWithLabels creates labeled associations between two objects.
func (s *SQLiteAssociationStore) AssociateWithLabels(ctx context.Context, fromType, fromID, toType, toID string, types []AssociationInput) (*DefaultAssocResult, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	if !s.objectExists(ctx, fromID) {
		return nil, fmt.Errorf("object %s not found: %w", fromID, ErrNotFound)
	}
	if !s.objectExists(ctx, toID) {
		return nil, fmt.Errorf("object %s not found: %w", toID, ErrNotFound)
	}
	ts := now()
	defaultTypeID, err := s.getDefaultTypeID(ctx, fromTypeID, toTypeID)
	if err == nil {
		_, _ = s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
			fromID, toID, defaultTypeID, ts,
		)
	}
	for _, t := range types {
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM association_types WHERE id = ?`, t.AssociationTypeID).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("association type %d not found: %w", t.AssociationTypeID, ErrNotFound)
		}
		_, err = s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
			fromID, toID, t.AssociationTypeID, ts,
		)
		if err != nil {
			return nil, fmt.Errorf("create labeled association: %w", err)
		}
	}
	s.createReverseAssociation(ctx, toTypeID, toID, fromTypeID, fromID, ts)
	category := "HUBSPOT_DEFINED"
	typeID := defaultTypeID
	if len(types) > 0 {
		category = types[0].AssociationCategory
		typeID = types[0].AssociationTypeID
	}
	return &DefaultAssocResult{Category: category, TypeID: typeID}, nil
}

// GetAssociations returns all associations from one object to a target type.
func (s *SQLiteAssociationStore) GetAssociations(ctx context.Context, fromType, fromID, toType string) ([]domain.AssociationResult, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	return s.getAssocResults(ctx, fromTypeID, fromID, toTypeID)
}

// RemoveAssociations deletes all associations between two specific objects.
func (s *SQLiteAssociationStore) RemoveAssociations(ctx context.Context, fromType, fromID, toType, toID string) error {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM associations WHERE from_object_id = ? AND to_object_id = ? AND association_type_id IN (SELECT id FROM association_types WHERE from_object_type = ? AND to_object_type = ?)`,
		fromID, toID, fromTypeID, toTypeID,
	)
	if err != nil {
		return fmt.Errorf("remove associations: %w", err)
	}
	return nil
}

// ListLabels returns all association type labels between two object types.
func (s *SQLiteAssociationStore) ListLabels(ctx context.Context, fromType, toType string) ([]domain.AssociationLabel, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, category, COALESCE(label, '') FROM association_types WHERE from_object_type = ? AND to_object_type = ?`,
		fromTypeID, toTypeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var labels []domain.AssociationLabel
	for rows.Next() {
		var l domain.AssociationLabel
		if err := rows.Scan(&l.TypeID, &l.Category, &l.Label); err != nil {
			return nil, fmt.Errorf("scan label: %w", err)
		}
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

// CreateLabel creates a new association type label between two object types.
func (s *SQLiteAssociationStore) CreateLabel(ctx context.Context, fromType, toType, label, category string) (*domain.AssociationLabel, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	if category == "" {
		category = "USER_DEFINED"
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO association_types (from_object_type, to_object_type, category, label) VALUES (?, ?, ?, ?)`,
		fromTypeID, toTypeID, category, label,
	)
	if err != nil {
		return nil, fmt.Errorf("create label: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return &domain.AssociationLabel{Category: category, TypeID: int(id), Label: label}, nil
}

// UpdateLabel updates the label text of an existing association type.
func (s *SQLiteAssociationStore) UpdateLabel(ctx context.Context, fromType, toType string, typeID int, label string) (*domain.AssociationLabel, error) {
	_, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	_, err = s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE association_types SET label = ? WHERE id = ?`, label, typeID)
	if err != nil {
		return nil, fmt.Errorf("update label: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("association type %d: %w", typeID, ErrNotFound)
	}
	var category string
	err = s.db.QueryRowContext(ctx, `SELECT category FROM association_types WHERE id = ?`, typeID).Scan(&category)
	if err != nil {
		return nil, fmt.Errorf("get updated label: %w", err)
	}
	return &domain.AssociationLabel{Category: category, TypeID: typeID, Label: label}, nil
}

// DeleteLabel removes an association type and all its associations.
func (s *SQLiteAssociationStore) DeleteLabel(ctx context.Context, fromType, toType string, typeID int) error {
	_, err := s.resolveType(ctx, fromType)
	if err != nil {
		return err
	}
	_, err = s.resolveType(ctx, toType)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM associations WHERE association_type_id = ?`, typeID)
	if err != nil {
		return fmt.Errorf("remove associations for type: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM association_types WHERE id = ?`, typeID)
	if err != nil {
		return fmt.Errorf("delete label: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("association type %d: %w", typeID, ErrNotFound)
	}
	return nil
}

// BatchAssociateDefault creates default associations for multiple object pairs.
func (s *SQLiteAssociationStore) BatchAssociateDefault(ctx context.Context, fromType, toType string, inputs []BatchAssocInput) ([]BatchDefaultAssocResult, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	assocTypeID, err := s.getDefaultTypeID(ctx, fromTypeID, toTypeID)
	if err != nil {
		return nil, err
	}
	ts := now()
	var results []BatchDefaultAssocResult
	for _, input := range inputs {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
			input.From.ID, input.To.ID, assocTypeID, ts,
		)
		if err != nil {
			return nil, fmt.Errorf("batch default associate: %w", err)
		}
		s.createReverseAssociation(ctx, toTypeID, input.To.ID, fromTypeID, input.From.ID, ts)
		results = append(results, BatchDefaultAssocResult{
			FromID: input.From.ID, ToID: input.To.ID, Category: "HUBSPOT_DEFINED", TypeID: assocTypeID,
		})
	}
	return results, nil
}

// BatchCreate creates labeled associations for multiple object pairs.
func (s *SQLiteAssociationStore) BatchCreate(ctx context.Context, fromType, toType string, inputs []BatchAssocCreateInput) ([]BatchCreateResult, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	ts := now()
	var results []BatchCreateResult
	for _, input := range inputs {
		defaultTypeID, defErr := s.getDefaultTypeID(ctx, fromTypeID, toTypeID)
		if defErr == nil {
			_, _ = s.db.ExecContext(ctx,
				`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
				input.From.ID, input.To.ID, defaultTypeID, ts,
			)
		}
		var labels []domain.AssociationLabel
		for _, t := range input.Types {
			_, err := s.db.ExecContext(ctx,
				`INSERT OR IGNORE INTO associations (from_object_id, to_object_id, association_type_id, created_at) VALUES (?, ?, ?, ?)`,
				input.From.ID, input.To.ID, t.AssociationTypeID, ts,
			)
			if err != nil {
				return nil, fmt.Errorf("batch create association: %w", err)
			}
			var category string
			var label sql.NullString
			_ = s.db.QueryRowContext(ctx, `SELECT category, label FROM association_types WHERE id = ?`, t.AssociationTypeID).Scan(&category, &label)
			labels = append(labels, domain.AssociationLabel{TypeID: t.AssociationTypeID, Category: category, Label: label.String})
		}
		s.createReverseAssociation(ctx, toTypeID, input.To.ID, fromTypeID, input.From.ID, ts)
		results = append(results, BatchCreateResult{
			FromObjectID: input.From.ID, FromObjectTypeID: fromTypeID,
			ToObjectID: input.To.ID, ToObjectTypeID: toTypeID, Labels: labels,
		})
	}
	return results, nil
}

// BatchRead retrieves associations for multiple objects in a single call.
func (s *SQLiteAssociationStore) BatchRead(ctx context.Context, fromType, toType string, inputs []BatchAssocReadInput) ([]BatchAssocResult, error) {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return nil, err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return nil, err
	}
	var results []BatchAssocResult
	for _, input := range inputs {
		assocs, err := s.getAssocResults(ctx, fromTypeID, input.ID, toTypeID)
		if err != nil {
			return nil, err
		}
		results = append(results, BatchAssocResult{From: input.ID, To: assocs})
	}
	return results, nil
}

// BatchArchive removes all associations for multiple object pairs.
func (s *SQLiteAssociationStore) BatchArchive(ctx context.Context, fromType, toType string, inputs []BatchArchiveInput) error {
	fromTypeID, err := s.resolveType(ctx, fromType)
	if err != nil {
		return err
	}
	toTypeID, err := s.resolveType(ctx, toType)
	if err != nil {
		return err
	}
	for _, input := range inputs {
		_, err := s.db.ExecContext(ctx,
			`DELETE FROM associations WHERE from_object_id = ? AND to_object_id = ? AND association_type_id IN (SELECT id FROM association_types WHERE from_object_type = ? AND to_object_type = ?)`,
			input.From.ID, input.To.ID, fromTypeID, toTypeID,
		)
		if err != nil {
			return fmt.Errorf("batch archive association: %w", err)
		}
	}
	return nil
}

// BatchArchiveLabels removes specific labeled associations for multiple object pairs.
func (s *SQLiteAssociationStore) BatchArchiveLabels(ctx context.Context, fromType, toType string, inputs []BatchArchiveLabelInput) error {
	_, err := s.resolveType(ctx, fromType)
	if err != nil {
		return err
	}
	_, err = s.resolveType(ctx, toType)
	if err != nil {
		return err
	}
	for _, input := range inputs {
		for _, t := range input.Types {
			_, err := s.db.ExecContext(ctx,
				`DELETE FROM associations WHERE from_object_id = ? AND to_object_id = ? AND association_type_id = ?`,
				input.From.ID, input.To.ID, t.AssociationTypeID,
			)
			if err != nil {
				return fmt.Errorf("batch archive label: %w", err)
			}
		}
	}
	return nil
}

func (s *SQLiteAssociationStore) getAssocResults(ctx context.Context, fromTypeID, fromID, toTypeID string) ([]domain.AssociationResult, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.to_object_id, at.id, at.category, COALESCE(at.label, '') FROM associations a JOIN association_types at ON at.id = a.association_type_id WHERE a.from_object_id = ? AND at.from_object_type = ? AND at.to_object_type = ? ORDER BY a.to_object_id, at.id`,
		fromID, fromTypeID, toTypeID,
	)
	if err != nil {
		return nil, fmt.Errorf("get associations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	resultMap := make(map[string]*domain.AssociationResult)
	var order []string
	for rows.Next() {
		var toID, category, label string
		var typeID int
		if err := rows.Scan(&toID, &typeID, &category, &label); err != nil {
			return nil, fmt.Errorf("scan association: %w", err)
		}
		if _, ok := resultMap[toID]; !ok {
			resultMap[toID] = &domain.AssociationResult{ToObjectID: toID}
			order = append(order, toID)
		}
		resultMap[toID].Types = append(resultMap[toID].Types, domain.AssociationType{TypeID: typeID, Category: category, Label: label})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	results := make([]domain.AssociationResult, 0, len(order))
	for _, id := range order {
		results = append(results, *resultMap[id])
	}
	return results, nil
}

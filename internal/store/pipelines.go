package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/johnwards/hubspot/internal/domain"
)

// PipelineStore defines operations for managing pipelines and their stages.
type PipelineStore interface {
	List(ctx context.Context, objectType string) ([]domain.Pipeline, error)
	Create(ctx context.Context, objectType string, p *domain.Pipeline) (*domain.Pipeline, error)
	Get(ctx context.Context, objectType, id string) (*domain.Pipeline, error)
	Update(ctx context.Context, objectType, id string, p *domain.Pipeline) (*domain.Pipeline, error)
	Replace(ctx context.Context, objectType, id string, p *domain.Pipeline) (*domain.Pipeline, error)
	Delete(ctx context.Context, objectType, id string) error
	ListStages(ctx context.Context, objectType, pipelineID string) ([]domain.PipelineStage, error)
	CreateStage(ctx context.Context, objectType, pipelineID string, s *domain.PipelineStage) (*domain.PipelineStage, error)
	GetStage(ctx context.Context, objectType, pipelineID, stageID string) (*domain.PipelineStage, error)
	UpdateStage(ctx context.Context, objectType, pipelineID, stageID string, s *domain.PipelineStage) (*domain.PipelineStage, error)
	ReplaceStage(ctx context.Context, objectType, pipelineID, stageID string, s *domain.PipelineStage) (*domain.PipelineStage, error)
	DeleteStage(ctx context.Context, objectType, pipelineID, stageID string) error
}

// SQLitePipelineStore implements PipelineStore backed by SQLite.
type SQLitePipelineStore struct {
	db *sql.DB
}

// NewSQLitePipelineStore creates a new SQLitePipelineStore.
func NewSQLitePipelineStore(db *sql.DB) *SQLitePipelineStore {
	return &SQLitePipelineStore{db: db}
}

// List returns all pipelines for the given object type, including stages.
func (s *SQLitePipelineStore) List(ctx context.Context, objectType string) ([]domain.Pipeline, error) {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, label, display_order, archived, created_at, updated_at
		 FROM pipelines WHERE object_type_id = ? ORDER BY display_order`,
		typeID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	var pipelines []domain.Pipeline
	for rows.Next() {
		var p domain.Pipeline
		if err := rows.Scan(&p.ID, &p.Label, &p.DisplayOrder, &p.Archived, &p.CreatedAt, &p.UpdatedAt); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		pipelines = append(pipelines, p)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	// Load stages in a separate pass to avoid holding the rows cursor
	// (SQLite MaxOpenConns=1).
	for i := range pipelines {
		stages, err := s.loadStages(ctx, pipelines[i].ID)
		if err != nil {
			return nil, err
		}
		pipelines[i].Stages = stages
	}
	return pipelines, nil
}

// Create inserts a new pipeline for the given object type.
func (s *SQLitePipelineStore) Create(ctx context.Context, objectType string, p *domain.Pipeline) (*domain.Pipeline, error) {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return nil, err
	}

	ts := now()
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO pipelines (object_type_id, label, display_order, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		typeID, p.Label, p.DisplayOrder, false, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	id, _ := result.LastInsertId()
	p.ID = strconv.FormatInt(id, 10)
	p.Archived = false
	p.CreatedAt = ts
	p.UpdatedAt = ts

	for i := range p.Stages {
		created, err := s.createStageRow(ctx, p.ID, &p.Stages[i])
		if err != nil {
			return nil, err
		}
		p.Stages[i] = *created
	}

	return p, nil
}

// Get returns a single pipeline by ID, including stages.
func (s *SQLitePipelineStore) Get(ctx context.Context, objectType, id string) (*domain.Pipeline, error) {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return nil, err
	}

	var p domain.Pipeline
	err = s.db.QueryRowContext(ctx,
		`SELECT id, label, display_order, archived, created_at, updated_at
		 FROM pipelines WHERE id = ? AND object_type_id = ?`,
		id, typeID,
	).Scan(&p.ID, &p.Label, &p.DisplayOrder, &p.Archived, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("pipeline %q not found", id)
		}
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	stages, err := s.loadStages(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Stages = stages
	return &p, nil
}

// Update partially updates a pipeline (PATCH semantics).
func (s *SQLitePipelineStore) Update(ctx context.Context, objectType, id string, p *domain.Pipeline) (*domain.Pipeline, error) {
	existing, err := s.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}

	if p.Label != "" {
		existing.Label = p.Label
	}
	if p.DisplayOrder != 0 {
		existing.DisplayOrder = p.DisplayOrder
	}
	existing.UpdatedAt = now()

	_, err = s.db.ExecContext(ctx,
		`UPDATE pipelines SET label = ?, display_order = ?, updated_at = ? WHERE id = ?`,
		existing.Label, existing.DisplayOrder, existing.UpdatedAt, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update pipeline: %w", err)
	}
	return existing, nil
}

// Replace fully replaces a pipeline (PUT semantics).
func (s *SQLitePipelineStore) Replace(ctx context.Context, objectType, id string, p *domain.Pipeline) (*domain.Pipeline, error) {
	existing, err := s.Get(ctx, objectType, id)
	if err != nil {
		return nil, err
	}

	ts := now()
	_, err = s.db.ExecContext(ctx,
		`UPDATE pipelines SET label = ?, display_order = ?, updated_at = ? WHERE id = ?`,
		p.Label, p.DisplayOrder, ts, id,
	)
	if err != nil {
		return nil, fmt.Errorf("replace pipeline: %w", err)
	}

	existing.Label = p.Label
	existing.DisplayOrder = p.DisplayOrder
	existing.UpdatedAt = ts

	if p.Stages != nil {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM pipeline_stages WHERE pipeline_id = ?`, id); err != nil {
			return nil, fmt.Errorf("delete stages for replace: %w", err)
		}
		var newStages []domain.PipelineStage
		for i := range p.Stages {
			created, err := s.createStageRow(ctx, id, &p.Stages[i])
			if err != nil {
				return nil, err
			}
			newStages = append(newStages, *created)
		}
		existing.Stages = newStages
	}

	return existing, nil
}

// Delete removes a pipeline and its stages.
func (s *SQLitePipelineStore) Delete(ctx context.Context, objectType, id string) error {
	typeID, err := ResolveObjectType(ctx, s.db, objectType)
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM pipeline_stages WHERE pipeline_id = ?`, id); err != nil {
		return fmt.Errorf("delete pipeline stages: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		`DELETE FROM pipelines WHERE id = ? AND object_type_id = ?`,
		id, typeID,
	)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline %q not found", id)
	}
	return nil
}

// ListStages returns all stages for a pipeline.
func (s *SQLitePipelineStore) ListStages(ctx context.Context, objectType, pipelineID string) ([]domain.PipelineStage, error) {
	if _, err := s.Get(ctx, objectType, pipelineID); err != nil {
		return nil, err
	}
	return s.loadStages(ctx, pipelineID)
}

// CreateStage adds a new stage to a pipeline.
func (s *SQLitePipelineStore) CreateStage(ctx context.Context, objectType, pipelineID string, st *domain.PipelineStage) (*domain.PipelineStage, error) {
	if _, err := s.Get(ctx, objectType, pipelineID); err != nil {
		return nil, err
	}
	return s.createStageRow(ctx, pipelineID, st)
}

// GetStage returns a single stage by ID.
func (s *SQLitePipelineStore) GetStage(ctx context.Context, objectType, pipelineID, stageID string) (*domain.PipelineStage, error) {
	if _, err := s.Get(ctx, objectType, pipelineID); err != nil {
		return nil, err
	}

	var st domain.PipelineStage
	var metaJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, label, display_order, metadata, archived, created_at, updated_at
		 FROM pipeline_stages WHERE id = ? AND pipeline_id = ?`,
		stageID, pipelineID,
	).Scan(&st.ID, &st.Label, &st.DisplayOrder, &metaJSON, &st.Archived, &st.CreatedAt, &st.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("stage %q not found", stageID)
		}
		return nil, fmt.Errorf("get stage: %w", err)
	}
	if err := json.Unmarshal([]byte(metaJSON), &st.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal stage metadata: %w", err)
	}
	return &st, nil
}

// UpdateStage partially updates a stage (PATCH semantics).
func (s *SQLitePipelineStore) UpdateStage(ctx context.Context, objectType, pipelineID, stageID string, st *domain.PipelineStage) (*domain.PipelineStage, error) {
	existing, err := s.GetStage(ctx, objectType, pipelineID, stageID)
	if err != nil {
		return nil, err
	}

	if st.Label != "" {
		existing.Label = st.Label
	}
	if st.DisplayOrder != 0 {
		existing.DisplayOrder = st.DisplayOrder
	}
	if st.Metadata != nil {
		existing.Metadata = st.Metadata
	}
	existing.UpdatedAt = now()

	metaJSON, err := json.Marshal(existing.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE pipeline_stages SET label = ?, display_order = ?, metadata = ?, updated_at = ? WHERE id = ?`,
		existing.Label, existing.DisplayOrder, string(metaJSON), existing.UpdatedAt, stageID,
	)
	if err != nil {
		return nil, fmt.Errorf("update stage: %w", err)
	}
	return existing, nil
}

// ReplaceStage fully replaces a stage (PUT semantics).
func (s *SQLitePipelineStore) ReplaceStage(ctx context.Context, objectType, pipelineID, stageID string, st *domain.PipelineStage) (*domain.PipelineStage, error) {
	existing, err := s.GetStage(ctx, objectType, pipelineID, stageID)
	if err != nil {
		return nil, err
	}

	ts := now()
	metaJSON, err := json.Marshal(st.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE pipeline_stages SET label = ?, display_order = ?, metadata = ?, updated_at = ? WHERE id = ?`,
		st.Label, st.DisplayOrder, string(metaJSON), ts, stageID,
	)
	if err != nil {
		return nil, fmt.Errorf("replace stage: %w", err)
	}

	existing.Label = st.Label
	existing.DisplayOrder = st.DisplayOrder
	existing.Metadata = st.Metadata
	existing.UpdatedAt = ts
	return existing, nil
}

// DeleteStage removes a stage from a pipeline.
func (s *SQLitePipelineStore) DeleteStage(ctx context.Context, objectType, pipelineID, stageID string) error {
	if _, err := s.Get(ctx, objectType, pipelineID); err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx,
		`DELETE FROM pipeline_stages WHERE id = ? AND pipeline_id = ?`,
		stageID, pipelineID,
	)
	if err != nil {
		return fmt.Errorf("delete stage: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("stage %q not found", stageID)
	}
	return nil
}

func (s *SQLitePipelineStore) loadStages(ctx context.Context, pipelineID string) ([]domain.PipelineStage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, label, display_order, metadata, archived, created_at, updated_at
		 FROM pipeline_stages WHERE pipeline_id = ? ORDER BY display_order`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("load stages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stages []domain.PipelineStage
	for rows.Next() {
		var st domain.PipelineStage
		var metaJSON string
		if err := rows.Scan(&st.ID, &st.Label, &st.DisplayOrder, &metaJSON, &st.Archived, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}
		if err := json.Unmarshal([]byte(metaJSON), &st.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal stage metadata: %w", err)
		}
		stages = append(stages, st)
	}
	if stages == nil {
		stages = []domain.PipelineStage{}
	}
	return stages, rows.Err()
}

func (s *SQLitePipelineStore) createStageRow(ctx context.Context, pipelineID string, st *domain.PipelineStage) (*domain.PipelineStage, error) {
	ts := now()
	meta := st.Metadata
	if meta == nil {
		meta = map[string]string{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO pipeline_stages (pipeline_id, label, display_order, metadata, archived, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		pipelineID, st.Label, st.DisplayOrder, string(metaJSON), false, ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("create stage: %w", err)
	}

	id, _ := result.LastInsertId()
	return &domain.PipelineStage{
		ID:           strconv.FormatInt(id, 10),
		Label:        st.Label,
		DisplayOrder: st.DisplayOrder,
		Metadata:     meta,
		Archived:     false,
		CreatedAt:    ts,
		UpdatedAt:    ts,
	}, nil
}

package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupPipelineTest(t *testing.T) (*store.SQLitePipelineStore, context.Context) {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Seed a "deals" object type for testing.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO object_types (id, name, label_singular, label_plural, created_at, updated_at)
		 VALUES ('0-3', 'deals', 'Deal', 'Deals', '2024-01-01T00:00:00.000Z', '2024-01-01T00:00:00.000Z')`,
	); err != nil {
		t.Fatalf("seed object type: %v", err)
	}
	return store.NewSQLitePipelineStore(db), ctx
}

func TestPipelineStore_CreateAndGet(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	p := &domain.Pipeline{
		Label:        "Sales Pipeline",
		DisplayOrder: 0,
		Stages: []domain.PipelineStage{
			{Label: "Stage 1", DisplayOrder: 0, Metadata: map[string]string{"probability": "0.5"}},
			{Label: "Stage 2", DisplayOrder: 1, Metadata: map[string]string{"probability": "1.0"}},
		},
	}

	created, err := s.Create(ctx, "deals", p)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	if created.Label != "Sales Pipeline" {
		t.Errorf("expected label 'Sales Pipeline', got %q", created.Label)
	}
	if len(created.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(created.Stages))
	}
	if created.Stages[0].Label != "Stage 1" {
		t.Errorf("expected stage label 'Stage 1', got %q", created.Stages[0].Label)
	}

	// Get by ID.
	got, err := s.Get(ctx, "deals", created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "Sales Pipeline" {
		t.Errorf("expected label 'Sales Pipeline', got %q", got.Label)
	}
	if len(got.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(got.Stages))
	}
}

func TestPipelineStore_CreateAndGetByObjectTypeID(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	p := &domain.Pipeline{Label: "Test Pipeline", DisplayOrder: 0}
	created, err := s.Create(ctx, "0-3", p)
	if err != nil {
		t.Fatalf("Create by type ID: %v", err)
	}

	got, err := s.Get(ctx, "0-3", created.ID)
	if err != nil {
		t.Fatalf("Get by type ID: %v", err)
	}
	if got.Label != "Test Pipeline" {
		t.Errorf("expected label 'Test Pipeline', got %q", got.Label)
	}
}

func TestPipelineStore_List(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	// Empty list.
	pipelines, err := s.List(ctx, "deals")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines, got %d", len(pipelines))
	}

	// Create two pipelines.
	if _, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "Pipeline A", DisplayOrder: 1}); err != nil {
		t.Fatalf("Create A: %v", err)
	}
	if _, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "Pipeline B", DisplayOrder: 0}); err != nil {
		t.Fatalf("Create B: %v", err)
	}

	pipelines, err = s.List(ctx, "deals")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(pipelines) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(pipelines))
	}
	// Ordered by display_order.
	if pipelines[0].Label != "Pipeline B" {
		t.Errorf("expected first pipeline 'Pipeline B', got %q", pipelines[0].Label)
	}
}

func TestPipelineStore_Update(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	created, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "Old Name", DisplayOrder: 0})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := s.Update(ctx, "deals", created.ID, &domain.Pipeline{Label: "New Name"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Label != "New Name" {
		t.Errorf("expected label 'New Name', got %q", updated.Label)
	}
}

func TestPipelineStore_Replace(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	created, err := s.Create(ctx, "deals", &domain.Pipeline{
		Label:        "Original",
		DisplayOrder: 0,
		Stages: []domain.PipelineStage{
			{Label: "Old Stage", DisplayOrder: 0, Metadata: map[string]string{}},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	replaced, err := s.Replace(ctx, "deals", created.ID, &domain.Pipeline{
		Label:        "Replaced",
		DisplayOrder: 5,
		Stages: []domain.PipelineStage{
			{Label: "New Stage A", DisplayOrder: 0, Metadata: map[string]string{"key": "val"}},
			{Label: "New Stage B", DisplayOrder: 1, Metadata: map[string]string{}},
		},
	})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if replaced.Label != "Replaced" {
		t.Errorf("expected label 'Replaced', got %q", replaced.Label)
	}
	if len(replaced.Stages) != 2 {
		t.Fatalf("expected 2 stages after replace, got %d", len(replaced.Stages))
	}
}

func TestPipelineStore_Delete(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	created, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "To Delete", DisplayOrder: 0})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Delete(ctx, "deals", created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should not be found.
	if _, err := s.Get(ctx, "deals", created.ID); err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestPipelineStore_DeleteNotFound(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	err := s.Delete(ctx, "deals", "999")
	if err == nil {
		t.Error("expected error deleting non-existent pipeline")
	}
}

func TestPipelineStore_GetNotFound(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	_, err := s.Get(ctx, "deals", "999")
	if err == nil {
		t.Error("expected error getting non-existent pipeline")
	}
}

func TestPipelineStore_InvalidObjectType(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	_, err := s.List(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for invalid object type")
	}
}

func TestPipelineStore_StagesCRUD(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	p, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "Test", DisplayOrder: 0})
	if err != nil {
		t.Fatalf("Create pipeline: %v", err)
	}

	// CreateStage.
	stage, err := s.CreateStage(ctx, "deals", p.ID, &domain.PipelineStage{
		Label:        "Stage A",
		DisplayOrder: 0,
		Metadata:     map[string]string{"probability": "0.5"},
	})
	if err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if stage.ID == "" {
		t.Error("expected non-empty stage ID")
	}
	if stage.Metadata["probability"] != "0.5" {
		t.Errorf("expected metadata probability '0.5', got %q", stage.Metadata["probability"])
	}

	// ListStages.
	stages, err := s.ListStages(ctx, "deals", p.ID)
	if err != nil {
		t.Fatalf("ListStages: %v", err)
	}
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(stages))
	}

	// GetStage.
	got, err := s.GetStage(ctx, "deals", p.ID, stage.ID)
	if err != nil {
		t.Fatalf("GetStage: %v", err)
	}
	if got.Label != "Stage A" {
		t.Errorf("expected label 'Stage A', got %q", got.Label)
	}

	// UpdateStage.
	updated, err := s.UpdateStage(ctx, "deals", p.ID, stage.ID, &domain.PipelineStage{Label: "Stage B"})
	if err != nil {
		t.Fatalf("UpdateStage: %v", err)
	}
	if updated.Label != "Stage B" {
		t.Errorf("expected label 'Stage B', got %q", updated.Label)
	}
	// Metadata should be preserved.
	if updated.Metadata["probability"] != "0.5" {
		t.Errorf("expected metadata preserved, got %v", updated.Metadata)
	}

	// ReplaceStage.
	replaced, err := s.ReplaceStage(ctx, "deals", p.ID, stage.ID, &domain.PipelineStage{
		Label:        "Stage C",
		DisplayOrder: 3,
		Metadata:     map[string]string{"new": "value"},
	})
	if err != nil {
		t.Fatalf("ReplaceStage: %v", err)
	}
	if replaced.Label != "Stage C" {
		t.Errorf("expected label 'Stage C', got %q", replaced.Label)
	}
	if replaced.Metadata["new"] != "value" {
		t.Errorf("expected new metadata, got %v", replaced.Metadata)
	}

	// DeleteStage.
	if err := s.DeleteStage(ctx, "deals", p.ID, stage.ID); err != nil {
		t.Fatalf("DeleteStage: %v", err)
	}

	// Should be empty now.
	stages, err = s.ListStages(ctx, "deals", p.ID)
	if err != nil {
		t.Fatalf("ListStages after delete: %v", err)
	}
	if len(stages) != 0 {
		t.Errorf("expected 0 stages, got %d", len(stages))
	}
}

func TestPipelineStore_StageNotFound(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	p, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "Test", DisplayOrder: 0})
	if err != nil {
		t.Fatalf("Create pipeline: %v", err)
	}

	if _, err := s.GetStage(ctx, "deals", p.ID, "999"); err == nil {
		t.Error("expected error getting non-existent stage")
	}

	if err := s.DeleteStage(ctx, "deals", p.ID, "999"); err == nil {
		t.Error("expected error deleting non-existent stage")
	}
}

func TestPipelineStore_NilMetadata(t *testing.T) {
	s, ctx := setupPipelineTest(t)

	p, err := s.Create(ctx, "deals", &domain.Pipeline{Label: "Test", DisplayOrder: 0})
	if err != nil {
		t.Fatalf("Create pipeline: %v", err)
	}

	// Create a stage without metadata — should default to empty map.
	stage, err := s.CreateStage(ctx, "deals", p.ID, &domain.PipelineStage{
		Label:        "No Meta",
		DisplayOrder: 0,
	})
	if err != nil {
		t.Fatalf("CreateStage: %v", err)
	}
	if stage.Metadata == nil {
		t.Error("expected non-nil metadata map")
	}
}

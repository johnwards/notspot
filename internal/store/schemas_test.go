package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupSchemaStore(t *testing.T) (*store.SQLiteSchemaStore, context.Context) {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store.NewSQLiteSchemaStore(db), ctx
}

func createTestSchema(t *testing.T, s *store.SQLiteSchemaStore, ctx context.Context, name string) *domain.ObjectSchema {
	t.Helper()
	schema, err := s.Create(ctx, &domain.ObjectSchema{
		Name:                   name,
		Labels:                 domain.SchemaLabels{Singular: name, Plural: name + "s"},
		PrimaryDisplayProperty: "hs_object_id",
	})
	if err != nil {
		t.Fatalf("create schema %q: %v", name, err)
	}
	return schema
}

func TestSchemaStore_Create(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	schema, err := s.Create(ctx, &domain.ObjectSchema{
		Name:                   "cars",
		Labels:                 domain.SchemaLabels{Singular: "Car", Plural: "Cars"},
		PrimaryDisplayProperty: "hs_object_id",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if schema.ID != "2-1" {
		t.Errorf("id = %q, want 2-1", schema.ID)
	}
	if schema.Name != "cars" {
		t.Errorf("name = %q, want cars", schema.Name)
	}
	if schema.FullyQualifiedName != "p0_cars" {
		t.Errorf("fqn = %q, want p0_cars", schema.FullyQualifiedName)
	}
	if schema.Labels.Singular != "Car" {
		t.Errorf("singular = %q, want Car", schema.Labels.Singular)
	}
	if len(schema.Properties) != 3 {
		t.Errorf("len(properties) = %d, want 3", len(schema.Properties))
	}
}

func TestSchemaStore_Create_AutoIncrementID(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	s1 := createTestSchema(t, s, ctx, "cars")
	s2 := createTestSchema(t, s, ctx, "trucks")

	if s1.ID != "2-1" {
		t.Errorf("first id = %q, want 2-1", s1.ID)
	}
	if s2.ID != "2-2" {
		t.Errorf("second id = %q, want 2-2", s2.ID)
	}
}

func TestSchemaStore_Create_DuplicateName(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	createTestSchema(t, s, ctx, "cars")

	_, err := s.Create(ctx, &domain.ObjectSchema{
		Name:   "cars",
		Labels: domain.SchemaLabels{Singular: "Car", Plural: "Cars"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestSchemaStore_Create_EmptyName(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	_, err := s.Create(ctx, &domain.ObjectSchema{
		Labels: domain.SchemaLabels{Singular: "Car", Plural: "Cars"},
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSchemaStore_List(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	schemas, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("len = %d, want 0", len(schemas))
	}

	createTestSchema(t, s, ctx, "cars")
	createTestSchema(t, s, ctx, "trucks")

	schemas, err = s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(schemas) != 2 {
		t.Errorf("len = %d, want 2", len(schemas))
	}
}

func TestSchemaStore_Get_ByName(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	createTestSchema(t, s, ctx, "cars")

	schema, err := s.Get(ctx, "cars")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if schema.Name != "cars" {
		t.Errorf("name = %q, want cars", schema.Name)
	}
}

func TestSchemaStore_Get_ByID(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	created := createTestSchema(t, s, ctx, "cars")

	schema, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if schema.Name != "cars" {
		t.Errorf("name = %q, want cars", schema.Name)
	}
}

func TestSchemaStore_Get_NotFound(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	_, err := s.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestSchemaStore_Update(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	createTestSchema(t, s, ctx, "cars")

	updated, err := s.Update(ctx, "cars", &domain.ObjectSchema{
		Labels: domain.SchemaLabels{Singular: "Automobile", Plural: "Automobiles"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Labels.Singular != "Automobile" {
		t.Errorf("singular = %q, want Automobile", updated.Labels.Singular)
	}
	if updated.Labels.Plural != "Automobiles" {
		t.Errorf("plural = %q, want Automobiles", updated.Labels.Plural)
	}
}

func TestSchemaStore_Update_NotFound(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	_, err := s.Update(ctx, "nonexistent", &domain.ObjectSchema{
		Labels: domain.SchemaLabels{Singular: "X", Plural: "Xs"},
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestSchemaStore_Archive(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	createTestSchema(t, s, ctx, "cars")

	if err := s.Archive(ctx, "cars"); err != nil {
		t.Fatalf("archive: %v", err)
	}

	// Should not appear in list
	schemas, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("len = %d, want 0 after archive", len(schemas))
	}
}

func TestSchemaStore_Archive_NotFound(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	err := s.Archive(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestSchemaStore_DefaultProperties(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	schema := createTestSchema(t, s, ctx, "cars")

	expectedNames := map[string]bool{
		"hs_object_id":        false,
		"hs_createdate":       false,
		"hs_lastmodifieddate": false,
	}

	for _, p := range schema.Properties {
		if _, ok := expectedNames[p.Name]; ok {
			expectedNames[p.Name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected default property %q not found", name)
		}
	}
}

func TestSchemaStore_CreateAssociation(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	// We need a target object type. Seed a built-in one.
	db := testhelpers.NewTestDB(t)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Use a fresh store that shares the same DB as the schema store.
	// Instead, let's create both the schema and a target type in the same store.
	// We need to insert a standard type into the DB backing the schema store.
	// Since SQLiteSchemaStore doesn't expose the db, we'll create two custom types
	// and associate them.
	createTestSchema(t, s, ctx, "cars")
	createTestSchema(t, s, ctx, "drivers")

	assoc, err := s.CreateAssociation(ctx, "cars", &domain.SchemaAssociation{
		ToObjectTypeID: "2-2",
		Name:           "car_to_driver",
	})
	if err != nil {
		t.Fatalf("create association: %v", err)
	}
	if assoc.ID == "" {
		t.Error("association ID is empty")
	}
	if assoc.FromObjectTypeID != "2-1" {
		t.Errorf("fromObjectTypeId = %q, want 2-1", assoc.FromObjectTypeID)
	}

	// Verify it appears on the schema
	schema, err := s.Get(ctx, "cars")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(schema.Associations) != 1 {
		t.Fatalf("len(associations) = %d, want 1", len(schema.Associations))
	}
}

func TestSchemaStore_DeleteAssociation(t *testing.T) {
	s, ctx := setupSchemaStore(t)

	createTestSchema(t, s, ctx, "cars")
	createTestSchema(t, s, ctx, "drivers")

	assoc, err := s.CreateAssociation(ctx, "cars", &domain.SchemaAssociation{
		ToObjectTypeID: "2-2",
		Name:           "car_to_driver",
	})
	if err != nil {
		t.Fatalf("create association: %v", err)
	}

	if err := s.DeleteAssociation(ctx, "cars", assoc.ID); err != nil {
		t.Fatalf("delete association: %v", err)
	}

	schema, err := s.Get(ctx, "cars")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(schema.Associations) != 0 {
		t.Errorf("len(associations) = %d, want 0", len(schema.Associations))
	}
}

func TestSchemaStore_DeleteAssociation_NotFound(t *testing.T) {
	s, ctx := setupSchemaStore(t)
	createTestSchema(t, s, ctx, "cars")

	err := s.DeleteAssociation(ctx, "cars", "999")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

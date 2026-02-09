package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupAssocStore(t *testing.T) (store.AssociationStore, *store.SQLiteObjectStore, context.Context) {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Seed(ctx, db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := seed.AssociationTypes(ctx, db); err != nil {
		t.Fatalf("seed assoc types: %v", err)
	}
	return store.NewSQLiteAssociationStore(db), store.NewSQLiteObjectStore(db), ctx
}

func createTestObject(t *testing.T, objStore *store.SQLiteObjectStore, ctx context.Context, objectType string) string {
	t.Helper()
	obj, err := objStore.Create(ctx, objectType, map[string]string{"test": "value"})
	if err != nil {
		t.Fatalf("create %s: %v", objectType, err)
	}
	return obj.ID
}

func TestAssociateDefault(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	result, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("associate default: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Category != "HUBSPOT_DEFINED" {
		t.Errorf("expected category HUBSPOT_DEFINED, got %s", result.Category)
	}
	if result.TypeID == 0 {
		t.Error("expected non-zero TypeID")
	}

	// Verify the association exists via GetAssociations.
	assocs, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get associations: %v", err)
	}
	if len(assocs) != 1 {
		t.Fatalf("expected 1 result, got %d", len(assocs))
	}
	if assocs[0].ToObjectID != companyID {
		t.Errorf("expected toObjectId %s, got %s", companyID, assocs[0].ToObjectID)
	}
}

func TestAssociateDefaultIdempotent(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	_, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("first associate: %v", err)
	}

	// Second call should not error.
	result, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("second associate: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestAssociateWithLabels(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	types := []store.AssociationInput{
		{AssociationCategory: "HUBSPOT_DEFINED", AssociationTypeID: 279}, // Primary
	}

	result, err := assocStore.AssociateWithLabels(ctx, "contacts", contactID, "companies", companyID, types)
	if err != nil {
		t.Fatalf("associate with labels: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify both the default and Primary associations exist.
	assocs, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get associations: %v", err)
	}
	if len(assocs) != 1 {
		t.Fatalf("expected 1 result, got %d", len(assocs))
	}
	if len(assocs[0].Types) < 2 {
		t.Fatalf("expected at least 2 types (default + labeled), got %d", len(assocs[0].Types))
	}
}

func TestGetAssociations(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	_, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("associate: %v", err)
	}

	results, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToObjectID != companyID {
		t.Errorf("expected toObjectId %s, got %s", companyID, results[0].ToObjectID)
	}
}

func TestGetAssociationsEmpty(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")

	results, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestRemoveAssociations(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	_, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("associate: %v", err)
	}

	err = assocStore.RemoveAssociations(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	results, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get after remove: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results after remove, got %d", len(results))
	}
}

func TestListLabels(t *testing.T) {
	assocStore, _, ctx := setupAssocStore(t)

	labels, err := assocStore.ListLabels(ctx, "contacts", "companies")
	if err != nil {
		t.Fatalf("list labels: %v", err)
	}

	// Should have at least the seeded types (1, 279).
	if len(labels) < 2 {
		t.Fatalf("expected at least 2 labels, got %d", len(labels))
	}
}

func TestCreateLabel(t *testing.T) {
	assocStore, _, ctx := setupAssocStore(t)

	label, err := assocStore.CreateLabel(ctx, "contacts", "companies", "Partner", "USER_DEFINED")
	if err != nil {
		t.Fatalf("create label: %v", err)
	}

	if label.Label != "Partner" {
		t.Errorf("expected label 'Partner', got %q", label.Label)
	}
	if label.Category != "USER_DEFINED" {
		t.Errorf("expected category USER_DEFINED, got %s", label.Category)
	}
	if label.TypeID == 0 {
		t.Error("expected non-zero typeId")
	}
}

func TestUpdateLabel(t *testing.T) {
	assocStore, _, ctx := setupAssocStore(t)

	created, err := assocStore.CreateLabel(ctx, "contacts", "companies", "Old", "USER_DEFINED")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := assocStore.UpdateLabel(ctx, "contacts", "companies", created.TypeID, "New")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Label != "New" {
		t.Errorf("expected label 'New', got %q", updated.Label)
	}
}

func TestDeleteLabel(t *testing.T) {
	assocStore, _, ctx := setupAssocStore(t)

	created, err := assocStore.CreateLabel(ctx, "contacts", "companies", "Temp", "USER_DEFINED")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = assocStore.DeleteLabel(ctx, "contacts", "companies", created.TypeID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify it's gone.
	labels, err := assocStore.ListLabels(ctx, "contacts", "companies")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, l := range labels {
		if l.TypeID == created.TypeID {
			t.Error("deleted label still present")
		}
	}
}

func TestBatchAssociateDefault(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	c1 := createTestObject(t, objStore, ctx, "contacts")
	c2 := createTestObject(t, objStore, ctx, "contacts")
	co1 := createTestObject(t, objStore, ctx, "companies")
	co2 := createTestObject(t, objStore, ctx, "companies")

	inputs := []store.BatchAssocInput{
		{From: store.ObjectID{ID: c1}, To: store.ObjectID{ID: co1}},
		{From: store.ObjectID{ID: c2}, To: store.ObjectID{ID: co2}},
	}

	results, err := assocStore.BatchAssociateDefault(ctx, "contacts", "companies", inputs)
	if err != nil {
		t.Fatalf("batch default: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestAssocBatchRead(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	_, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("associate: %v", err)
	}

	results, err := assocStore.BatchRead(ctx, "contacts", "companies", []store.BatchAssocReadInput{{ID: contactID}})
	if err != nil {
		t.Fatalf("batch read: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].To) != 1 {
		t.Fatalf("expected 1 association, got %d", len(results[0].To))
	}
}

func TestAssocBatchArchive(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	_, err := assocStore.AssociateDefault(ctx, "contacts", contactID, "companies", companyID)
	if err != nil {
		t.Fatalf("associate: %v", err)
	}

	err = assocStore.BatchArchive(ctx, "contacts", "companies", []store.BatchArchiveInput{
		{From: store.ObjectID{ID: contactID}, To: store.ObjectID{ID: companyID}},
	})
	if err != nil {
		t.Fatalf("batch archive: %v", err)
	}

	results, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get after archive: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 after archive, got %d", len(results))
	}
}

func TestBatchArchiveLabels(t *testing.T) {
	assocStore, objStore, ctx := setupAssocStore(t)

	contactID := createTestObject(t, objStore, ctx, "contacts")
	companyID := createTestObject(t, objStore, ctx, "companies")

	// Associate with Primary label.
	_, err := assocStore.AssociateWithLabels(ctx, "contacts", contactID, "companies", companyID, []store.AssociationInput{
		{AssociationCategory: "HUBSPOT_DEFINED", AssociationTypeID: 279},
	})
	if err != nil {
		t.Fatalf("associate: %v", err)
	}

	// Archive only the Primary label, not the default.
	err = assocStore.BatchArchiveLabels(ctx, "contacts", "companies", []store.BatchArchiveLabelInput{
		{
			From:  store.ObjectID{ID: contactID},
			To:    store.ObjectID{ID: companyID},
			Types: []store.AssociationInput{{AssociationCategory: "HUBSPOT_DEFINED", AssociationTypeID: 279}},
		},
	})
	if err != nil {
		t.Fatalf("batch archive labels: %v", err)
	}

	// Should still have the default association.
	results, err := assocStore.GetAssociations(ctx, "contacts", contactID, "companies")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (default still present), got %d", len(results))
	}
	// Should have only the default type, not the Primary.
	for _, typ := range results[0].Types {
		if typ.TypeID == 279 {
			t.Error("Primary label should have been archived")
		}
	}
}

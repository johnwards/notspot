package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

// A schema's own `properties` must be persisted as property DEFINITIONS (not just stored later as
// EAV values), so they appear in the properties API and the UI. Regression test: without this the
// values exist but render blank because there is no definition for them.
func TestSchemaStore_CreatePersistsCustomProperties(t *testing.T) {
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	schemas := store.NewSQLiteSchemaStore(db)
	props := store.NewSQLitePropertyStore(db)

	_, err := schemas.Create(ctx, &domain.ObjectSchema{
		Name:                   "outreach_email",
		Labels:                 domain.SchemaLabels{Singular: "Outreach Email", Plural: "Outreach Emails"},
		PrimaryDisplayProperty: "subject",
		Properties: []domain.Property{
			{Name: "subject", Label: "Subject", Type: "string", FieldType: "text"},
			{Name: "body", Label: "Body", Type: "string", FieldType: "html"},
			{Name: "status", Label: "Status", Type: "enumeration", FieldType: "select", Options: []domain.Option{
				{Label: "Draft", Value: "draft"}, {Label: "Sent", Value: "sent"},
			}},
		},
	})
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	for _, name := range []string{"subject", "body", "status"} {
		got, err := props.Get(ctx, "outreach_email", name)
		if err != nil {
			t.Fatalf("property %q was not persisted as a definition: %v", name, err)
		}
		if got.Label == "" {
			t.Errorf("property %q has empty label", name)
		}
	}

	status, err := props.Get(ctx, "outreach_email", "status")
	if err != nil {
		t.Fatalf("get status property: %v", err)
	}
	if len(status.Options) != 2 {
		t.Errorf("status options = %d, want 2 (enumeration options must round-trip)", len(status.Options))
	}
}

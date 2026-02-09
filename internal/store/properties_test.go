package store_test

import (
	"context"
	"testing"

	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupPropertyStore(t *testing.T) (store.PropertyStore, context.Context) {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed a test object type.
	_, err := db.ExecContext(ctx,
		`INSERT INTO object_types (id, name, label_singular, label_plural, primary_display_property, is_custom, created_at, updated_at)
		 VALUES ('0-1', 'contacts', 'Contact', 'Contacts', 'email', FALSE, '2024-01-01T00:00:00.000Z', '2024-01-01T00:00:00.000Z')`)
	if err != nil {
		t.Fatalf("seed object type: %v", err)
	}

	// Seed a default property group.
	_, err = db.ExecContext(ctx,
		`INSERT INTO property_groups (object_type_id, name, label, display_order, archived)
		 VALUES ('0-1', 'contactinformation', 'Contact Information', 0, FALSE)`)
	if err != nil {
		t.Fatalf("seed property group: %v", err)
	}

	return store.NewSQLitePropertyStore(db), ctx
}

func TestPropertyStore_CreateAndGet(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	p := &domain.Property{
		Name:      "email",
		Label:     "Email",
		Type:      "string",
		FieldType: "text",
		GroupName: "contactinformation",
	}

	created, err := s.Create(ctx, "contacts", p)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Name != "email" {
		t.Errorf("Name = %q, want %q", created.Name, "email")
	}
	if created.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}

	got, err := s.Get(ctx, "contacts", "email")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "Email" {
		t.Errorf("Label = %q, want %q", got.Label, "Email")
	}
}

func TestPropertyStore_List(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	for _, name := range []string{"email", "firstname", "lastname"} {
		_, err := s.Create(ctx, "contacts", &domain.Property{
			Name:      name,
			Label:     name,
			Type:      "string",
			FieldType: "text",
			GroupName: "contactinformation",
		})
		if err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	props, err := s.List(ctx, "contacts")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(props) != 3 {
		t.Errorf("len(props) = %d, want 3", len(props))
	}
}

func TestPropertyStore_Update(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	_, err := s.Create(ctx, "contacts", &domain.Property{
		Name:      "email",
		Label:     "Email",
		Type:      "string",
		FieldType: "text",
		GroupName: "contactinformation",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated, err := s.Update(ctx, "contacts", "email", &domain.Property{
		Label:     "Email Address",
		FieldType: "text",
		GroupName: "contactinformation",
		Options:   []domain.Option{},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Label != "Email Address" {
		t.Errorf("Label = %q, want %q", updated.Label, "Email Address")
	}
}

func TestPropertyStore_Archive(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	_, err := s.Create(ctx, "contacts", &domain.Property{
		Name:      "email",
		Label:     "Email",
		Type:      "string",
		FieldType: "text",
		GroupName: "contactinformation",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Archive(ctx, "contacts", "email"); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	// Archived properties should not appear in list.
	props, err := s.List(ctx, "contacts")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(props) != 0 {
		t.Errorf("len(props) = %d, want 0 (archived)", len(props))
	}
}

func TestPropertyStore_ArchiveNotFound(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	err := s.Archive(ctx, "contacts", "nonexistent")
	if err == nil {
		t.Fatal("expected error for archiving nonexistent property")
	}
}

func TestPropertyStore_BatchCreate(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	props := []domain.Property{
		{Name: "email", Label: "Email", Type: "string", FieldType: "text", GroupName: "contactinformation"},
		{Name: "phone", Label: "Phone", Type: "string", FieldType: "phonenumber", GroupName: "contactinformation"},
	}

	created, err := s.BatchCreate(ctx, "contacts", props)
	if err != nil {
		t.Fatalf("BatchCreate: %v", err)
	}
	if len(created) != 2 {
		t.Errorf("len(created) = %d, want 2", len(created))
	}
}

func TestPropertyStore_BatchRead(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	for _, name := range []string{"email", "phone"} {
		_, err := s.Create(ctx, "contacts", &domain.Property{
			Name: name, Label: name, Type: "string", FieldType: "text", GroupName: "contactinformation",
		})
		if err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	results, err := s.BatchRead(ctx, "contacts", []string{"email", "phone"})
	if err != nil {
		t.Fatalf("BatchRead: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}
}

func TestPropertyStore_BatchArchive(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	for _, name := range []string{"email", "phone"} {
		_, err := s.Create(ctx, "contacts", &domain.Property{
			Name: name, Label: name, Type: "string", FieldType: "text", GroupName: "contactinformation",
		})
		if err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	if err := s.BatchArchive(ctx, "contacts", []string{"email", "phone"}); err != nil {
		t.Fatalf("BatchArchive: %v", err)
	}

	props, err := s.List(ctx, "contacts")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(props) != 0 {
		t.Errorf("len(props) = %d, want 0", len(props))
	}
}

func TestPropertyStore_GetNotFound(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	_, err := s.Get(ctx, "contacts", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent property")
	}
}

func TestPropertyStore_ResolveByID(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	_, err := s.Create(ctx, "0-1", &domain.Property{
		Name: "email", Label: "Email", Type: "string", FieldType: "text", GroupName: "contactinformation",
	})
	if err != nil {
		t.Fatalf("Create using type ID: %v", err)
	}

	got, err := s.Get(ctx, "0-1", "email")
	if err != nil {
		t.Fatalf("Get using type ID: %v", err)
	}
	if got.Name != "email" {
		t.Errorf("Name = %q, want %q", got.Name, "email")
	}
}

func TestPropertyStore_Options(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	opts := []domain.Option{
		{Label: "Lead", Value: "lead", DisplayOrder: 0},
		{Label: "Customer", Value: "customer", DisplayOrder: 1},
	}

	_, err := s.Create(ctx, "contacts", &domain.Property{
		Name:      "lifecyclestage",
		Label:     "Lifecycle Stage",
		Type:      "enumeration",
		FieldType: "radio",
		GroupName: "contactinformation",
		Options:   opts,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, "contacts", "lifecyclestage")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Options) != 2 {
		t.Fatalf("len(Options) = %d, want 2", len(got.Options))
	}
	if got.Options[0].Value != "lead" {
		t.Errorf("Options[0].Value = %q, want %q", got.Options[0].Value, "lead")
	}
}

// Property group tests

func TestPropertyStore_CreateAndGetGroup(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	g := &domain.PropertyGroup{
		Name:  "customgroup",
		Label: "Custom Group",
	}

	created, err := s.CreateGroup(ctx, "contacts", g)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if created.Name != "customgroup" {
		t.Errorf("Name = %q, want %q", created.Name, "customgroup")
	}

	got, err := s.GetGroup(ctx, "contacts", "customgroup")
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if got.Label != "Custom Group" {
		t.Errorf("Label = %q, want %q", got.Label, "Custom Group")
	}
}

func TestPropertyStore_ListGroups(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	// Already have "contactinformation" seeded.
	_, err := s.CreateGroup(ctx, "contacts", &domain.PropertyGroup{
		Name: "customgroup", Label: "Custom Group",
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	groups, err := s.ListGroups(ctx, "contacts")
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("len(groups) = %d, want 2", len(groups))
	}
}

func TestPropertyStore_UpdateGroup(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	updated, err := s.UpdateGroup(ctx, "contacts", "contactinformation", &domain.PropertyGroup{
		Label:        "Updated Contact Info",
		DisplayOrder: 5,
	})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if updated.Label != "Updated Contact Info" {
		t.Errorf("Label = %q, want %q", updated.Label, "Updated Contact Info")
	}
	if updated.DisplayOrder != 5 {
		t.Errorf("DisplayOrder = %d, want 5", updated.DisplayOrder)
	}
}

func TestPropertyStore_ArchiveGroup(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	if err := s.ArchiveGroup(ctx, "contacts", "contactinformation"); err != nil {
		t.Fatalf("ArchiveGroup: %v", err)
	}

	groups, err := s.ListGroups(ctx, "contacts")
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("len(groups) = %d, want 0", len(groups))
	}
}

func TestPropertyStore_ArchiveGroupNotFound(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	err := s.ArchiveGroup(ctx, "contacts", "nonexistent")
	if err == nil {
		t.Fatal("expected error for archiving nonexistent group")
	}
}

func TestPropertyStore_GetGroupNotFound(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	_, err := s.GetGroup(ctx, "contacts", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent group")
	}
}

func TestPropertyStore_InvalidObjectType(t *testing.T) {
	s, ctx := setupPropertyStore(t)

	_, err := s.List(ctx, "invalid_type")
	if err == nil {
		t.Fatal("expected error for invalid object type")
	}
}

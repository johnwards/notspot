package seed

import (
	"context"
	"database/sql"
	"fmt"
)

// AssociationTypeDef defines a standard association type to seed.
type AssociationTypeDef struct {
	ID             int
	FromObjectType string // object_types.id
	ToObjectType   string // object_types.id
	Category       string
	Label          string // empty for unlabeled/default
}

// StandardAssociationTypes are the HubSpot-defined association types seeded at startup.
var StandardAssociationTypes = []AssociationTypeDef{
	// Contact ↔ Company (1/2 = default unlabeled, 279/280 = Primary)
	{ID: 1, FromObjectType: "0-1", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 2, FromObjectType: "0-2", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 279, FromObjectType: "0-1", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: "Primary"},
	{ID: 280, FromObjectType: "0-2", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: "Primary"},

	// Contact ↔ Deal
	{ID: 3, FromObjectType: "0-1", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 4, FromObjectType: "0-3", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},

	// Company ↔ Deal
	{ID: 5, FromObjectType: "0-2", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 6, FromObjectType: "0-3", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},

	// Contact ↔ Ticket
	{ID: 15, FromObjectType: "0-1", ToObjectType: "0-5", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 16, FromObjectType: "0-5", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},

	// Deal ↔ Line Item
	{ID: 19, FromObjectType: "0-3", ToObjectType: "0-8", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 20, FromObjectType: "0-8", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},

	// Company ↔ Ticket
	{ID: 25, FromObjectType: "0-2", ToObjectType: "0-5", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 26, FromObjectType: "0-5", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},

	// Note (0-46) ↔ Contact/Company/Deal
	{ID: 202, FromObjectType: "0-46", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 203, FromObjectType: "0-1", ToObjectType: "0-46", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 204, FromObjectType: "0-46", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 205, FromObjectType: "0-2", ToObjectType: "0-46", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 206, FromObjectType: "0-46", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 207, FromObjectType: "0-3", ToObjectType: "0-46", Category: "HUBSPOT_DEFINED", Label: ""},

	// Call (0-48) ↔ Contact/Company/Deal
	{ID: 208, FromObjectType: "0-48", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 209, FromObjectType: "0-1", ToObjectType: "0-48", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 210, FromObjectType: "0-48", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 211, FromObjectType: "0-2", ToObjectType: "0-48", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 212, FromObjectType: "0-48", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 213, FromObjectType: "0-3", ToObjectType: "0-48", Category: "HUBSPOT_DEFINED", Label: ""},

	// Email (0-49) ↔ Contact/Company/Deal
	{ID: 214, FromObjectType: "0-49", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 215, FromObjectType: "0-1", ToObjectType: "0-49", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 216, FromObjectType: "0-49", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 217, FromObjectType: "0-2", ToObjectType: "0-49", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 218, FromObjectType: "0-49", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 219, FromObjectType: "0-3", ToObjectType: "0-49", Category: "HUBSPOT_DEFINED", Label: ""},

	// Task (0-27) ↔ Contact/Company/Deal
	{ID: 220, FromObjectType: "0-27", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 221, FromObjectType: "0-1", ToObjectType: "0-27", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 222, FromObjectType: "0-27", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 223, FromObjectType: "0-2", ToObjectType: "0-27", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 224, FromObjectType: "0-27", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 225, FromObjectType: "0-3", ToObjectType: "0-27", Category: "HUBSPOT_DEFINED", Label: ""},

	// Meeting (0-47) ↔ Contact/Company/Deal
	{ID: 226, FromObjectType: "0-47", ToObjectType: "0-1", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 227, FromObjectType: "0-1", ToObjectType: "0-47", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 228, FromObjectType: "0-47", ToObjectType: "0-2", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 229, FromObjectType: "0-2", ToObjectType: "0-47", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 230, FromObjectType: "0-47", ToObjectType: "0-3", Category: "HUBSPOT_DEFINED", Label: ""},
	{ID: 231, FromObjectType: "0-3", ToObjectType: "0-47", Category: "HUBSPOT_DEFINED", Label: ""},
}

// AssociationTypes inserts all standard association types. Idempotent.
func AssociationTypes(ctx context.Context, db *sql.DB) error {
	for _, def := range StandardAssociationTypes {
		_, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO association_types (id, from_object_type, to_object_type, category, label)
			 VALUES (?, ?, ?, ?, NULLIF(?, ''))`,
			def.ID, def.FromObjectType, def.ToObjectType, def.Category, def.Label,
		)
		if err != nil {
			return fmt.Errorf("seed association type %d: %w", def.ID, err)
		}
	}
	return nil
}

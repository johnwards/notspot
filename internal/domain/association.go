package domain

// AssociationType defines a type of association between two object types.
type AssociationType struct {
	TypeID   int    `json:"typeId"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

// AssociationResult represents a single association from a source object to a target.
type AssociationResult struct {
	ToObjectID string            `json:"toObjectId"`
	Types      []AssociationType `json:"associationTypes"`
}

// AssociationLabel represents a label definition for display/management.
type AssociationLabel struct {
	Category string `json:"category"`
	TypeID   int    `json:"typeId"`
	Label    string `json:"label"`
}

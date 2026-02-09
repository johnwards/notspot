package domain

// ObjectSchema represents a HubSpot custom object schema definition.
type ObjectSchema struct {
	ID                     string              `json:"id"`
	Name                   string              `json:"name"`
	Labels                 SchemaLabels        `json:"labels"`
	PrimaryDisplayProperty string              `json:"primaryDisplayProperty"`
	Properties             []Property          `json:"properties"`
	Associations           []SchemaAssociation `json:"associations"`
	AssociatedObjects      []string            `json:"associatedObjects,omitempty"`
	FullyQualifiedName     string              `json:"fullyQualifiedName"`
	Archived               bool                `json:"archived"`
	CreatedAt              string              `json:"createdAt"`
	UpdatedAt              string              `json:"updatedAt"`
}

// SchemaLabels holds the singular and plural display labels for a schema.
type SchemaLabels struct {
	Singular string `json:"singular"`
	Plural   string `json:"plural"`
}

// SchemaAssociation defines an association type between schemas.
type SchemaAssociation struct {
	ID               string       `json:"id"`
	FromObjectTypeID string       `json:"fromObjectTypeId"`
	ToObjectTypeID   string       `json:"toObjectTypeId"`
	Name             string       `json:"name,omitempty"`
	Labels           SchemaLabels `json:"labels,omitempty"`
	CreatedAt        string       `json:"createdAt,omitempty"`
	UpdatedAt        string       `json:"updatedAt,omitempty"`
}

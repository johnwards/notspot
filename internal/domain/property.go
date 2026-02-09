package domain

// Property represents a HubSpot CRM property definition.
type Property struct {
	Name                 string                `json:"name"`
	Label                string                `json:"label"`
	Type                 string                `json:"type"`
	FieldType            string                `json:"fieldType"`
	GroupName            string                `json:"groupName"`
	Description          string                `json:"description"`
	Options              []Option              `json:"options"`
	DisplayOrder         int                   `json:"displayOrder"`
	HasUniqueValue       bool                  `json:"hasUniqueValue"`
	Hidden               bool                  `json:"hidden"`
	FormField            bool                  `json:"formField"`
	Calculated           bool                  `json:"calculated"`
	ExternalOptions      bool                  `json:"externalOptions"`
	Archived             bool                  `json:"archived"`
	HubspotDefined       bool                  `json:"hubspotDefined"`
	CreatedAt            string                `json:"createdAt,omitempty"`
	UpdatedAt            string                `json:"updatedAt,omitempty"`
	ArchivedAt           string                `json:"archivedAt,omitempty"`
	ModificationMetadata *ModificationMetadata `json:"modificationMetadata,omitempty"`
}

// Option represents a selectable option for enumeration properties.
type Option struct {
	Label        string `json:"label"`
	Value        string `json:"value"`
	DisplayOrder int    `json:"displayOrder"`
	Hidden       bool   `json:"hidden"`
}

// ModificationMetadata describes the modification constraints of a property.
type ModificationMetadata struct {
	Archivable         bool `json:"archivable"`
	ReadOnlyDefinition bool `json:"readOnlyDefinition"`
	ReadOnlyOptions    bool `json:"readOnlyOptions"`
	ReadOnlyValue      bool `json:"readOnlyValue"`
}

// PropertyGroup represents a grouping of related properties.
type PropertyGroup struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	DisplayOrder int    `json:"displayOrder"`
	Archived     bool   `json:"archived"`
}

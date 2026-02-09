package domain

// Pipeline represents a HubSpot CRM pipeline (e.g. deals "Sales Pipeline").
type Pipeline struct {
	ID           string          `json:"id"`
	Label        string          `json:"label"`
	DisplayOrder int             `json:"displayOrder"`
	Stages       []PipelineStage `json:"stages"`
	Archived     bool            `json:"archived"`
	CreatedAt    string          `json:"createdAt"`
	UpdatedAt    string          `json:"updatedAt"`
}

// PipelineStage represents a single stage within a pipeline.
type PipelineStage struct {
	ID           string            `json:"id"`
	Label        string            `json:"label"`
	DisplayOrder int               `json:"displayOrder"`
	Metadata     map[string]string `json:"metadata"`
	Archived     bool              `json:"archived"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

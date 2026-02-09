package domain

// Object represents a CRM object (contact, company, deal, etc.).
type Object struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
	CreatedAt  string            `json:"createdAt"`
	UpdatedAt  string            `json:"updatedAt"`
	Archived   bool              `json:"archived"`
	ArchivedAt string            `json:"archivedAt,omitempty"`
}

// CreateInput holds the data needed to create a new object.
type CreateInput struct {
	Properties map[string]string `json:"properties"`
}

// UpdateInput holds the data needed to update an existing object.
type UpdateInput struct {
	ID         string            `json:"id"`
	Properties map[string]string `json:"properties"`
}

// UpsertInput holds the data for an upsert operation.
type UpsertInput struct {
	ID         string            `json:"id,omitempty"`
	IDProperty string            `json:"idProperty,omitempty"`
	Properties map[string]string `json:"properties"`
}

// ListOpts holds the parameters for listing objects.
type ListOpts struct {
	Limit      int
	After      string
	Properties []string
	Archived   bool
}

// ObjectPage is a paginated list of objects.
type ObjectPage struct {
	Results []*Object
	After   string
	HasMore bool
}

// BatchResult wraps the result of a batch operation.
type BatchResult struct {
	Status      string    `json:"status"`
	Results     []*Object `json:"results"`
	StartedAt   string    `json:"startedAt"`
	CompletedAt string    `json:"completedAt"`
	NumErrors   int       `json:"numErrors,omitempty"`
}

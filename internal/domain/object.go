package domain

import "encoding/json"

// Properties is a HubSpot-style property bag. HubSpot accepts property values as JSON strings,
// numbers, or booleans on writes and coerces them all to strings; a strict map[string]string would
// reject a numeric value (e.g. an integer step_number or icp_score) at JSON-decode time with a 400.
// UnmarshalJSON mirrors HubSpot by stringifying any JSON scalar, so clients that legitimately send
// numbers/booleans (as the real API allows) interoperate. Underlying type stays map[string]string,
// so it is assignable to/from a plain map[string]string throughout the store and validators.
type Properties map[string]string

// UnmarshalJSON decodes each value as a raw token and stringifies scalars: a JSON string is
// unquoted, null becomes "", and numbers/booleans keep their literal text (e.g. 7 -> "7").
func (p *Properties) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	m := make(map[string]string, len(raw))
	for k, v := range raw {
		s := string(v)
		switch {
		case s == "null":
			m[k] = ""
		case s != "" && s[0] == '"':
			var str string
			if err := json.Unmarshal(v, &str); err != nil {
				return err
			}
			m[k] = str
		default: // number, boolean -> literal JSON text (HubSpot stores these as strings)
			m[k] = s
		}
	}
	*p = m
	return nil
}

// HistoryValue is a single point-in-time property value, mirroring HubSpot's
// ValueWithTimestamp schema. The required trio is value + timestamp + sourceType;
// sourceId and sourceLabel are optional and omitted when empty.
type HistoryValue struct {
	Value       string `json:"value"`
	Timestamp   string `json:"timestamp"`
	SourceType  string `json:"sourceType"`
	SourceID    string `json:"sourceId,omitempty"`
	SourceLabel string `json:"sourceLabel,omitempty"`
}

// Object represents a CRM object (contact, company, deal, etc.).
type Object struct {
	ID                    string                    `json:"id"`
	Properties            map[string]string         `json:"properties"`
	PropertiesWithHistory map[string][]HistoryValue `json:"propertiesWithHistory,omitempty"`
	CreatedAt             string                    `json:"createdAt"`
	UpdatedAt             string                    `json:"updatedAt"`
	Archived              bool                      `json:"archived"`
	ArchivedAt            string                    `json:"archivedAt,omitempty"`
}

// CreateInput holds the data needed to create a new object.
type CreateInput struct {
	Properties Properties `json:"properties"`
}

// UpdateInput holds the data needed to update an existing object.
type UpdateInput struct {
	ID         string     `json:"id"`
	Properties Properties `json:"properties"`
}

// UpsertInput holds the data for an upsert operation.
type UpsertInput struct {
	ID         string     `json:"id,omitempty"`
	IDProperty string     `json:"idProperty,omitempty"`
	Properties Properties `json:"properties"`
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

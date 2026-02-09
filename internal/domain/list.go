package domain

import "encoding/json"

// List represents a HubSpot list.
type List struct {
	ListID           string          `json:"listId"`
	Name             string          `json:"name"`
	ObjectTypeId     string          `json:"objectTypeId"`
	ProcessingType   string          `json:"processingType"`
	ProcessingStatus string          `json:"processingStatus"`
	FilterBranch     json.RawMessage `json:"filterBranch,omitempty"`
	ListVersion      int             `json:"listVersion"`
	Size             int             `json:"size"`
	CreatedAt        string          `json:"createdAt"`
	UpdatedAt        string          `json:"updatedAt"`
}

// ListMembership represents a single membership record.
type ListMembership struct {
	RecordID string `json:"recordId"`
	ListID   string `json:"listId"`
	AddedAt  string `json:"addedAt"`
}

// MembershipPage is a paginated list of memberships.
type MembershipPage struct {
	Results []*ListMembership
	After   string
	HasMore bool
}

// MembershipUpdateResponse is the response for add/remove membership operations.
// Note: HubSpot has a typo in the spec — "recordsIdsAdded" (extra 's').
type MembershipUpdateResponse struct {
	RecordIdsAdded    []string `json:"recordIdsAdded"`
	RecordsIdsAdded   []string `json:"recordsIdsAdded"`
	RecordIdsMissing  []string `json:"recordIdsMissing"`
	RecordIdsRemoved  []string `json:"recordIdsRemoved"`
	RecordsIdsRemoved []string `json:"recordsIdsRemoved"`
}

// ListSearchOpts holds offset-based pagination for list search.
type ListSearchOpts struct {
	Query  string
	Offset int
	Limit  int
}

// ListSearchPage is an offset-paginated list of lists.
type ListSearchPage struct {
	Results    []*List
	Offset     int
	HasMore    bool
	TotalCount int
}

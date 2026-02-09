package domain

// SearchRequest represents a CRM search API request.
type SearchRequest struct {
	Query        string        `json:"query"`
	FilterGroups []FilterGroup `json:"filterGroups"`
	Sorts        []Sort        `json:"sorts"`
	Properties   []string      `json:"properties"`
	Limit        int           `json:"limit"`
	After        string        `json:"after"`
}

// FilterGroup is a group of filters combined with AND.
type FilterGroup struct {
	Filters []Filter `json:"filters"`
}

// Filter represents a single property filter.
type Filter struct {
	PropertyName string   `json:"propertyName"`
	Operator     string   `json:"operator"`
	Value        string   `json:"value,omitempty"`
	HighValue    string   `json:"highValue,omitempty"`
	Values       []string `json:"values,omitempty"`
}

// Sort specifies a sort order for search results.
type Sort struct {
	PropertyName string `json:"propertyName"`
	Direction    string `json:"direction"`
}

// SearchResult is the response from a CRM search.
type SearchResult struct {
	Total   int           `json:"total"`
	Results []*Object     `json:"results"`
	Paging  *SearchPaging `json:"paging,omitempty"`
}

// SearchPaging holds pagination info for search results.
type SearchPaging struct {
	Next SearchPagingNext `json:"next"`
}

// SearchPagingNext holds the cursor for the next page of search results.
type SearchPagingNext struct {
	After string `json:"after"`
}

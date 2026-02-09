package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON marshals v as JSON and writes it to w with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "error", err)
	}
}

// Paging represents cursor-based pagination info in HubSpot responses.
type Paging struct {
	Next *PagingNext `json:"next,omitempty"`
}

// PagingNext holds the cursor for the next page.
type PagingNext struct {
	After string `json:"after"`
	Link  string `json:"link,omitempty"`
}

// CollectionResponse is a generic paginated list response.
type CollectionResponse struct {
	Results []any   `json:"results"`
	Paging  *Paging `json:"paging,omitempty"`
}

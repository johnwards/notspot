package lists

import (
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes adds all list endpoints to the given mux.
func RegisterRoutes(mux *http.ServeMux, s *store.Store) {
	h := &Handler{store: s}

	mux.HandleFunc("POST /crm/v3/lists", h.Create)
	mux.HandleFunc("POST /crm/v3/lists/search", h.Search)
	mux.HandleFunc("GET /crm/v3/lists/", h.GetMultiple)
	mux.HandleFunc("GET /crm/v3/lists/{listId}", h.Get)
	mux.HandleFunc("DELETE /crm/v3/lists/{listId}", h.Delete)
	mux.HandleFunc("PUT /crm/v3/lists/{listId}/restore", h.Restore)
	mux.HandleFunc("PUT /crm/v3/lists/{listId}/update-list-name", h.UpdateName)
	mux.HandleFunc("PUT /crm/v3/lists/{listId}/update-list-filters", h.UpdateFilters)
	mux.HandleFunc("GET /crm/v3/lists/{listId}/memberships", h.GetMemberships)
	mux.HandleFunc("PUT /crm/v3/lists/{listId}/memberships/add", h.AddMembers)
	mux.HandleFunc("PUT /crm/v3/lists/{listId}/memberships/remove", h.RemoveMembers)
	mux.HandleFunc("PUT /crm/v3/lists/{listId}/memberships/add-and-remove", h.AddAndRemoveMembers)
	mux.HandleFunc("DELETE /crm/v3/lists/{listId}/memberships", h.RemoveAllMembers)
}

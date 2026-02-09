package owners

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler handles owner HTTP requests.
type Handler struct {
	store *store.Store
}

// List handles GET /crm/v3/owners.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	after := r.URL.Query().Get("after")
	email := r.URL.Query().Get("email")
	archived := r.URL.Query().Get("archived") == "true"

	owners, hasMore, nextAfter, err := h.store.Owners.List(r.Context(), limit, after, email, archived)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(owners))
	for i, o := range owners {
		results[i] = o
	}

	resp := api.CollectionResponse{Results: results}
	if hasMore {
		resp.Paging = &api.Paging{
			Next: &api.PagingNext{After: nextAfter},
		}
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// Get handles GET /crm/v3/owners/{ownerId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ownerID := r.PathValue("ownerId")
	corrID := api.CorrelationID(r.Context())

	owner, err := h.store.Owners.Get(r.Context(), ownerID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Owner not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, owner)
}

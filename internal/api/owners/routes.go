package owners

import (
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes adds all owner endpoints to the given mux.
func RegisterRoutes(mux *http.ServeMux, s *store.Store) {
	h := &Handler{store: s}

	mux.HandleFunc("GET /crm/v3/owners", h.List)
	mux.HandleFunc("GET /crm/v3/owners/", h.List)
	mux.HandleFunc("GET /crm/v3/owners/{ownerId}", h.Get)
}

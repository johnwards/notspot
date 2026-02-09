package imports

import (
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes adds all import endpoints to the given mux.
func RegisterRoutes(mux *http.ServeMux, s *store.Store) {
	h := &Handler{store: s}

	mux.HandleFunc("POST /crm/v3/imports", h.Start)
	mux.HandleFunc("POST /crm/v3/imports/", h.Start)
	mux.HandleFunc("GET /crm/v3/imports", h.List)
	mux.HandleFunc("GET /crm/v3/imports/", h.List)
	mux.HandleFunc("GET /crm/v3/imports/{importId}", h.Get)
	mux.HandleFunc("POST /crm/v3/imports/{importId}/cancel", h.Cancel)
	mux.HandleFunc("GET /crm/v3/imports/{importId}/errors", h.GetErrors)
}

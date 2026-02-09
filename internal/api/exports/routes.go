package exports

import (
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes adds all export endpoints to the given mux.
func RegisterRoutes(mux *http.ServeMux, s *store.Store) {
	h := &Handler{store: s}

	mux.HandleFunc("POST /crm/v3/exports/export/async", h.Start)
	mux.HandleFunc("GET /crm/v3/exports/export/async/tasks/{taskId}/status", h.GetStatus)
}

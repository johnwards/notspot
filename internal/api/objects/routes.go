package objects

import (
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes adds all CRM object endpoints to the given mux.
func RegisterRoutes(mux *http.ServeMux, s *store.Store) {
	h := &Handler{store: s}

	mux.HandleFunc("GET /crm/v3/objects/{objectType}", h.List)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}", h.Create)
	mux.HandleFunc("GET /crm/v3/objects/{objectType}/{objectId}", h.Get)
	mux.HandleFunc("PATCH /crm/v3/objects/{objectType}/{objectId}", h.Update)
	mux.HandleFunc("DELETE /crm/v3/objects/{objectType}/{objectId}", h.Archive)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/search", h.Search)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/batch/create", h.BatchCreate)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/batch/read", h.BatchRead)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/batch/update", h.BatchUpdate)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/batch/upsert", h.BatchUpsert)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/batch/archive", h.BatchArchive)
	mux.HandleFunc("POST /crm/v3/objects/{objectType}/merge", h.Merge)
}

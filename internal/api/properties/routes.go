package properties

import (
	"database/sql"
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes registers all property and property group endpoints on the mux.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	h := &Handler{store: store.NewSQLitePropertyStore(db)}

	// Property CRUD
	mux.HandleFunc("GET /crm/v3/properties/{objectType}", h.List)
	mux.HandleFunc("POST /crm/v3/properties/{objectType}", h.Create)
	mux.HandleFunc("GET /crm/v3/properties/{objectType}/{propertyName}", h.Get)
	mux.HandleFunc("PATCH /crm/v3/properties/{objectType}/{propertyName}", h.Update)
	mux.HandleFunc("DELETE /crm/v3/properties/{objectType}/{propertyName}", h.Archive)

	// Batch operations
	mux.HandleFunc("POST /crm/v3/properties/{objectType}/batch/create", h.BatchCreate)
	mux.HandleFunc("POST /crm/v3/properties/{objectType}/batch/read", h.BatchRead)
	mux.HandleFunc("POST /crm/v3/properties/{objectType}/batch/archive", h.BatchArchive)

	// Property groups
	mux.HandleFunc("GET /crm/v3/properties/{objectType}/groups", h.ListGroups)
	mux.HandleFunc("POST /crm/v3/properties/{objectType}/groups", h.CreateGroup)
	mux.HandleFunc("GET /crm/v3/properties/{objectType}/groups/{groupName}", h.GetGroup)
	mux.HandleFunc("PATCH /crm/v3/properties/{objectType}/groups/{groupName}", h.UpdateGroup)
	mux.HandleFunc("DELETE /crm/v3/properties/{objectType}/groups/{groupName}", h.ArchiveGroup)
}

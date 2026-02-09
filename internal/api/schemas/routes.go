package schemas

import (
	"database/sql"
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes registers all custom object schema endpoints on the mux.
// Both /crm/v3/schemas and /crm-object-schemas/v3/schemas paths are supported.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	h := &Handler{store: store.NewSQLiteSchemaStore(db)}

	for _, prefix := range []string{"/crm/v3/schemas", "/crm-object-schemas/v3/schemas"} {
		mux.HandleFunc("GET "+prefix, h.List)
		mux.HandleFunc("POST "+prefix, h.Create)
		mux.HandleFunc("GET "+prefix+"/{objectType}", h.Get)
		mux.HandleFunc("PATCH "+prefix+"/{objectType}", h.Update)
		mux.HandleFunc("DELETE "+prefix+"/{objectType}", h.Archive)
		mux.HandleFunc("POST "+prefix+"/{objectType}/associations", h.CreateAssociation)
		mux.HandleFunc("DELETE "+prefix+"/{objectType}/associations/{associationId}", h.DeleteAssociation)
	}
}

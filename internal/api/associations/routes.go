package associations

import (
	"database/sql"
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes registers all association v4 endpoints on the mux.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	h := &Handler{store: store.NewSQLiteAssociationStore(db)}

	// Record-level association endpoints.
	mux.HandleFunc("PUT /crm/v4/objects/{from}/{fromId}/associations/default/{to}/{toId}", h.AssociateDefault)
	mux.HandleFunc("PUT /crm/v4/objects/{from}/{fromId}/associations/{to}/{toId}", h.AssociateWithLabels)
	mux.HandleFunc("GET /crm/v4/objects/{from}/{fromId}/associations/{to}", h.GetAssociations)
	mux.HandleFunc("DELETE /crm/v4/objects/{from}/{fromId}/associations/{to}/{toId}", h.RemoveAssociations)

	// Batch endpoints.
	mux.HandleFunc("POST /crm/v4/associations/{from}/{to}/batch/associate/default", h.BatchAssociateDefault)
	mux.HandleFunc("POST /crm/v4/associations/{from}/{to}/batch/create", h.BatchCreate)
	mux.HandleFunc("POST /crm/v4/associations/{from}/{to}/batch/read", h.BatchRead)
	mux.HandleFunc("POST /crm/v4/associations/{from}/{to}/batch/archive", h.BatchArchive)
	mux.HandleFunc("POST /crm/v4/associations/{from}/{to}/batch/labels/archive", h.BatchArchiveLabels)

	// Label management endpoints.
	mux.HandleFunc("GET /crm/v4/associations/{from}/{to}/labels", h.ListLabels)
	mux.HandleFunc("POST /crm/v4/associations/{from}/{to}/labels", h.CreateLabel)
	mux.HandleFunc("PUT /crm/v4/associations/{from}/{to}/labels", h.UpdateLabel)
	mux.HandleFunc("DELETE /crm/v4/associations/{from}/{to}/labels/{typeId}", h.DeleteLabel)
}

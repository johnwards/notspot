package pipelines

import (
	"database/sql"
	"net/http"

	"github.com/johnwards/hubspot/internal/store"
)

// RegisterRoutes registers all pipeline and pipeline stage routes on the mux.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	h := &Handler{store: store.NewSQLitePipelineStore(db)}

	mux.HandleFunc("GET /crm/v3/pipelines/{objectType}", h.List)
	mux.HandleFunc("POST /crm/v3/pipelines/{objectType}", h.Create)
	mux.HandleFunc("GET /crm/v3/pipelines/{objectType}/{pipelineId}", h.Get)
	mux.HandleFunc("PATCH /crm/v3/pipelines/{objectType}/{pipelineId}", h.Update)
	mux.HandleFunc("PUT /crm/v3/pipelines/{objectType}/{pipelineId}", h.Replace)
	mux.HandleFunc("DELETE /crm/v3/pipelines/{objectType}/{pipelineId}", h.Delete)
	mux.HandleFunc("GET /crm/v3/pipelines/{objectType}/{pipelineId}/stages", h.ListStages)
	mux.HandleFunc("POST /crm/v3/pipelines/{objectType}/{pipelineId}/stages", h.CreateStage)
	mux.HandleFunc("GET /crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}", h.GetStage)
	mux.HandleFunc("PATCH /crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}", h.UpdateStage)
	mux.HandleFunc("PUT /crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}", h.ReplaceStage)
	mux.HandleFunc("DELETE /crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}", h.DeleteStage)
}

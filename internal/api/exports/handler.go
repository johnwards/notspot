package exports

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler handles export HTTP requests.
type Handler struct {
	store *store.Store
}

// exportRequest is the JSON body for starting an export.
type exportRequest struct {
	ExportType       string   `json:"exportType"`
	ExportName       string   `json:"exportName"`
	ObjectType       string   `json:"objectType"`
	ObjectProperties []string `json:"objectProperties"`
}

// statusResponse represents the export status API response.
type statusResponse struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	Result    *exportResult `json:"result,omitempty"`
	CreatedAt string        `json:"createdAt"`
	UpdatedAt string        `json:"updatedAt"`
}

type exportResult struct {
	RecordCount int    `json:"recordCount"`
	DownloadURL string `json:"downloadUrl,omitempty"`
}

// Start handles POST /crm/v3/exports/export/async.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if req.ObjectType == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("objectType is required", corrID, nil))
		return
	}
	if len(req.ObjectProperties) == 0 {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("objectProperties is required", corrID, nil))
		return
	}

	// Validate object type exists.
	var typeExists int
	err := h.store.DB.QueryRowContext(r.Context(),
		`SELECT 1 FROM object_types WHERE name = ? OR id = ?`,
		req.ObjectType, req.ObjectType,
	).Scan(&typeExists)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			fmt.Sprintf("Object type %q not found", req.ObjectType), corrID, nil))
		return
	}

	exportType := req.ExportType
	if exportType == "" {
		exportType = "VIEW"
	}

	reqJSON, _ := json.Marshal(req)

	// Create the export record.
	exp, err := h.store.Exports.Create(r.Context(), req.ExportName, exportType, req.ObjectType, req.ObjectProperties, reqJSON)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	// Query objects and generate CSV immediately (mock server — synchronous).
	page, err := h.store.Objects.List(r.Context(), req.ObjectType, domain.ListOpts{
		Limit:      10000,
		Properties: req.ObjectProperties,
	})
	if err != nil {
		// Complete with zero records on error.
		_ = h.store.Exports.Complete(r.Context(), exp.ID, nil, 0)
		result, _ := h.store.Exports.Get(r.Context(), exp.ID)
		if result != nil {
			api.WriteJSON(w, http.StatusAccepted, statusResponse{
				ID:        result.ID,
				Status:    result.State,
				CreatedAt: result.CreatedAt,
				UpdatedAt: result.UpdatedAt,
			})
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	// Generate CSV in memory.
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	// Write header.
	header := append([]string{"hs_object_id"}, req.ObjectProperties...)
	_ = csvWriter.Write(header)

	// Write rows.
	for _, obj := range page.Results {
		row := make([]string, len(header))
		row[0] = obj.ID
		for i, prop := range req.ObjectProperties {
			row[i+1] = obj.Properties[prop]
		}
		_ = csvWriter.Write(row)
	}
	csvWriter.Flush()

	_ = h.store.Exports.Complete(r.Context(), exp.ID, buf.Bytes(), len(page.Results))

	api.WriteJSON(w, http.StatusAccepted, statusResponse{
		ID:        exp.ID,
		Status:    "COMPLETE",
		CreatedAt: exp.CreatedAt,
		UpdatedAt: exp.UpdatedAt,
		Result:    &exportResult{RecordCount: len(page.Results)},
	})
}

// GetStatus handles GET /crm/v3/exports/export/async/tasks/{taskId}/status.
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	corrID := api.CorrelationID(r.Context())

	exp, err := h.store.Exports.Get(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Export task not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	resp := statusResponse{
		ID:        exp.ID,
		Status:    exp.State,
		CreatedAt: exp.CreatedAt,
		UpdatedAt: exp.UpdatedAt,
	}

	if exp.State == "COMPLETE" {
		resp.Result = &exportResult{
			RecordCount: exp.RecordCount,
		}
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

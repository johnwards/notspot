package pipelines

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler handles pipeline and pipeline stage HTTP requests.
type Handler struct {
	store store.PipelineStore
}

// List returns all pipelines for the given object type.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	pipelines, err := h.store.List(r.Context(), objectType)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	results := make([]any, len(pipelines))
	for i := range pipelines {
		results[i] = pipelines[i]
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

// Create adds a new pipeline.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	validPipelineTypes := map[string]bool{"deals": true, "tickets": true, "0-3": true, "0-5": true}
	if !validPipelineTypes[objectType] {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			fmt.Sprintf("Object type %q does not support pipelines", objectType), corrID, nil))
		return
	}

	var p domain.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input", corrID, nil))
		return
	}

	created, err := h.store.Create(r.Context(), objectType, &p)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusCreated, created)
}

// Get returns a single pipeline.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	corrID := api.CorrelationID(r.Context())

	p, err := h.store.Get(r.Context(), objectType, pipelineID)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, p)
}

// Update partially updates a pipeline (PATCH).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	corrID := api.CorrelationID(r.Context())

	var p domain.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input", corrID, nil))
		return
	}

	updated, err := h.store.Update(r.Context(), objectType, pipelineID, &p)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, updated)
}

// Replace fully replaces a pipeline (PUT).
func (h *Handler) Replace(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	corrID := api.CorrelationID(r.Context())

	var p domain.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input", corrID, nil))
		return
	}

	replaced, err := h.store.Replace(r.Context(), objectType, pipelineID, &p)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, replaced)
}

// Delete removes a pipeline.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	corrID := api.CorrelationID(r.Context())

	if err := h.store.Delete(r.Context(), objectType, pipelineID); err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListStages returns all stages for a pipeline.
func (h *Handler) ListStages(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	corrID := api.CorrelationID(r.Context())

	stages, err := h.store.ListStages(r.Context(), objectType, pipelineID)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	results := make([]any, len(stages))
	for i := range stages {
		results[i] = stages[i]
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

// CreateStage adds a new stage to a pipeline.
func (h *Handler) CreateStage(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	corrID := api.CorrelationID(r.Context())

	var s domain.PipelineStage
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input", corrID, nil))
		return
	}

	// Deal pipeline stages require metadata.probability.
	if objectType == "deals" {
		if s.Metadata == nil || s.Metadata["probability"] == "" {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
				"metadata.probability is required for deal pipeline stages", corrID, nil))
			return
		}
	}

	created, err := h.store.CreateStage(r.Context(), objectType, pipelineID, &s)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusCreated, created)
}

// GetStage returns a single stage.
func (h *Handler) GetStage(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	stageID := r.PathValue("stageId")
	corrID := api.CorrelationID(r.Context())

	s, err := h.store.GetStage(r.Context(), objectType, pipelineID, stageID)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, s)
}

// UpdateStage partially updates a stage (PATCH).
func (h *Handler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	stageID := r.PathValue("stageId")
	corrID := api.CorrelationID(r.Context())

	var s domain.PipelineStage
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input", corrID, nil))
		return
	}

	updated, err := h.store.UpdateStage(r.Context(), objectType, pipelineID, stageID, &s)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, updated)
}

// ReplaceStage fully replaces a stage (PUT).
func (h *Handler) ReplaceStage(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	stageID := r.PathValue("stageId")
	corrID := api.CorrelationID(r.Context())

	var s domain.PipelineStage
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input", corrID, nil))
		return
	}

	replaced, err := h.store.ReplaceStage(r.Context(), objectType, pipelineID, stageID, &s)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, replaced)
}

// DeleteStage removes a stage from a pipeline.
func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	pipelineID := r.PathValue("pipelineId")
	stageID := r.PathValue("stageId")
	corrID := api.CorrelationID(r.Context())

	if err := h.store.DeleteStage(r.Context(), objectType, pipelineID, stageID); err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isNotFound checks if an error message indicates a not-found condition.
func isNotFound(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "not found")
}

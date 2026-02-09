package schemas

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler serves the CRM custom object schemas API endpoints.
type Handler struct {
	store store.SchemaStore
}

// List returns all custom object schemas.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	schemas, err := h.store.List(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	results := make([]any, len(schemas))
	for i := range schemas {
		results[i] = schemas[i]
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

// Create adds a new custom object schema.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	var schema domain.ObjectSchema
	if err := json.NewDecoder(r.Body).Decode(&schema); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if schema.Name == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			"Schema name is required", corrID, nil))
		return
	}
	if schema.Labels.Singular == "" || schema.Labels.Plural == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			"Schema labels (singular and plural) are required", corrID, nil))
		return
	}

	created, err := h.store.Create(r.Context(), &schema)
	if err != nil {
		if isDuplicate(err) {
			api.WriteError(w, http.StatusConflict, api.NewConflictError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	api.WriteJSON(w, http.StatusCreated, created)
}

// Get retrieves a single custom object schema.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	schema, err := h.store.Get(r.Context(), objectType)
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

	api.WriteJSON(w, http.StatusOK, schema)
}

// Update partially modifies a custom object schema.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var patch domain.ObjectSchema
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	updated, err := h.store.Update(r.Context(), objectType, &patch)
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

// Archive soft-deletes a custom object schema.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	if err := h.store.Archive(r.Context(), objectType); err != nil {
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

// CreateAssociation adds a new association type to a schema.
func (h *Handler) CreateAssociation(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var a domain.SchemaAssociation
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if a.ToObjectTypeID == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			"toObjectTypeId is required", corrID, nil))
		return
	}

	created, err := h.store.CreateAssociation(r.Context(), objectType, &a)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
			return
		}
		if isDuplicate(err) {
			api.WriteError(w, http.StatusConflict, api.NewConflictError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	api.WriteJSON(w, http.StatusCreated, created)
}

// DeleteAssociation removes an association type from a schema.
func (h *Handler) DeleteAssociation(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	associationID := r.PathValue("associationId")
	corrID := api.CorrelationID(r.Context())

	if err := h.store.DeleteAssociation(r.Context(), objectType, associationID); err != nil {
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

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}

func isDuplicate(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed") ||
		strings.Contains(err.Error(), "already exists")
}

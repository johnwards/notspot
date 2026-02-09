package properties

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler serves the CRM properties API endpoints.
type Handler struct {
	store store.PropertyStore
}

// List returns all properties for the given object type.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	props, err := h.store.List(r.Context(), objectType)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	results := make([]any, len(props))
	for i := range props {
		results[i] = props[i]
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

// Create adds a new property definition.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var p domain.Property
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if p.Name == "" || p.Label == "" || p.Type == "" || p.FieldType == "" || p.GroupName == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			"Property name, label, type, fieldType, and groupName are required", corrID, nil))
		return
	}

	validTypes := map[string]bool{
		"string": true, "number": true, "date": true, "datetime": true,
		"enumeration": true, "bool": true, "phone_number": true,
	}
	if !validTypes[p.Type] {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			fmt.Sprintf("Invalid property type: %s", p.Type), corrID, nil))
		return
	}

	created, err := h.store.Create(r.Context(), objectType, &p)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
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

// Get retrieves a single property by name.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	name := r.PathValue("propertyName")
	corrID := api.CorrelationID(r.Context())

	p, err := h.store.Get(r.Context(), objectType, name)
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

// Update partially modifies a property definition.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	name := r.PathValue("propertyName")
	corrID := api.CorrelationID(r.Context())

	existing, err := h.store.Get(r.Context(), objectType, name)
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

	var patch domain.Property
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if patch.Label != "" {
		existing.Label = patch.Label
	}
	if patch.Description != "" {
		existing.Description = patch.Description
	}
	if patch.GroupName != "" {
		existing.GroupName = patch.GroupName
	}
	if patch.Options != nil {
		existing.Options = patch.Options
	}
	existing.DisplayOrder = patch.DisplayOrder
	existing.Hidden = patch.Hidden
	existing.FormField = patch.FormField

	updated, err := h.store.Update(r.Context(), objectType, name, existing)
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

// Archive soft-deletes a property definition.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	name := r.PathValue("propertyName")
	corrID := api.CorrelationID(r.Context())

	if err := h.store.Archive(r.Context(), objectType, name); err != nil {
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

type batchCreateInput struct {
	Inputs []domain.Property `json:"inputs"`
}

// BatchCreate inserts multiple property definitions.
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var input batchCreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	created, err := h.store.BatchCreate(r.Context(), objectType, input.Inputs)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	results := make([]any, len(created))
	for i := range created {
		results[i] = created[i]
	}
	api.WriteJSON(w, http.StatusCreated, api.CollectionResponse{Results: results})
}

type batchReadInput struct {
	Inputs []struct {
		Name string `json:"name"`
	} `json:"inputs"`
}

// BatchRead retrieves multiple properties by name.
func (h *Handler) BatchRead(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var input batchReadInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	names := make([]string, len(input.Inputs))
	for i, inp := range input.Inputs {
		names[i] = inp.Name
	}

	props, err := h.store.BatchRead(r.Context(), objectType, names)
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

	results := make([]any, len(props))
	for i := range props {
		results[i] = props[i]
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

type batchArchiveInput struct {
	Inputs []struct {
		Name string `json:"name"`
	} `json:"inputs"`
}

// BatchArchive soft-deletes multiple properties.
func (h *Handler) BatchArchive(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var input batchArchiveInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	names := make([]string, len(input.Inputs))
	for i, inp := range input.Inputs {
		names[i] = inp.Name
	}

	if err := h.store.BatchArchive(r.Context(), objectType, names); err != nil {
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

// ListGroups returns all property groups for the given object type.
func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	groups, err := h.store.ListGroups(r.Context(), objectType)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
		})
		return
	}

	results := make([]any, len(groups))
	for i := range groups {
		results[i] = groups[i]
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

// CreateGroup adds a new property group.
func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var g domain.PropertyGroup
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if g.Name == "" || g.Label == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(
			"Group name and label are required", corrID, nil))
		return
	}

	created, err := h.store.CreateGroup(r.Context(), objectType, &g)
	if err != nil {
		if isNotFound(err) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
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

// GetGroup retrieves a single property group by name.
func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	name := r.PathValue("groupName")
	corrID := api.CorrelationID(r.Context())

	g, err := h.store.GetGroup(r.Context(), objectType, name)
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

	api.WriteJSON(w, http.StatusOK, g)
}

// UpdateGroup partially modifies a property group.
func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	name := r.PathValue("groupName")
	corrID := api.CorrelationID(r.Context())

	existing, err := h.store.GetGroup(r.Context(), objectType, name)
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

	var patch domain.PropertyGroup
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if patch.Label != "" {
		existing.Label = patch.Label
	}
	existing.DisplayOrder = patch.DisplayOrder

	updated, err := h.store.UpdateGroup(r.Context(), objectType, name, existing)
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

// ArchiveGroup soft-deletes a property group.
func (h *Handler) ArchiveGroup(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	name := r.PathValue("groupName")
	corrID := api.CorrelationID(r.Context())

	if err := h.store.ArchiveGroup(r.Context(), objectType, name); err != nil {
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

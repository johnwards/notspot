package objects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler handles CRM object HTTP requests.
type Handler struct {
	store *store.Store
}

const maxBatchSize = 100

// Create handles POST /crm/v3/objects/{objectType}.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Properties map[string]string `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if body.Properties == nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("properties is required", corrID, nil))
		return
	}

	// Validate property values against property definitions.
	if err := h.validatePropertyValues(r.Context(), objectType, body.Properties); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
		return
	}

	obj, err := h.store.Objects.Create(r.Context(), objectType, body.Properties)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object type not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusCreated, obj)
}

// Get handles GET /crm/v3/objects/{objectType}/{objectId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	objectID := r.PathValue("objectId")
	corrID := api.CorrelationID(r.Context())

	props := parsePropertiesParam(r)
	idProperty := r.URL.Query().Get("idProperty")

	var obj *domain.Object
	var err error
	if idProperty != "" && idProperty != "hs_object_id" {
		obj, err = h.store.Objects.GetByProperty(r.Context(), objectType, idProperty, objectID, props)
	} else {
		obj, err = h.store.Objects.Get(r.Context(), objectType, objectID, props)
	}

	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, obj)
}

// List handles GET /crm/v3/objects/{objectType}.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	archived := false
	if r.URL.Query().Get("archived") == "true" {
		archived = true
	}

	opts := domain.ListOpts{
		Limit:      limit,
		After:      r.URL.Query().Get("after"),
		Properties: parsePropertiesParam(r),
		Archived:   archived,
	}

	page, err := h.store.Objects.List(r.Context(), objectType, opts)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object type not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(page.Results))
	for i, obj := range page.Results {
		results[i] = obj
	}

	resp := api.CollectionResponse{Results: results}
	if page.HasMore {
		resp.Paging = &api.Paging{
			Next: &api.PagingNext{After: page.After},
		}
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// Update handles PATCH /crm/v3/objects/{objectType}/{objectId}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	objectID := r.PathValue("objectId")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Properties map[string]string `json:"properties"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	obj, err := h.store.Objects.Update(r.Context(), objectType, objectID, body.Properties)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, obj)
}

// Archive handles DELETE /crm/v3/objects/{objectType}/{objectId}.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	objectID := r.PathValue("objectId")
	corrID := api.CorrelationID(r.Context())

	err := h.store.Objects.Archive(r.Context(), objectType, objectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BatchCreate handles POST /crm/v3/objects/{objectType}/batch/create.
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []domain.CreateInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if len(body.Inputs) > maxBatchSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 100", corrID, nil))
		return
	}

	for _, input := range body.Inputs {
		if input.Properties == nil {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Each input must have a properties field", corrID, nil))
			return
		}
	}

	result, err := h.store.Objects.BatchCreate(r.Context(), objectType, body.Inputs)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object type not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusCreated, result)
}

// BatchRead handles POST /crm/v3/objects/{objectType}/batch/read.
func (h *Handler) BatchRead(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []struct {
			ID string `json:"id"`
		} `json:"inputs"`
		Properties []string `json:"properties"`
		IDProperty string   `json:"idProperty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if len(body.Inputs) > maxBatchSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 100", corrID, nil))
		return
	}

	ids := make([]string, len(body.Inputs))
	for i, input := range body.Inputs {
		ids[i] = input.ID
	}

	result, err := h.store.Objects.BatchRead(r.Context(), objectType, ids, body.Properties, body.IDProperty)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object type not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, result)
}

// BatchUpdate handles POST /crm/v3/objects/{objectType}/batch/update.
func (h *Handler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []domain.UpdateInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if len(body.Inputs) > maxBatchSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 100", corrID, nil))
		return
	}

	result, err := h.store.Objects.BatchUpdate(r.Context(), objectType, body.Inputs)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, result)
}

// BatchUpsert handles POST /crm/v3/objects/{objectType}/batch/upsert.
func (h *Handler) BatchUpsert(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []domain.UpsertInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if len(body.Inputs) > maxBatchSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 100", corrID, nil))
		return
	}

	// Default idProperty for contacts is email.
	idProperty := "hs_object_id"
	if objectType == "contacts" || objectType == "0-1" {
		idProperty = "email"
	}

	result, err := h.store.Objects.BatchUpsert(r.Context(), objectType, body.Inputs, idProperty)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object type not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, result)
}

// BatchArchive handles POST /crm/v3/objects/{objectType}/batch/archive.
func (h *Handler) BatchArchive(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []struct {
			ID string `json:"id"`
		} `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if len(body.Inputs) > maxBatchSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 100", corrID, nil))
		return
	}

	ids := make([]string, len(body.Inputs))
	for i, input := range body.Inputs {
		ids[i] = input.ID
	}

	err := h.store.Objects.BatchArchive(r.Context(), objectType, ids)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Merge handles POST /crm/v3/objects/{objectType}/merge.
func (h *Handler) Merge(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		PrimaryObjectID string `json:"primaryObjectId"`
		ObjectIDToMerge string `json:"objectIdToMerge"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if body.PrimaryObjectID == "" || body.ObjectIDToMerge == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("primaryObjectId and objectIdToMerge are required", corrID, nil))
		return
	}

	obj, err := h.store.Objects.Merge(r.Context(), objectType, body.PrimaryObjectID, body.ObjectIDToMerge)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, obj)
}

func (h *Handler) validatePropertyValues(ctx context.Context, objectType string, properties map[string]string) error {
	if len(properties) == 0 {
		return nil
	}
	rows, err := h.store.DB.QueryContext(ctx,
		`SELECT pd.name, pd.type FROM property_definitions pd
		 JOIN object_types ot ON pd.object_type_id = ot.id
		 WHERE (ot.name = ? OR ot.id = ?)`,
		objectType, objectType,
	)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	propTypes := make(map[string]string)
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			continue
		}
		propTypes[name] = typ
	}

	for propName, propValue := range properties {
		if typ, ok := propTypes[propName]; ok {
			if typ == "number" {
				if _, err := strconv.ParseFloat(propValue, 64); err != nil {
					return fmt.Errorf("Property value %q is not valid for type %s", propValue, typ)
				}
			}
		}
	}
	return nil
}

func parsePropertiesParam(r *http.Request) []string {
	v := r.URL.Query().Get("properties")
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

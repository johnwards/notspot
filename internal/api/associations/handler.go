package associations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

const (
	maxBatchCreateSize = 2000
	maxBatchReadSize   = 1000
)

// Handler serves the CRM associations v4 API endpoints.
type Handler struct {
	store store.AssociationStore
}

// AssociateDefault handles creating a default association between two objects.
func (h *Handler) AssociateDefault(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	fromID := r.PathValue("fromId")
	toType := r.PathValue("to")
	toID := r.PathValue("toId")
	corrID := api.CorrelationID(r.Context())

	result, err := h.store.AssociateDefault(r.Context(), fromType, fromID, toType, toID)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{
		"results": []any{
			map[string]any{
				"toObjectId": toID,
				"associationTypes": []any{
					map[string]any{
						"category": result.Category,
						"typeId":   result.TypeID,
						"label":    nil,
					},
				},
			},
		},
	})
}

// AssociateWithLabels handles creating labeled associations between two objects.
func (h *Handler) AssociateWithLabels(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	fromID := r.PathValue("fromId")
	toType := r.PathValue("to")
	toID := r.PathValue("toId")
	corrID := api.CorrelationID(r.Context())

	var types []store.AssociationInput
	if err := json.NewDecoder(r.Body).Decode(&types); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	result, err := h.store.AssociateWithLabels(r.Context(), fromType, fromID, toType, toID, types)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]any{
		"results": []any{
			map[string]any{
				"toObjectId": toID,
				"associationTypes": []any{
					map[string]any{
						"category": result.Category,
						"typeId":   result.TypeID,
						"label":    nil,
					},
				},
			},
		},
	})
}

// GetAssociations handles listing associations from an object to a target type.
func (h *Handler) GetAssociations(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	fromID := r.PathValue("fromId")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	limit := 500
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 {
			limit = v
		}
	}
	after := r.URL.Query().Get("after")

	results, err := h.store.GetAssociations(r.Context(), fromType, fromID, toType)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	if results == nil {
		results = []domain.AssociationResult{}
	}

	startIdx := 0
	if after != "" {
		if idx, err := strconv.Atoi(after); err == nil && idx < len(results) {
			startIdx = idx
		}
	}
	endIdx := startIdx + limit
	var paging *api.Paging
	if endIdx < len(results) {
		paging = &api.Paging{Next: &api.PagingNext{After: strconv.Itoa(endIdx)}}
	}
	if startIdx > len(results) {
		startIdx = len(results)
	}
	if endIdx > len(results) {
		endIdx = len(results)
	}
	page := results[startIdx:endIdx]

	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: toAnySlice(page), Paging: paging})
}

// RemoveAssociations handles deleting all associations between two objects.
func (h *Handler) RemoveAssociations(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	fromID := r.PathValue("fromId")
	toType := r.PathValue("to")
	toID := r.PathValue("toId")
	corrID := api.CorrelationID(r.Context())

	err := h.store.RemoveAssociations(r.Context(), fromType, fromID, toType, toID)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListLabels handles listing association type labels between two object types.
func (h *Handler) ListLabels(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	labels, err := h.store.ListLabels(r.Context(), fromType, toType)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	if labels == nil {
		labels = []domain.AssociationLabel{}
	}
	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: toLabelAnySlice(labels)})
}

// CreateLabel handles creating a new association type label.
func (h *Handler) CreateLabel(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Label    string `json:"label"`
		Name     string `json:"name"`
		Category string `json:"associationCategory"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if body.Label == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("label is required", corrID, nil))
		return
	}

	label, err := h.store.CreateLabel(r.Context(), fromType, toType, body.Label, body.Category)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, label)
}

// UpdateLabel handles updating an existing association type label.
func (h *Handler) UpdateLabel(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		AssociationTypeID int    `json:"associationTypeId"`
		Label             string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if body.AssociationTypeID == 0 {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("associationTypeId is required", corrID, nil))
		return
	}

	updated, err := h.store.UpdateLabel(r.Context(), fromType, toType, body.AssociationTypeID, body.Label)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, updated)
}

// DeleteLabel handles deleting an association type label.
func (h *Handler) DeleteLabel(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	typeIDStr := r.PathValue("typeId")
	corrID := api.CorrelationID(r.Context())

	typeID, err := strconv.Atoi(typeIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid typeId", corrID, nil))
		return
	}
	if err := h.store.DeleteLabel(r.Context(), fromType, toType, typeID); err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BatchAssociateDefault handles creating default associations for multiple pairs.
func (h *Handler) BatchAssociateDefault(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []store.BatchAssocInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if len(body.Inputs) > maxBatchCreateSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 2000", corrID, nil))
		return
	}

	results, err := h.store.BatchAssociateDefault(r.Context(), fromType, toType, body.Inputs)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}

	writeBatchDefaultResults(w, results)
}

// BatchCreate handles creating labeled associations for multiple pairs.
func (h *Handler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []store.BatchAssocCreateInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if len(body.Inputs) > maxBatchCreateSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 2000", corrID, nil))
		return
	}

	results, err := h.store.BatchCreate(r.Context(), fromType, toType, body.Inputs)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}

	writeBatchCreateResults(w, results)
}

// BatchRead handles reading associations for multiple objects.
func (h *Handler) BatchRead(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []store.BatchAssocReadInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if len(body.Inputs) > maxBatchReadSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 1000", corrID, nil))
		return
	}

	results, err := h.store.BatchRead(r.Context(), fromType, toType, body.Inputs)
	if err != nil {
		writeStoreError(w, corrID, err)
		return
	}

	writeBatchReadResults(w, results)
}

// BatchArchive handles removing all associations for multiple pairs.
func (h *Handler) BatchArchive(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []store.BatchArchiveInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if len(body.Inputs) > maxBatchCreateSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 2000", corrID, nil))
		return
	}

	if err := h.store.BatchArchive(r.Context(), fromType, toType, body.Inputs); err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BatchArchiveLabels handles removing specific labeled associations for multiple pairs.
func (h *Handler) BatchArchiveLabels(w http.ResponseWriter, r *http.Request) {
	fromType := r.PathValue("from")
	toType := r.PathValue("to")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Inputs []store.BatchArchiveLabelInput `json:"inputs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}
	if len(body.Inputs) > maxBatchCreateSize {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Batch size exceeds maximum of 2000", corrID, nil))
		return
	}

	if err := h.store.BatchArchiveLabels(r.Context(), fromType, toType, body.Inputs); err != nil {
		writeStoreError(w, corrID, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeStoreError(w http.ResponseWriter, corrID string, err error) {
	if errors.Is(err, store.ErrNotFound) {
		api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(err.Error(), corrID))
		return
	}
	api.WriteError(w, http.StatusInternalServerError, &api.Error{
		Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR",
	})
}

func writeBatchDefaultResults(w http.ResponseWriter, results []store.BatchDefaultAssocResult) {
	ts := store.Now()
	out := make([]any, len(results))
	for i, r := range results {
		out[i] = map[string]any{
			"from": map[string]string{"id": r.FromID},
			"to": []any{
				map[string]any{
					"toObjectId": r.ToID,
					"associationTypes": []any{
						map[string]any{
							"category": r.Category,
							"typeId":   r.TypeID,
							"label":    nil,
						},
					},
				},
			},
		}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "COMPLETE", "startedAt": ts, "completedAt": ts,
		"results": out, "numErrors": 0,
	})
}

func writeBatchCreateResults(w http.ResponseWriter, results []store.BatchCreateResult) {
	ts := store.Now()
	out := make([]any, len(results))
	for i, r := range results {
		toTypes := make([]any, len(r.Labels))
		for j, l := range r.Labels {
			toTypes[j] = map[string]any{
				"category": l.Category,
				"typeId":   l.TypeID,
				"label":    l.Label,
			}
		}
		out[i] = map[string]any{
			"from": map[string]string{"id": r.FromObjectID},
			"to": []any{
				map[string]any{
					"toObjectId":       r.ToObjectID,
					"associationTypes": toTypes,
				},
			},
		}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "COMPLETE", "startedAt": ts, "completedAt": ts,
		"results": out, "numErrors": 0,
	})
}

func writeBatchReadResults(w http.ResponseWriter, results []store.BatchAssocResult) {
	ts := store.Now()
	out := make([]any, len(results))
	for i, r := range results {
		toResults := make([]any, len(r.To))
		for j, t := range r.To {
			toResults[j] = map[string]any{
				"toObjectId":       t.ToObjectID,
				"associationTypes": t.Types,
			}
		}
		out[i] = map[string]any{
			"from": map[string]string{"id": r.From},
			"to":   toResults,
		}
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "COMPLETE", "startedAt": ts, "completedAt": ts,
		"results": out, "numErrors": 0,
	})
}

func toAnySlice(results []domain.AssociationResult) []any {
	out := make([]any, len(results))
	for i, r := range results {
		out[i] = map[string]any{
			"toObjectId":       r.ToObjectID,
			"associationTypes": r.Types,
		}
	}
	return out
}

func toLabelAnySlice(labels []domain.AssociationLabel) []any {
	out := make([]any, len(labels))
	for i, l := range labels {
		out[i] = l
	}
	return out
}

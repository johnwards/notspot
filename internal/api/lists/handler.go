package lists

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler handles list HTTP requests.
type Handler struct {
	store *store.Store
}

// Create handles POST /crm/v3/lists.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Name           string          `json:"name"`
		ObjectTypeId   string          `json:"objectTypeId"`
		ProcessingType string          `json:"processingType"`
		FilterBranch   json.RawMessage `json:"filterBranch,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if body.Name == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("name is required", corrID, nil))
		return
	}
	if body.ObjectTypeId == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("objectTypeId is required", corrID, nil))
		return
	}
	if body.ProcessingType == "" {
		body.ProcessingType = "MANUAL"
	}

	list, err := h.store.Lists.Create(r.Context(), body.Name, body.ObjectTypeId, body.ProcessingType, body.FilterBranch)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			api.WriteError(w, http.StatusConflict, api.NewConflictError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, list)
}

// Get handles GET /crm/v3/lists/{listId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	list, err := h.store.Lists.Get(r.Context(), listID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, list)
}

// Delete handles DELETE /crm/v3/lists/{listId}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	err := h.store.Lists.Delete(r.Context(), listID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Restore handles PUT /crm/v3/lists/{listId}/restore.
func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	err := h.store.Lists.Restore(r.Context(), listID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateName handles PUT /crm/v3/lists/{listId}/update-list-name.
func (h *Handler) UpdateName(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	if body.Name == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("name is required", corrID, nil))
		return
	}

	list, err := h.store.Lists.UpdateName(r.Context(), listID, body.Name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		if errors.Is(err, store.ErrConflict) {
			api.WriteError(w, http.StatusConflict, api.NewConflictError(err.Error(), corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, list)
}

// UpdateFilters handles PUT /crm/v3/lists/{listId}/update-list-filters.
func (h *Handler) UpdateFilters(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		FilterBranch json.RawMessage `json:"filterBranch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	list, err := h.store.Lists.UpdateFilters(r.Context(), listID, body.FilterBranch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, list)
}

// Search handles POST /crm/v3/lists/search.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	var body struct {
		Query  string `json:"query"`
		Offset int    `json:"offset"`
		Count  int    `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	limit := body.Count
	if limit <= 0 {
		limit = 25
	}

	page, err := h.store.Lists.Search(r.Context(), domain.ListSearchOpts{
		Query:  body.Query,
		Offset: body.Offset,
		Limit:  limit,
	})
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(page.Results))
	for i, l := range page.Results {
		results[i] = l
	}

	resp := struct {
		Lists   []any `json:"lists"`
		Offset  int   `json:"offset"`
		HasMore bool  `json:"hasMore"`
		Total   int   `json:"total"`
	}{
		Lists:   results,
		Offset:  page.Offset,
		HasMore: page.HasMore,
		Total:   page.TotalCount,
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// GetMultiple handles GET /crm/v3/lists/.
func (h *Handler) GetMultiple(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	idsParam := r.URL.Query().Get("listId")
	if idsParam == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("listId query parameter is required", corrID, nil))
		return
	}

	ids := strings.Split(idsParam, ",")

	lists, err := h.store.Lists.GetMultiple(r.Context(), ids)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(lists))
	for i, l := range lists {
		results[i] = l
	}

	resp := struct {
		Lists []any `json:"lists"`
	}{
		Lists: results,
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// GetMemberships handles GET /crm/v3/lists/{listId}/memberships.
func (h *Handler) GetMemberships(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	after := r.URL.Query().Get("after")

	page, err := h.store.Lists.GetMemberships(r.Context(), listID, after, limit)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(page.Results))
	for i, m := range page.Results {
		results[i] = m
	}

	resp := api.CollectionResponse{Results: results}
	if page.HasMore {
		resp.Paging = &api.Paging{
			Next: &api.PagingNext{After: page.After},
		}
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// AddMembers handles PUT /crm/v3/lists/{listId}/memberships/add.
func (h *Handler) AddMembers(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	var recordIDs []string
	if err := json.NewDecoder(r.Body).Decode(&recordIDs); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	added, err := h.store.Lists.AddMembers(r.Context(), listID, recordIDs)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		if errors.Is(err, store.ErrDynamicListMutation) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, membershipUpdateResponse(added, nil))
}

// RemoveMembers handles PUT /crm/v3/lists/{listId}/memberships/remove.
func (h *Handler) RemoveMembers(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	var recordIDs []string
	if err := json.NewDecoder(r.Body).Decode(&recordIDs); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	removed, err := h.store.Lists.RemoveMembers(r.Context(), listID, recordIDs)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		if errors.Is(err, store.ErrDynamicListMutation) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, membershipUpdateResponse(nil, removed))
}

// AddAndRemoveMembers handles PUT /crm/v3/lists/{listId}/memberships/add-and-remove.
func (h *Handler) AddAndRemoveMembers(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	var body struct {
		RecordIdsToAdd    []string `json:"recordIdsToAdd"`
		RecordIdsToRemove []string `json:"recordIdsToRemove"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	added, err := h.store.Lists.AddMembers(r.Context(), listID, body.RecordIdsToAdd)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		if errors.Is(err, store.ErrDynamicListMutation) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	removed, err := h.store.Lists.RemoveMembers(r.Context(), listID, body.RecordIdsToRemove)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, membershipUpdateResponse(added, removed))
}

// RemoveAllMembers handles DELETE /crm/v3/lists/{listId}/memberships.
func (h *Handler) RemoveAllMembers(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("listId")
	corrID := api.CorrelationID(r.Context())

	err := h.store.Lists.RemoveAllMembers(r.Context(), listID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("List not found", corrID))
			return
		}
		if errors.Is(err, store.ErrDynamicListMutation) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(err.Error(), corrID, nil))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func membershipUpdateResponse(added, removed []string) map[string][]string {
	if added == nil {
		added = []string{}
	}
	if removed == nil {
		removed = []string{}
	}
	// HubSpot includes both correct and typo'd keys in the response.
	return map[string][]string{
		"recordIdsAdded":    added,
		"recordsIdsAdded":   added,
		"recordIdsMissing":  {},
		"recordIdsRemoved":  removed,
		"recordsIdsRemoved": removed,
	}
}

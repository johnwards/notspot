package objects

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/store"
)

// Search handles POST /crm/v3/objects/{objectType}/search.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	objectType := r.PathValue("objectType")
	corrID := api.CorrelationID(r.Context())

	var req domain.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid input JSON", corrID, nil))
		return
	}

	result, err := h.store.Search.Search(r.Context(), objectType, &req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Object type not found", corrID))
			return
		}
		var validationErr *store.ValidationError
		if errors.As(err, &validationErr) {
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError(validationErr.Message, corrID, []api.ErrorDetail{
				{Message: validationErr.Message},
			}))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status:        "error",
			Message:       err.Error(),
			CorrelationID: corrID,
			Category:      "INTERNAL_ERROR",
		})
		return
	}

	api.WriteJSON(w, http.StatusOK, result)
}

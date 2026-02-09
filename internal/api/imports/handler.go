package imports

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/store"
)

// Handler handles import HTTP requests.
type Handler struct {
	store *store.Store
}

// importRequest is the JSON config that accompanies the CSV file.
type importRequest struct {
	Name             string              `json:"name"`
	ImportOperations map[string]importOp `json:"importOperations"`
	Files            []importFile        `json:"files"`
}

type importOp struct {
	ObjectTypeID        string `json:"objectTypeId"`
	ImportOperationType string `json:"importOperationType"`
}

type importFile struct {
	FileName       string         `json:"fileName"`
	FileFormat     string         `json:"fileFormat"`
	FileImportPage fileImportPage `json:"fileImportPage"`
}

type fileImportPage struct {
	HasHeader      bool            `json:"hasHeader"`
	ColumnMappings []columnMapping `json:"columnMappings"`
}

type columnMapping struct {
	ColumnObjectTypeID string `json:"columnObjectTypeId"`
	ColumnName         string `json:"columnName"`
	PropertyName       string `json:"propertyName"`
}

// importMetadata is stored in the metadata column.
type importMetadata struct {
	ObjectLists []importObjectList `json:"objectLists"`
}

type importObjectList struct {
	ObjectType      string `json:"objectType"`
	ObjectsImported int    `json:"objectsImported"`
	ObjectsFailed   int    `json:"objectsFailed"`
}

// Start handles POST /crm/v3/imports.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	// Parse multipart form (max 32MB).
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid multipart form data", corrID, nil))
		return
	}

	// Read the importRequest JSON part.
	reqJSON := r.FormValue("importRequest")
	if reqJSON == "" {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("importRequest is required", corrID, nil))
		return
	}

	var req importRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Invalid importRequest JSON", corrID, nil))
		return
	}

	if len(req.Files) == 0 {
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("At least one file definition is required", corrID, nil))
		return
	}

	// Create the import record.
	imp, err := h.store.Imports.Create(r.Context(), req.Name, json.RawMessage(reqJSON))
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	// Update to PROCESSING state.
	if err := h.store.Imports.UpdateState(r.Context(), imp.ID, "PROCESSING", nil); err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	// Process the CSV file.
	file, _, err := r.FormFile("files")
	if err != nil {
		// No file uploaded — mark as failed.
		_ = h.store.Imports.UpdateState(r.Context(), imp.ID, "FAILED", nil)
		api.WriteError(w, http.StatusBadRequest, api.NewValidationError("CSV file is required", corrID, nil))
		return
	}
	defer func() { _ = file.Close() }()

	// Determine the object type and operation from the first file config.
	fileCfg := req.Files[0]
	var objectTypeID string
	var opType string
	for _, op := range req.ImportOperations {
		objectTypeID = op.ObjectTypeID
		opType = op.ImportOperationType
		break
	}
	if objectTypeID == "" && len(fileCfg.FileImportPage.ColumnMappings) > 0 {
		objectTypeID = fileCfg.FileImportPage.ColumnMappings[0].ColumnObjectTypeID
	}
	if opType == "" {
		opType = "CREATE"
	}

	// Build column-to-property mapping.
	colMap := make(map[int]string)
	for i, cm := range fileCfg.FileImportPage.ColumnMappings {
		colMap[i] = cm.PropertyName
	}

	reader := csv.NewReader(file)
	lineNumber := 0
	imported := 0
	failed := 0

	// Skip header if present.
	if fileCfg.FileImportPage.HasHeader {
		header, err := reader.Read()
		if err != nil {
			_ = h.store.Imports.UpdateState(r.Context(), imp.ID, "FAILED", nil)
			api.WriteError(w, http.StatusBadRequest, api.NewValidationError("Failed to read CSV header", corrID, nil))
			return
		}
		lineNumber++

		// If no column mappings provided, use header names as property names.
		if len(colMap) == 0 {
			for i, name := range header {
				colMap[i] = name
			}
		}
	}

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		lineNumber++
		if err != nil {
			failed++
			_ = h.store.Imports.AddError(r.Context(), imp.ID, "INVALID_ROW", err.Error(), "", objectTypeID, lineNumber)
			continue
		}

		// Build properties from the row.
		props := make(map[string]string)
		for i, value := range record {
			if propName, ok := colMap[i]; ok && propName != "" {
				props[propName] = value
			}
		}

		switch opType {
		case "CREATE":
			_, err = h.store.Objects.Create(r.Context(), objectTypeID, props)
		case "UPDATE":
			// For updates, we need an ID property in the mapping.
			if id, ok := props["hs_object_id"]; ok {
				delete(props, "hs_object_id")
				_, err = h.store.Objects.Update(r.Context(), objectTypeID, id, props)
			} else {
				err = errors.New("hs_object_id required for UPDATE operation")
			}
		case "UPSERT":
			// Use email as the default lookup property for contacts.
			idProp := "email"
			if objectTypeID != "0-1" && objectTypeID != "contacts" {
				idProp = "hs_object_id"
			}
			lookupValue := props[idProp]
			if lookupValue == "" {
				err = errors.New("lookup property " + idProp + " is empty")
			} else {
				existing, getErr := h.store.Objects.GetByProperty(r.Context(), objectTypeID, idProp, lookupValue, nil)
				if getErr != nil {
					_, err = h.store.Objects.Create(r.Context(), objectTypeID, props)
				} else {
					_, err = h.store.Objects.Update(r.Context(), objectTypeID, existing.ID, props)
				}
			}
		default:
			_, err = h.store.Objects.Create(r.Context(), objectTypeID, props)
		}

		if err != nil {
			failed++
			_ = h.store.Imports.AddError(r.Context(), imp.ID, "OBJECT_CREATE_ERROR", err.Error(), "", objectTypeID, lineNumber)
		} else {
			imported++
		}
	}

	// Build metadata.
	meta := importMetadata{
		ObjectLists: []importObjectList{
			{ObjectType: objectTypeID, ObjectsImported: imported, ObjectsFailed: failed},
		},
	}
	metaJSON, _ := json.Marshal(meta)

	finalState := "DONE"
	if imported == 0 && failed > 0 {
		finalState = "FAILED"
	}

	_ = h.store.Imports.UpdateState(r.Context(), imp.ID, finalState, metaJSON)

	// Re-read the import for the response.
	result, err := h.store.Imports.Get(r.Context(), imp.ID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, store.ImportResponse{
		ID:           result.ID,
		Name:         result.Name,
		State:        result.State,
		OptOutImport: result.OptOutImport,
		Metadata:     result.Metadata,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	})
}

// List handles GET /crm/v3/imports.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	corrID := api.CorrelationID(r.Context())

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	after := r.URL.Query().Get("after")

	imports, hasMore, nextAfter, err := h.store.Imports.List(r.Context(), limit, after)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(imports))
	for i, imp := range imports {
		results[i] = store.ImportResponse{
			ID:           imp.ID,
			Name:         imp.Name,
			State:        imp.State,
			OptOutImport: imp.OptOutImport,
			Metadata:     imp.Metadata,
			CreatedAt:    imp.CreatedAt,
			UpdatedAt:    imp.UpdatedAt,
		}
	}

	resp := api.CollectionResponse{Results: results}
	if hasMore {
		resp.Paging = &api.Paging{
			Next: &api.PagingNext{After: nextAfter},
		}
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// Get handles GET /crm/v3/imports/{importId}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	importID := r.PathValue("importId")
	corrID := api.CorrelationID(r.Context())

	imp, err := h.store.Imports.Get(r.Context(), importID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Import not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, store.ImportResponse{
		ID:           imp.ID,
		Name:         imp.Name,
		State:        imp.State,
		OptOutImport: imp.OptOutImport,
		Metadata:     imp.Metadata,
		CreatedAt:    imp.CreatedAt,
		UpdatedAt:    imp.UpdatedAt,
	})
}

// Cancel handles POST /crm/v3/imports/{importId}/cancel.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	importID := r.PathValue("importId")
	corrID := api.CorrelationID(r.Context())

	imp, err := h.store.Imports.Get(r.Context(), importID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Import not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	if err := h.store.Imports.UpdateState(r.Context(), importID, "CANCELED", imp.Metadata); err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	// Re-read for response.
	result, err := h.store.Imports.Get(r.Context(), importID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	api.WriteJSON(w, http.StatusOK, store.ImportResponse{
		ID:           result.ID,
		Name:         result.Name,
		State:        result.State,
		OptOutImport: result.OptOutImport,
		Metadata:     result.Metadata,
		CreatedAt:    result.CreatedAt,
		UpdatedAt:    result.UpdatedAt,
	})
}

// GetErrors handles GET /crm/v3/imports/{importId}/errors.
func (h *Handler) GetErrors(w http.ResponseWriter, r *http.Request) {
	importID := r.PathValue("importId")
	corrID := api.CorrelationID(r.Context())

	// Verify import exists.
	_, err := h.store.Imports.Get(r.Context(), importID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, api.NewNotFoundError("Import not found", corrID))
			return
		}
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	importErrors, err := h.store.Imports.GetErrors(r.Context(), importID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, &api.Error{Status: "error", Message: err.Error(), CorrelationID: corrID, Category: "INTERNAL_ERROR"})
		return
	}

	results := make([]any, len(importErrors))
	for i, ie := range importErrors {
		results[i] = ie
	}

	api.WriteJSON(w, http.StatusOK, api.CollectionResponse{Results: results})
}

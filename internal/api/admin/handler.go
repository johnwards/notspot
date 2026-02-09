package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/seed"
)

// Handler serves the admin API at /_notspot/.
type Handler struct {
	db *sql.DB
}

// dataTableNames lists all data tables in foreign-key-safe deletion order.
var dataTableNames = []string{
	"list_memberships",
	"associations",
	"property_value_history",
	"property_values",
	"import_errors",
	"request_log",
	"pipeline_stages",
	"objects",
	"imports",
	"exports",
	"lists",
	"pipelines",
	"association_types",
	"property_definitions",
	"property_groups",
	"owners",
	"object_types",
}

// Reset drops all data from all tables and re-runs seeds.
func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	for _, table := range dataTableNames {
		if _, err := h.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil { //nolint:gosec // table names are hardcoded constants
			corrID := api.CorrelationID(ctx)
			api.WriteError(w, http.StatusInternalServerError, &api.Error{
				Status:        "error",
				Message:       fmt.Sprintf("failed to clear table %s: %s", table, err),
				CorrelationID: corrID,
				Category:      "INTERNAL_ERROR",
			})
			return
		}
	}

	if err := seed.Seed(ctx, h.db); err != nil {
		corrID := api.CorrelationID(ctx)
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status:        "error",
			Message:       fmt.Sprintf("failed to re-seed: %s", err),
			CorrelationID: corrID,
			Category:      "INTERNAL_ERROR",
		})
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SeedData runs seed data without dropping existing data first.
func (h *Handler) SeedData(w http.ResponseWriter, r *http.Request) {
	if err := seed.Seed(r.Context(), h.db); err != nil {
		corrID := api.CorrelationID(r.Context())
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status:        "error",
			Message:       fmt.Sprintf("failed to seed: %s", err),
			CorrelationID: corrID,
			Category:      "INTERNAL_ERROR",
		})
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type requestLogEntry struct {
	ID            int64  `json:"id"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	StatusCode    int    `json:"statusCode"`
	RequestBody   string `json:"requestBody,omitempty"`
	ResponseBody  string `json:"responseBody,omitempty"`
	DurationMs    int64  `json:"durationMs"`
	CorrelationID string `json:"correlationId,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

// Requests returns request log entries, newest first, with cursor-based pagination.
func (h *Handler) Requests(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	var afterID int64
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			afterID = n
		}
	}

	query := `SELECT id, method, path, status_code, COALESCE(request_body,''), COALESCE(response_body,''),
			  COALESCE(duration_ms,0), COALESCE(correlation_id,''), created_at
			  FROM request_log`
	args := []any{}

	if afterID > 0 {
		query += " WHERE id < ?"
		args = append(args, afterID)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		corrID := api.CorrelationID(r.Context())
		api.WriteError(w, http.StatusInternalServerError, &api.Error{
			Status:        "error",
			Message:       fmt.Sprintf("query request log: %s", err),
			CorrelationID: corrID,
			Category:      "INTERNAL_ERROR",
		})
		return
	}
	defer func() { _ = rows.Close() }()

	entries := make([]requestLogEntry, 0, limit)
	for rows.Next() {
		var e requestLogEntry
		if err := rows.Scan(&e.ID, &e.Method, &e.Path, &e.StatusCode,
			&e.RequestBody, &e.ResponseBody, &e.DurationMs,
			&e.CorrelationID, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	results := make([]json.RawMessage, 0, len(entries))
	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	for i := range entries {
		b, _ := json.Marshal(entries[i])
		results = append(results, b)
	}

	resp := struct {
		Results []json.RawMessage `json:"results"`
		Paging  *api.Paging       `json:"paging,omitempty"`
	}{
		Results: results,
	}

	if hasMore {
		lastID := entries[len(entries)-1].ID
		resp.Paging = &api.Paging{
			Next: &api.PagingNext{
				After: strconv.FormatInt(lastID, 10),
			},
		}
	}

	api.WriteJSON(w, http.StatusOK, resp)
}

// ResetData clears all data tables within a transaction and re-seeds.
// Exported for reuse by tests or other callers.
func ResetData(ctx context.Context, db *sql.DB) error {
	for _, table := range dataTableNames {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil { //nolint:gosec // table names are hardcoded constants
			return fmt.Errorf("clear table %s: %w", table, err)
		}
	}
	return seed.Seed(ctx, db)
}

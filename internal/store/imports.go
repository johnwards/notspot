package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
)

// Import represents a HubSpot import job.
type Import struct {
	ID            string          `json:"id"`
	Name          string          `json:"name,omitempty"`
	State         string          `json:"state"`
	OptOutImport  bool            `json:"optOutImport"`
	Metadata      json.RawMessage `json:"metadata"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
	ObjectsCount  int             `json:"importRequestJson,omitempty"`
	ImportRequest json.RawMessage `json:"-"`
}

// ImportResponse is the JSON shape returned by the API.
type ImportResponse struct {
	ID           string          `json:"id"`
	Name         string          `json:"name,omitempty"`
	State        string          `json:"state"`
	OptOutImport bool            `json:"optOutImport"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    string          `json:"createdAt"`
	UpdatedAt    string          `json:"updatedAt"`
}

// ImportError represents an error that occurred during an import.
type ImportError struct {
	ID           string `json:"id"`
	ImportID     string `json:"-"`
	ErrorType    string `json:"errorType"`
	ErrorMessage string `json:"message"`
	InvalidValue string `json:"invalidValue,omitempty"`
	ObjectType   string `json:"objectType,omitempty"`
	LineNumber   int    `json:"lineNumber,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

// ImportStore defines the interface for import persistence.
type ImportStore interface {
	Create(ctx context.Context, name string, requestJSON json.RawMessage) (*Import, error)
	Get(ctx context.Context, id string) (*Import, error)
	List(ctx context.Context, limit int, after string) ([]*Import, bool, string, error)
	UpdateState(ctx context.Context, id, state string, metadata json.RawMessage) error
	AddError(ctx context.Context, importID, errType, errMsg, invalidValue, objectType string, lineNumber int) error
	GetErrors(ctx context.Context, importID string) ([]*ImportError, error)
}

// SQLiteImportStore implements ImportStore backed by SQLite.
type SQLiteImportStore struct {
	db *sql.DB
}

// NewSQLiteImportStore creates a new SQLiteImportStore.
func NewSQLiteImportStore(db *sql.DB) *SQLiteImportStore {
	return &SQLiteImportStore{db: db}
}

// Create inserts a new import record.
func (s *SQLiteImportStore) Create(ctx context.Context, name string, requestJSON json.RawMessage) (*Import, error) {
	ts := now()
	metadata := json.RawMessage(`{}`)

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO imports (name, state, request_json, metadata, created_at, updated_at) VALUES (?, 'STARTED', ?, ?, ?, ?)`,
		name, string(requestJSON), string(metadata), ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("insert import: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return &Import{
		ID:            strconv.FormatInt(id, 10),
		Name:          name,
		State:         "STARTED",
		Metadata:      metadata,
		ImportRequest: requestJSON,
		CreatedAt:     ts,
		UpdatedAt:     ts,
	}, nil
}

// Get retrieves an import by ID.
func (s *SQLiteImportStore) Get(ctx context.Context, id string) (*Import, error) {
	var imp Import
	var dbID int64
	var name sql.NullString
	var reqJSON sql.NullString
	var metaJSON sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, state, opt_out_import, request_json, metadata, created_at, updated_at FROM imports WHERE id = ?`,
		id,
	).Scan(&dbID, &name, &imp.State, &imp.OptOutImport, &reqJSON, &metaJSON, &imp.CreatedAt, &imp.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get import: %w", err)
	}

	imp.ID = strconv.FormatInt(dbID, 10)
	if name.Valid {
		imp.Name = name.String
	}
	if reqJSON.Valid {
		imp.ImportRequest = json.RawMessage(reqJSON.String)
	}
	if metaJSON.Valid {
		imp.Metadata = json.RawMessage(metaJSON.String)
	} else {
		imp.Metadata = json.RawMessage(`{}`)
	}

	return &imp, nil
}

// List returns a paginated list of imports.
//
//nolint:gocritic // named results provide clarity for multiple return values
func (s *SQLiteImportStore) List(ctx context.Context, limit int, after string) ([]*Import, bool, string, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, name, state, opt_out_import, metadata, created_at, updated_at FROM imports`
	args := []any{}

	if after != "" {
		query += ` WHERE id > ?`
		args = append(args, after)
	}

	query += ` ORDER BY id ASC LIMIT ?`
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, "", fmt.Errorf("list imports: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var imports []*Import
	for rows.Next() {
		var imp Import
		var dbID int64
		var name sql.NullString
		var metaJSON sql.NullString

		if err := rows.Scan(&dbID, &name, &imp.State, &imp.OptOutImport, &metaJSON, &imp.CreatedAt, &imp.UpdatedAt); err != nil {
			return nil, false, "", fmt.Errorf("scan import: %w", err)
		}

		imp.ID = strconv.FormatInt(dbID, 10)
		if name.Valid {
			imp.Name = name.String
		}
		if metaJSON.Valid {
			imp.Metadata = json.RawMessage(metaJSON.String)
		} else {
			imp.Metadata = json.RawMessage(`{}`)
		}
		imports = append(imports, &imp)
	}
	if err := rows.Err(); err != nil {
		return nil, false, "", fmt.Errorf("rows iteration: %w", err)
	}

	hasMore := false
	nextAfter := ""
	if len(imports) > limit {
		hasMore = true
		nextAfter = imports[limit-1].ID
		imports = imports[:limit]
	}

	return imports, hasMore, nextAfter, nil
}

// UpdateState changes the state of an import and updates metadata.
func (s *SQLiteImportStore) UpdateState(ctx context.Context, id, state string, metadata json.RawMessage) error {
	ts := now()
	metaStr := "{}"
	if metadata != nil {
		metaStr = string(metadata)
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE imports SET state = ?, metadata = ?, updated_at = ? WHERE id = ?`,
		state, metaStr, ts, id,
	)
	if err != nil {
		return fmt.Errorf("update import state: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AddError records an import error.
func (s *SQLiteImportStore) AddError(ctx context.Context, importID, errType, errMsg, invalidValue, objectType string, lineNumber int) error {
	ts := now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO import_errors (import_id, error_type, error_message, invalid_value, object_type, line_number, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		importID, errType, errMsg, invalidValue, objectType, lineNumber, ts,
	)
	if err != nil {
		return fmt.Errorf("add import error: %w", err)
	}
	return nil
}

// GetErrors returns all errors for an import.
func (s *SQLiteImportStore) GetErrors(ctx context.Context, importID string) ([]*ImportError, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, error_type, error_message, invalid_value, object_type, line_number, created_at
		 FROM import_errors WHERE import_id = ? ORDER BY id ASC`,
		importID,
	)
	if err != nil {
		return nil, fmt.Errorf("get import errors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var errs []*ImportError
	for rows.Next() {
		var ie ImportError
		var dbID int64
		var invalidValue, objectType sql.NullString
		if err := rows.Scan(&dbID, &ie.ErrorType, &ie.ErrorMessage, &invalidValue, &objectType, &ie.LineNumber, &ie.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan import error: %w", err)
		}
		ie.ID = strconv.FormatInt(dbID, 10)
		ie.ImportID = importID
		if invalidValue.Valid {
			ie.InvalidValue = invalidValue.String
		}
		if objectType.Valid {
			ie.ObjectType = objectType.String
		}
		errs = append(errs, &ie)
	}
	return errs, rows.Err()
}

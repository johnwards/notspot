package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
)

// Export represents a HubSpot export task.
type Export struct {
	ID               string          `json:"id"`
	Name             string          `json:"name,omitempty"`
	State            string          `json:"state"`
	ExportType       string          `json:"exportType"`
	ObjectType       string          `json:"objectType"`
	ObjectProperties []string        `json:"objectProperties"`
	RequestJSON      json.RawMessage `json:"-"`
	ResultData       []byte          `json:"-"`
	RecordCount      int             `json:"recordCount"`
	CreatedAt        string          `json:"createdAt"`
	UpdatedAt        string          `json:"updatedAt"`
}

// ExportStore defines the interface for export persistence.
type ExportStore interface {
	Create(ctx context.Context, name, exportType, objectType string, properties []string, requestJSON json.RawMessage) (*Export, error)
	Get(ctx context.Context, id string) (*Export, error)
	Complete(ctx context.Context, id string, data []byte, recordCount int) error
}

// SQLiteExportStore implements ExportStore backed by SQLite.
type SQLiteExportStore struct {
	db *sql.DB
}

// NewSQLiteExportStore creates a new SQLiteExportStore.
func NewSQLiteExportStore(db *sql.DB) *SQLiteExportStore {
	return &SQLiteExportStore{db: db}
}

// Create inserts a new export record.
func (s *SQLiteExportStore) Create(ctx context.Context, name, exportType, objectType string, properties []string, requestJSON json.RawMessage) (*Export, error) {
	ts := now()

	propsJSON, err := json.Marshal(properties)
	if err != nil {
		return nil, fmt.Errorf("marshal properties: %w", err)
	}

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO exports (name, state, export_type, object_type, object_properties, request_json, created_at, updated_at)
		 VALUES (?, 'ENQUEUED', ?, ?, ?, ?, ?, ?)`,
		name, exportType, objectType, string(propsJSON), string(requestJSON), ts, ts,
	)
	if err != nil {
		return nil, fmt.Errorf("insert export: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return &Export{
		ID:               strconv.FormatInt(id, 10),
		Name:             name,
		State:            "ENQUEUED",
		ExportType:       exportType,
		ObjectType:       objectType,
		ObjectProperties: properties,
		RequestJSON:      requestJSON,
		CreatedAt:        ts,
		UpdatedAt:        ts,
	}, nil
}

// Get retrieves an export by ID.
func (s *SQLiteExportStore) Get(ctx context.Context, id string) (*Export, error) {
	var exp Export
	var dbID int64
	var name sql.NullString
	var propsJSON string
	var reqJSON sql.NullString
	var resultData []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, state, export_type, object_type, object_properties, request_json, result_data, record_count, created_at, updated_at
		 FROM exports WHERE id = ?`,
		id,
	).Scan(&dbID, &name, &exp.State, &exp.ExportType, &exp.ObjectType, &propsJSON, &reqJSON, &resultData, &exp.RecordCount, &exp.CreatedAt, &exp.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get export: %w", err)
	}

	exp.ID = strconv.FormatInt(dbID, 10)
	if name.Valid {
		exp.Name = name.String
	}
	if reqJSON.Valid {
		exp.RequestJSON = json.RawMessage(reqJSON.String)
	}
	exp.ResultData = resultData

	if err := json.Unmarshal([]byte(propsJSON), &exp.ObjectProperties); err != nil {
		return nil, fmt.Errorf("unmarshal properties: %w", err)
	}

	return &exp, nil
}

// Complete marks an export as complete with the generated CSV data.
func (s *SQLiteExportStore) Complete(ctx context.Context, id string, data []byte, recordCount int) error {
	ts := now()

	res, err := s.db.ExecContext(ctx,
		`UPDATE exports SET state = 'COMPLETE', result_data = ?, record_count = ?, updated_at = ? WHERE id = ?`,
		data, recordCount, ts, id,
	)
	if err != nil {
		return fmt.Errorf("complete export: %w", err)
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

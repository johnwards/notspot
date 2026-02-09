package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type pipelineDef struct {
	objectType string
	label      string
	stages     []stageDef
}

type stageDef struct {
	label        string
	displayOrder int
	metadata     map[string]string
}

var defaultPipelines = []pipelineDef{
	{
		objectType: "deals",
		label:      "Sales Pipeline",
		stages: []stageDef{
			{label: "Appointment Scheduled", displayOrder: 0, metadata: map[string]string{"probability": "0.2"}},
			{label: "Qualified To Buy", displayOrder: 1, metadata: map[string]string{"probability": "0.3"}},
			{label: "Presentation Scheduled", displayOrder: 2, metadata: map[string]string{"probability": "0.4"}},
			{label: "Decision Maker Bought-In", displayOrder: 3, metadata: map[string]string{"probability": "0.6"}},
			{label: "Contract Sent", displayOrder: 4, metadata: map[string]string{"probability": "0.8"}},
			{label: "Closed Won", displayOrder: 5, metadata: map[string]string{"probability": "1.0", "isClosed": "true"}},
			{label: "Closed Lost", displayOrder: 6, metadata: map[string]string{"probability": "0.0", "isClosed": "true"}},
		},
	},
	{
		objectType: "tickets",
		label:      "Support Pipeline",
		stages: []stageDef{
			{label: "New", displayOrder: 0, metadata: map[string]string{"ticketState": "OPEN"}},
			{label: "Waiting on contact", displayOrder: 1, metadata: map[string]string{"ticketState": "OPEN"}},
			{label: "Waiting on us", displayOrder: 2, metadata: map[string]string{"ticketState": "OPEN"}},
			{label: "Closed", displayOrder: 3, metadata: map[string]string{"ticketState": "CLOSED"}},
		},
	},
}

// Pipelines inserts default pipelines and stages if none exist yet.
func Pipelines(ctx context.Context, db *sql.DB) error {
	for _, pd := range defaultPipelines {
		// Resolve object type ID.
		var typeID string
		err := db.QueryRowContext(ctx,
			`SELECT id FROM object_types WHERE name = ?`, pd.objectType,
		).Scan(&typeID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Object type not seeded yet — skip.
				continue
			}
			return fmt.Errorf("resolve object type %q: %w", pd.objectType, err)
		}

		// Check if a pipeline already exists for this object type.
		var count int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pipelines WHERE object_type_id = ?`, typeID,
		).Scan(&count); err != nil {
			return fmt.Errorf("count pipelines: %w", err)
		}
		if count > 0 {
			continue
		}

		ts := "2024-01-01T00:00:00.000Z"
		result, err := db.ExecContext(ctx,
			`INSERT INTO pipelines (object_type_id, label, display_order, archived, created_at, updated_at)
			 VALUES (?, ?, 0, FALSE, ?, ?)`,
			typeID, pd.label, ts, ts,
		)
		if err != nil {
			return fmt.Errorf("insert pipeline %q: %w", pd.label, err)
		}

		pipelineID, _ := result.LastInsertId()

		for _, sd := range pd.stages {
			metaJSON, err := json.Marshal(sd.metadata)
			if err != nil {
				return fmt.Errorf("marshal stage metadata: %w", err)
			}
			if _, err := db.ExecContext(ctx,
				`INSERT INTO pipeline_stages (pipeline_id, label, display_order, metadata, archived, created_at, updated_at)
				 VALUES (?, ?, ?, ?, FALSE, ?, ?)`,
				pipelineID, sd.label, sd.displayOrder, string(metaJSON), ts, ts,
			); err != nil {
				return fmt.Errorf("insert stage %q: %w", sd.label, err)
			}
		}
	}

	return nil
}

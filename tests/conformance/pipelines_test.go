package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

// TestListDefaultPipelines verifies that the seeded "Sales Pipeline" exists for deals.
func TestListDefaultPipelines(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/pipelines/deals", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) == 0 {
		t.Fatal("expected at least one seeded pipeline for deals")
	}

	// Find the Sales Pipeline.
	var found bool
	for _, r := range results {
		p := toObject(t, r)
		if assertIsString(t, p, "label") == "Sales Pipeline" {
			found = true
			assertIsString(t, p, "id")
			assertBoolField(t, p, "archived", false)
			assertISOTimestamp(t, assertIsString(t, p, "createdAt"))
			assertISOTimestamp(t, assertIsString(t, p, "updatedAt"))

			// Pipeline response must include stages array.
			stages := assertIsArray(t, p, "stages")
			if len(stages) < 2 {
				t.Errorf("expected Sales Pipeline to have at least 2 stages, got %d", len(stages))
			}

			// Verify first stage shape.
			if len(stages) > 0 {
				s := toObject(t, stages[0])
				assertIsString(t, s, "id")
				assertIsString(t, s, "label")
				assertFieldPresent(t, s, "displayOrder")
				assertFieldPresent(t, s, "metadata")
				assertBoolField(t, s, "archived", false)
			}
			break
		}
	}
	if !found {
		t.Error("seeded Sales Pipeline not found in deals pipelines")
	}
}

// TestListPipelinesNoPaging verifies pipelines list has no paging wrapper.
func TestListPipelinesNoPaging(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/pipelines/deals", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	// Must have "results" but no "paging" key.
	assertIsArray(t, body, "results")
	if _, ok := body["paging"]; ok {
		t.Error("pipeline list should not have a paging field")
	}
}

// TestCreatePipeline creates a new pipeline with stages.
func TestCreatePipeline(t *testing.T) {
	resetServer(t)

	input := map[string]any{
		"label":        "Custom Pipeline",
		"displayOrder": 1,
		"stages": []map[string]any{
			{"label": "Stage A", "displayOrder": 0, "metadata": map[string]string{"probability": "0.5"}},
			{"label": "Stage B", "displayOrder": 1, "metadata": map[string]string{"probability": "1.0"}},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/pipelines/deals", input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	assertStringField(t, body, "label", "Custom Pipeline")
	pipelineID := assertIsString(t, body, "id")
	if pipelineID == "" {
		t.Fatal("expected non-empty pipeline id")
	}
	assertBoolField(t, body, "archived", false)
	assertISOTimestamp(t, assertIsString(t, body, "createdAt"))
	assertISOTimestamp(t, assertIsString(t, body, "updatedAt"))

	stages := assertIsArray(t, body, "stages")
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stages))
	}
	s0 := toObject(t, stages[0])
	assertIsString(t, s0, "id")
}

// TestGetPipeline retrieves a pipeline by ID and verifies stages are included.
func TestGetPipeline(t *testing.T) {
	resetServer(t)

	// Create a pipeline first.
	input := map[string]any{
		"label":        "Get Test Pipeline",
		"displayOrder": 0,
		"stages": []map[string]any{
			{"label": "First", "displayOrder": 0, "metadata": map[string]string{"probability": "0.1"}},
		},
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/pipelines/deals", input)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)
	pipelineID := assertIsString(t, created, "id")

	// GET the pipeline.
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/pipelines/deals/%s", pipelineID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "id", pipelineID)
	assertStringField(t, body, "label", "Get Test Pipeline")
	assertBoolField(t, body, "archived", false)
	assertISOTimestamp(t, assertIsString(t, body, "createdAt"))
	assertISOTimestamp(t, assertIsString(t, body, "updatedAt"))

	stages := assertIsArray(t, body, "stages")
	if len(stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(stages))
	}
}

// TestUpdatePipeline updates a pipeline label via PATCH.
func TestUpdatePipeline(t *testing.T) {
	resetServer(t)

	// Create a pipeline.
	input := map[string]any{
		"label":        "Before Update",
		"displayOrder": 0,
		"stages": []map[string]any{
			{"label": "Open", "displayOrder": 0, "metadata": map[string]string{"probability": "0.5"}},
		},
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/pipelines/deals", input)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)
	pipelineID := assertIsString(t, created, "id")

	// PATCH the label.
	patch := map[string]any{"label": "After Update"}
	resp := doRequest(t, http.MethodPatch, fmt.Sprintf("/crm/v3/pipelines/deals/%s", pipelineID), patch)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "id", pipelineID)
	assertStringField(t, body, "label", "After Update")

	// Also test PUT (Replace).
	put := map[string]any{
		"label":        "Replaced Pipeline",
		"displayOrder": 2,
		"stages": []map[string]any{
			{"label": "New Stage", "displayOrder": 0, "metadata": map[string]string{"probability": "0.0"}},
		},
	}
	putResp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/pipelines/deals/%s", pipelineID), put)
	mustStatus(t, putResp, http.StatusOK)
	putBody := readJSON(t, putResp)

	assertStringField(t, putBody, "label", "Replaced Pipeline")
}

// TestDeletePipeline deletes a pipeline.
func TestDeletePipeline(t *testing.T) {
	resetServer(t)

	// Create a pipeline to delete.
	input := map[string]any{
		"label":        "To Delete",
		"displayOrder": 0,
		"stages": []map[string]any{
			{"label": "Temp", "displayOrder": 0, "metadata": map[string]string{"probability": "0.5"}},
		},
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/pipelines/deals", input)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)
	pipelineID := assertIsString(t, created, "id")

	// DELETE the pipeline.
	delResp := doRequest(t, http.MethodDelete, fmt.Sprintf("/crm/v3/pipelines/deals/%s", pipelineID), nil)
	mustStatus(t, delResp, http.StatusNoContent)
	_ = delResp.Body.Close()

	// GET should now 404.
	getResp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/pipelines/deals/%s", pipelineID), nil)
	mustStatus(t, getResp, http.StatusNotFound)
	errBody := readJSON(t, getResp)
	assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
}

// TestListStages lists stages for a pipeline.
func TestListStages(t *testing.T) {
	resetServer(t)

	// Get the default deals pipeline ID.
	pipelineID := getDefaultDealsPipelineID(t)

	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages", pipelineID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	// Default Sales Pipeline has 7 stages.
	if len(results) < 7 {
		t.Errorf("expected at least 7 stages for Sales Pipeline, got %d", len(results))
	}

	// No paging for stages either.
	if _, ok := body["paging"]; ok {
		t.Error("stage list should not have a paging field")
	}

	// Verify stage shape.
	s := toObject(t, results[0])
	assertIsString(t, s, "id")
	assertIsString(t, s, "label")
	assertFieldPresent(t, s, "displayOrder")
	assertFieldPresent(t, s, "metadata")
	assertBoolField(t, s, "archived", false)
	assertISOTimestamp(t, assertIsString(t, s, "createdAt"))
	assertISOTimestamp(t, assertIsString(t, s, "updatedAt"))
}

// TestCreateStage creates a new stage in a pipeline.
func TestCreateStage(t *testing.T) {
	resetServer(t)

	pipelineID := getDefaultDealsPipelineID(t)

	input := map[string]any{
		"label":        "Negotiation",
		"displayOrder": 10,
		"metadata":     map[string]string{"probability": "0.7"},
	}

	resp := doRequest(t, http.MethodPost, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages", pipelineID), input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	assertStringField(t, body, "label", "Negotiation")
	stageID := assertIsString(t, body, "id")
	if stageID == "" {
		t.Fatal("expected non-empty stage id")
	}
	assertBoolField(t, body, "archived", false)
	assertISOTimestamp(t, assertIsString(t, body, "createdAt"))
	assertISOTimestamp(t, assertIsString(t, body, "updatedAt"))

	meta := assertIsObject(t, body, "metadata")
	if meta != nil {
		if v, ok := meta["probability"]; !ok || v != "0.7" {
			t.Errorf("expected metadata.probability = 0.7, got %v", v)
		}
	}
}

// TestGetStage retrieves a stage by ID.
func TestGetStage(t *testing.T) {
	resetServer(t)

	pipelineID := getDefaultDealsPipelineID(t)

	// Create a stage.
	input := map[string]any{
		"label":        "Get Stage Test",
		"displayOrder": 20,
		"metadata":     map[string]string{"probability": "0.3"},
	}
	createResp := doRequest(t, http.MethodPost, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages", pipelineID), input)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)
	stageID := assertIsString(t, created, "id")

	// GET the stage.
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages/%s", pipelineID, stageID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "id", stageID)
	assertStringField(t, body, "label", "Get Stage Test")
	assertBoolField(t, body, "archived", false)
}

// TestUpdateStage updates a stage label and displayOrder via PATCH and PUT.
func TestUpdateStage(t *testing.T) {
	resetServer(t)

	pipelineID := getDefaultDealsPipelineID(t)

	// Create a stage.
	input := map[string]any{
		"label":        "Original Stage",
		"displayOrder": 30,
		"metadata":     map[string]string{"probability": "0.5"},
	}
	createResp := doRequest(t, http.MethodPost, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages", pipelineID), input)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)
	stageID := assertIsString(t, created, "id")

	// PATCH the label and displayOrder.
	patch := map[string]any{
		"label":        "Updated Stage",
		"displayOrder": 31,
	}
	resp := doRequest(t, http.MethodPatch, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages/%s", pipelineID, stageID), patch)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "id", stageID)
	assertStringField(t, body, "label", "Updated Stage")

	// Also test PUT (Replace).
	put := map[string]any{
		"label":        "Replaced Stage",
		"displayOrder": 32,
		"metadata":     map[string]string{"probability": "0.9"},
	}
	putResp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages/%s", pipelineID, stageID), put)
	mustStatus(t, putResp, http.StatusOK)
	putBody := readJSON(t, putResp)

	assertStringField(t, putBody, "label", "Replaced Stage")
}

// TestDeleteStage deletes a stage from a pipeline.
func TestDeleteStage(t *testing.T) {
	resetServer(t)

	pipelineID := getDefaultDealsPipelineID(t)

	// Create a stage to delete.
	input := map[string]any{
		"label":        "Stage To Delete",
		"displayOrder": 40,
		"metadata":     map[string]string{"probability": "0.1"},
	}
	createResp := doRequest(t, http.MethodPost, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages", pipelineID), input)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)
	stageID := assertIsString(t, created, "id")

	// DELETE the stage.
	delResp := doRequest(t, http.MethodDelete, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages/%s", pipelineID, stageID), nil)
	mustStatus(t, delResp, http.StatusNoContent)
	_ = delResp.Body.Close()

	// GET should 404.
	getResp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages/%s", pipelineID, stageID), nil)
	mustStatus(t, getResp, http.StatusNotFound)
	errBody := readJSON(t, getResp)
	assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
}

// TestTicketPipeline verifies that tickets have a seeded "Support Pipeline".
func TestTicketPipeline(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/pipelines/tickets", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) == 0 {
		t.Fatal("expected at least one seeded pipeline for tickets")
	}

	var found bool
	for _, r := range results {
		p := toObject(t, r)
		if assertIsString(t, p, "label") == "Support Pipeline" {
			found = true
			assertIsString(t, p, "id")
			assertBoolField(t, p, "archived", false)

			stages := assertIsArray(t, p, "stages")
			if len(stages) < 4 {
				t.Errorf("expected Support Pipeline to have at least 4 stages, got %d", len(stages))
			}

			// Verify ticket stage metadata includes ticketState.
			if len(stages) > 0 {
				s := toObject(t, stages[0])
				meta := assertIsObject(t, s, "metadata")
				if meta != nil {
					assertFieldPresent(t, meta, "ticketState")
				}
			}
			break
		}
	}
	if !found {
		t.Error("seeded Support Pipeline not found in tickets pipelines")
	}
}

// getDefaultDealsPipelineID returns the ID of the seeded "Sales Pipeline".
func getDefaultDealsPipelineID(t *testing.T) string {
	t.Helper()

	resp := doRequest(t, http.MethodGet, "/crm/v3/pipelines/deals", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	for _, r := range results {
		p := toObject(t, r)
		if assertIsString(t, p, "label") == "Sales Pipeline" {
			return assertIsString(t, p, "id")
		}
	}
	t.Fatal("seeded Sales Pipeline not found")
	return ""
}

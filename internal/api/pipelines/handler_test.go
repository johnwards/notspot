package pipelines_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/pipelines"
	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/domain"
	"github.com/johnwards/hubspot/internal/testhelpers"
)

func setupTestServer(t *testing.T) *http.ServeMux {
	t.Helper()
	db := testhelpers.NewTestDB(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO object_types (id, name, label_singular, label_plural, created_at, updated_at)
		 VALUES ('0-3', 'deals', 'Deal', 'Deals', '2024-01-01T00:00:00.000Z', '2024-01-01T00:00:00.000Z')`,
	); err != nil {
		t.Fatalf("seed object type: %v", err)
	}

	mux := http.NewServeMux()
	pipelines.RegisterRoutes(mux, db)
	return mux
}

func decode(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestListPipelines_Empty(t *testing.T) {
	mux := setupTestServer(t)

	req := httptest.NewRequest("GET", "/crm/v3/pipelines/deals", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.CollectionResponse
	decode(t, w.Body.Bytes(), &resp)
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestCreateAndGetPipeline(t *testing.T) {
	mux := setupTestServer(t)

	body := `{"label":"Sales Pipeline","displayOrder":0,"stages":[{"label":"New","displayOrder":0,"metadata":{"probability":"0.5"}}]}`
	req := httptest.NewRequest("POST", "/crm/v3/pipelines/deals", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var created domain.Pipeline
	decode(t, w.Body.Bytes(), &created)
	if created.Label != "Sales Pipeline" {
		t.Errorf("expected 'Sales Pipeline', got %q", created.Label)
	}
	if len(created.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(created.Stages))
	}
	if created.Stages[0].Metadata["probability"] != "0.5" {
		t.Errorf("expected probability '0.5', got %q", created.Stages[0].Metadata["probability"])
	}

	req = httptest.NewRequest("GET", "/crm/v3/pipelines/deals/"+created.ID, http.NoBody)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got domain.Pipeline
	decode(t, w.Body.Bytes(), &got)
	if got.ID != created.ID {
		t.Errorf("expected ID %q, got %q", created.ID, got.ID)
	}
}

func TestPatchPipeline(t *testing.T) {
	mux := setupTestServer(t)

	body := `{"label":"Original","displayOrder":0}`
	req := httptest.NewRequest("POST", "/crm/v3/pipelines/deals", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var created domain.Pipeline
	decode(t, w.Body.Bytes(), &created)

	patchBody := `{"label":"Updated"}`
	req = httptest.NewRequest("PATCH", "/crm/v3/pipelines/deals/"+created.ID, bytes.NewBufferString(patchBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated domain.Pipeline
	decode(t, w.Body.Bytes(), &updated)
	if updated.Label != "Updated" {
		t.Errorf("expected label 'Updated', got %q", updated.Label)
	}
}

func TestDeletePipeline(t *testing.T) {
	mux := setupTestServer(t)

	body := `{"label":"To Delete","displayOrder":0}`
	req := httptest.NewRequest("POST", "/crm/v3/pipelines/deals", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var created domain.Pipeline
	decode(t, w.Body.Bytes(), &created)

	req = httptest.NewRequest("DELETE", "/crm/v3/pipelines/deals/"+created.ID, http.NoBody)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest("GET", "/crm/v3/pipelines/deals/"+created.ID, http.NoBody)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestGetPipeline_NotFound(t *testing.T) {
	mux := setupTestServer(t)

	req := httptest.NewRequest("GET", "/crm/v3/pipelines/deals/999", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipeline_InvalidObjectType(t *testing.T) {
	mux := setupTestServer(t)

	req := httptest.NewRequest("GET", "/crm/v3/pipelines/nonexistent", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStageCRUD(t *testing.T) {
	mux := setupTestServer(t)

	body := `{"label":"Test Pipeline","displayOrder":0}`
	req := httptest.NewRequest("POST", "/crm/v3/pipelines/deals", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var p domain.Pipeline
	decode(t, w.Body.Bytes(), &p)

	stageBody := `{"label":"Stage 1","displayOrder":0,"metadata":{"probability":"0.5","key":"val"}}`
	req = httptest.NewRequest("POST", "/crm/v3/pipelines/deals/"+p.ID+"/stages", bytes.NewBufferString(stageBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var stage domain.PipelineStage
	decode(t, w.Body.Bytes(), &stage)
	if stage.Label != "Stage 1" {
		t.Errorf("expected 'Stage 1', got %q", stage.Label)
	}

	req = httptest.NewRequest("GET", "/crm/v3/pipelines/deals/"+p.ID+"/stages", http.NoBody)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var listResp api.CollectionResponse
	decode(t, w.Body.Bytes(), &listResp)
	if len(listResp.Results) != 1 {
		t.Errorf("expected 1 stage, got %d", len(listResp.Results))
	}

	req = httptest.NewRequest("GET", "/crm/v3/pipelines/deals/"+p.ID+"/stages/"+stage.ID, http.NoBody)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updateBody := `{"label":"Updated Stage"}`
	req = httptest.NewRequest("PATCH", "/crm/v3/pipelines/deals/"+p.ID+"/stages/"+stage.ID, bytes.NewBufferString(updateBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updatedStage domain.PipelineStage
	decode(t, w.Body.Bytes(), &updatedStage)
	if updatedStage.Label != "Updated Stage" {
		t.Errorf("expected 'Updated Stage', got %q", updatedStage.Label)
	}

	req = httptest.NewRequest("DELETE", "/crm/v3/pipelines/deals/"+p.ID+"/stages/"+stage.ID, http.NoBody)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPutPipeline(t *testing.T) {
	mux := setupTestServer(t)

	body := `{"label":"Original","displayOrder":0,"stages":[{"label":"S1","displayOrder":0,"metadata":{}}]}`
	req := httptest.NewRequest("POST", "/crm/v3/pipelines/deals", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var created domain.Pipeline
	decode(t, w.Body.Bytes(), &created)

	putBody := `{"label":"Replaced","displayOrder":1,"stages":[{"label":"NS1","displayOrder":0,"metadata":{"k":"v"}},{"label":"NS2","displayOrder":1,"metadata":{}}]}`
	req = httptest.NewRequest("PUT", "/crm/v3/pipelines/deals/"+created.ID, bytes.NewBufferString(putBody))
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var replaced domain.Pipeline
	decode(t, w.Body.Bytes(), &replaced)
	if replaced.Label != "Replaced" {
		t.Errorf("expected 'Replaced', got %q", replaced.Label)
	}
	if len(replaced.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(replaced.Stages))
	}
}

func TestCreatePipeline_InvalidJSON(t *testing.T) {
	mux := setupTestServer(t)

	req := httptest.NewRequest("POST", "/crm/v3/pipelines/deals", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

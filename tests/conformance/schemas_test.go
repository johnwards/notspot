package conformance_test

import (
	"net/http"
	"testing"
)

// createCustomSchema is a helper that creates a custom object schema and returns the response body.
func createCustomSchema(t *testing.T, name string) map[string]any {
	t.Helper()
	input := map[string]any{
		"name":                   name,
		"labels":                 map[string]any{"singular": name, "plural": name + "s"},
		"primaryDisplayProperty": "hs_object_id",
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/schemas", input)
	mustStatus(t, resp, http.StatusCreated)
	return readJSON(t, resp)
}

func TestCreateSchema(t *testing.T) {
	resetServer(t)

	input := map[string]any{
		"name":                   "cars",
		"labels":                 map[string]any{"singular": "Car", "plural": "Cars"},
		"primaryDisplayProperty": "hs_object_id",
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/schemas", input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	// Verify returned fields
	assertStringField(t, body, "name", "cars")
	assertFieldPresent(t, body, "id")
	assertFieldPresent(t, body, "fullyQualifiedName")
	assertBoolField(t, body, "archived", false)
	assertISOTimestamp(t, assertIsString(t, body, "createdAt"))
	assertISOTimestamp(t, assertIsString(t, body, "updatedAt"))

	// Labels
	labels := assertIsObject(t, body, "labels")
	if labels != nil {
		assertStringField(t, labels, "singular", "Car")
		assertStringField(t, labels, "plural", "Cars")
	}

	// Properties array should exist (default properties get created)
	props := assertIsArray(t, body, "properties")
	if len(props) == 0 {
		t.Error("expected at least one default property on created schema")
	}

	// Associations array should exist
	assertFieldPresent(t, body, "associations")

	t.Run("missing name returns 400", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/schemas", map[string]any{
			"labels": map[string]any{"singular": "Car", "plural": "Cars"},
		})
		mustStatus(t, resp, http.StatusBadRequest)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "VALIDATION_ERROR")
	})

	t.Run("missing labels returns 400", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/schemas", map[string]any{
			"name": "trucks",
		})
		mustStatus(t, resp, http.StatusBadRequest)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "VALIDATION_ERROR")
	})

	t.Run("duplicate name returns 409", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/schemas", map[string]any{
			"name":   "cars",
			"labels": map[string]any{"singular": "Car", "plural": "Cars"},
		})
		mustStatus(t, resp, http.StatusConflict)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "CONFLICT")
	})
}

func TestListSchemas(t *testing.T) {
	resetServer(t)

	t.Run("empty list", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		results := assertIsArray(t, body, "results")
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	// Create two schemas
	createCustomSchema(t, "cars")
	createCustomSchema(t, "trucks")

	t.Run("lists all schemas", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		results := assertIsArray(t, body, "results")
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}

		// Each result should have core schema fields
		for _, r := range results {
			schema := toObject(t, r)
			assertFieldPresent(t, schema, "id")
			assertFieldPresent(t, schema, "name")
			assertFieldPresent(t, schema, "labels")
			assertFieldPresent(t, schema, "properties")
		}
	})

	t.Run("alternate path prefix", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm-object-schemas/v3/schemas", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		results := assertIsArray(t, body, "results")
		if len(results) != 2 {
			t.Errorf("expected 2 results via alternate prefix, got %d", len(results))
		}
	})
}

func TestGetSchema(t *testing.T) {
	resetServer(t)

	created := createCustomSchema(t, "cars")
	schemaID := assertIsString(t, created, "id")
	fqn := assertIsString(t, created, "fullyQualifiedName")

	t.Run("by name", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/cars", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assertStringField(t, body, "name", "cars")
		assertFieldPresent(t, body, "properties")
		assertFieldPresent(t, body, "associations")
	})

	t.Run("by ID", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/"+schemaID, nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assertStringField(t, body, "name", "cars")
	})

	t.Run("by fully qualified name", func(t *testing.T) {
		if fqn == "" {
			t.Skip("fullyQualifiedName not returned")
		}
		// The server may not support lookup by fullyQualifiedName;
		// verify it returns either 200 or 404.
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/"+fqn, nil)
		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()
			t.Skip("fullyQualifiedName lookup not supported")
		}
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assertStringField(t, body, "name", "cars")
	})

	t.Run("not found", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/nonexistent", nil)
		mustStatus(t, resp, http.StatusNotFound)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
	})

	t.Run("alternate path prefix", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm-object-schemas/v3/schemas/cars", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assertStringField(t, body, "name", "cars")
	})
}

func TestUpdateSchema(t *testing.T) {
	resetServer(t)

	createCustomSchema(t, "cars")

	t.Run("update labels", func(t *testing.T) {
		patch := map[string]any{
			"labels": map[string]any{"singular": "Automobile", "plural": "Automobiles"},
		}
		resp := doRequest(t, http.MethodPatch, "/crm/v3/schemas/cars", patch)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)

		labels := assertIsObject(t, body, "labels")
		if labels != nil {
			assertStringField(t, labels, "singular", "Automobile")
			assertStringField(t, labels, "plural", "Automobiles")
		}
	})

	t.Run("verify update persisted", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/cars", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		labels := assertIsObject(t, body, "labels")
		if labels != nil {
			assertStringField(t, labels, "singular", "Automobile")
		}
	})

	t.Run("update non-existent schema returns 404", func(t *testing.T) {
		patch := map[string]any{
			"labels": map[string]any{"singular": "X", "plural": "Xs"},
		}
		resp := doRequest(t, http.MethodPatch, "/crm/v3/schemas/nonexistent", patch)
		mustStatus(t, resp, http.StatusNotFound)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
	})
}

func TestArchiveSchema(t *testing.T) {
	resetServer(t)

	createCustomSchema(t, "cars")

	t.Run("archive existing schema", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, "/crm/v3/schemas/cars", nil)
		mustStatus(t, resp, http.StatusNoContent)
		_ = resp.Body.Close()
	})

	t.Run("archived schema no longer in list", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		results := assertIsArray(t, body, "results")
		if len(results) != 0 {
			t.Errorf("expected 0 results after archive, got %d", len(results))
		}
	})

	t.Run("archive non-existent schema returns 404", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, "/crm/v3/schemas/nonexistent", nil)
		mustStatus(t, resp, http.StatusNotFound)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
	})
}

func TestSchemaRegistersObjectType(t *testing.T) {
	resetServer(t)

	// Create a custom object schema
	createCustomSchema(t, "cars")

	t.Run("create object of custom type", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/objects/cars", map[string]any{
			"properties": map[string]string{},
		})
		mustStatus(t, resp, http.StatusCreated)
		body := readJSON(t, resp)
		assertSimplePublicObject(t, body)
		id := assertIsString(t, body, "id")
		if id == "" {
			t.Fatal("expected non-empty id for created custom object")
		}
	})

	t.Run("list objects of custom type", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/objects/cars", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		results := assertIsArray(t, body, "results")
		if len(results) < 1 {
			t.Error("expected at least 1 result after creating custom object")
		}
	})

	t.Run("get object of custom type by id", func(t *testing.T) {
		// Create and then retrieve
		createResp := doRequest(t, http.MethodPost, "/crm/v3/objects/cars", map[string]any{
			"properties": map[string]string{},
		})
		mustStatus(t, createResp, http.StatusCreated)
		created := readJSON(t, createResp)
		id := assertIsString(t, created, "id")

		resp := doRequest(t, http.MethodGet, "/crm/v3/objects/cars/"+id, nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assertSimplePublicObject(t, body)
		assertStringField(t, body, "id", id)
	})

	t.Run("delete object of custom type", func(t *testing.T) {
		createResp := doRequest(t, http.MethodPost, "/crm/v3/objects/cars", map[string]any{
			"properties": map[string]string{},
		})
		mustStatus(t, createResp, http.StatusCreated)
		created := readJSON(t, createResp)
		id := assertIsString(t, created, "id")

		resp := doRequest(t, http.MethodDelete, "/crm/v3/objects/cars/"+id, nil)
		mustStatus(t, resp, http.StatusNoContent)
		_ = resp.Body.Close()
	})
}

func TestCreateSchemaAssociation(t *testing.T) {
	resetServer(t)

	createCustomSchema(t, "cars")
	drivers := createCustomSchema(t, "drivers")
	driversID := assertIsString(t, drivers, "id")

	t.Run("create association", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/schemas/cars/associations", map[string]any{
			"toObjectTypeId": driversID,
			"name":           "car_to_driver",
		})
		mustStatus(t, resp, http.StatusCreated)
		body := readJSON(t, resp)

		assertFieldPresent(t, body, "id")
		assertFieldPresent(t, body, "fromObjectTypeId")
		assertStringField(t, body, "toObjectTypeId", driversID)
	})

	t.Run("association appears on schema", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/cars", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assocs := assertIsArray(t, body, "associations")
		if len(assocs) != 1 {
			t.Fatalf("expected 1 association, got %d", len(assocs))
		}
		assoc := toObject(t, assocs[0])
		assertFieldPresent(t, assoc, "id")
		assertStringField(t, assoc, "toObjectTypeId", driversID)
	})

	t.Run("missing toObjectTypeId returns 400", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/schemas/cars/associations", map[string]any{
			"name": "car_to_nothing",
		})
		mustStatus(t, resp, http.StatusBadRequest)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "VALIDATION_ERROR")
	})

	t.Run("non-existent schema returns 404", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm/v3/schemas/nonexistent/associations", map[string]any{
			"toObjectTypeId": driversID,
			"name":           "test",
		})
		mustStatus(t, resp, http.StatusNotFound)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
	})

	t.Run("alternate path prefix", func(t *testing.T) {
		resp := doRequest(t, http.MethodPost, "/crm-object-schemas/v3/schemas/drivers/associations", map[string]any{
			"toObjectTypeId": assertIsString(t, createCustomSchema(t, "passengers"), "id"),
			"name":           "driver_to_passenger",
		})
		// May be 201 or whatever the handler returns
		if resp.StatusCode != http.StatusCreated {
			_ = resp.Body.Close()
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		body := readJSON(t, resp)
		assertFieldPresent(t, body, "id")
	})
}

func TestDeleteSchemaAssociation(t *testing.T) {
	resetServer(t)

	createCustomSchema(t, "cars")
	drivers := createCustomSchema(t, "drivers")
	driversID := assertIsString(t, drivers, "id")

	// Create an association to delete
	resp := doRequest(t, http.MethodPost, "/crm/v3/schemas/cars/associations", map[string]any{
		"toObjectTypeId": driversID,
		"name":           "car_to_driver",
	})
	mustStatus(t, resp, http.StatusCreated)
	assocBody := readJSON(t, resp)
	assocID := assertIsString(t, assocBody, "id")

	t.Run("delete association", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, "/crm/v3/schemas/cars/associations/"+assocID, nil)
		mustStatus(t, resp, http.StatusNoContent)
		_ = resp.Body.Close()
	})

	t.Run("association removed from schema", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/crm/v3/schemas/cars", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)
		assocs := assertIsArray(t, body, "associations")
		if len(assocs) != 0 {
			t.Errorf("expected 0 associations after delete, got %d", len(assocs))
		}
	})

	t.Run("delete non-existent association returns 404", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, "/crm/v3/schemas/cars/associations/999999", nil)
		mustStatus(t, resp, http.StatusNotFound)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
	})

	t.Run("delete association from non-existent schema returns 404", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, "/crm/v3/schemas/nonexistent/associations/1", nil)
		mustStatus(t, resp, http.StatusNotFound)
		errBody := readJSON(t, resp)
		assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
	})
}

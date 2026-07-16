package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestCreateContact(t *testing.T) {
	resetServer(t)

	body := map[string]any{
		"properties": map[string]string{
			"email":     "test@example.com",
			"firstname": "Test",
			"lastname":  "User",
		},
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts", body)
	mustStatus(t, resp, http.StatusCreated)
	result := readJSON(t, resp)

	assertSimplePublicObject(t, result)

	props := assertIsObject(t, result, "properties")
	assertStringField(t, props, "email", "test@example.com")
	assertStringField(t, props, "firstname", "Test")
	assertStringField(t, props, "lastname", "User")
}

func TestGetContact(t *testing.T) {
	resetServer(t)

	created := createContact(t, map[string]string{
		"email":     "get@example.com",
		"firstname": "Get",
		"lastname":  "Test",
	})
	id := assertIsString(t, created, "id")

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+id+"?properties=email,firstname,lastname", nil)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertSimplePublicObject(t, result)
	assertStringField(t, result, "id", id)

	props := assertIsObject(t, result, "properties")
	assertStringField(t, props, "email", "get@example.com")
	assertStringField(t, props, "firstname", "Get")
	assertStringField(t, props, "lastname", "Test")
}

func TestGetContactByIdProperty(t *testing.T) {
	resetServer(t)

	created := createContact(t, map[string]string{
		"email":     "idprop@example.com",
		"firstname": "IdProp",
	})
	id := assertIsString(t, created, "id")

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/idprop@example.com?idProperty=email&properties=email,firstname", nil)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertSimplePublicObject(t, result)
	assertStringField(t, result, "id", id)

	props := assertIsObject(t, result, "properties")
	assertStringField(t, props, "email", "idprop@example.com")
}

func TestListContacts(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"email": "list1@example.com"})
	createContact(t, map[string]string{"email": "list2@example.com"})

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts", nil)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	results := assertIsArray(t, result, "results")
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Verify each result is a valid object.
	for _, r := range results {
		obj := toObject(t, r)
		assertSimplePublicObject(t, obj)
	}
}

func TestListContactsPagination(t *testing.T) {
	resetServer(t)

	// Create 12 contacts so we exceed the default page size of 10.
	for i := 0; i < 12; i++ {
		createContact(t, map[string]string{
			"email": fmt.Sprintf("page%d@example.com", i),
		})
	}

	// First page with limit=5.
	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=5", nil)
	mustStatus(t, resp, http.StatusOK)
	page1 := readJSON(t, resp)

	results1 := assertIsArray(t, page1, "results")
	if len(results1) != 5 {
		t.Fatalf("expected 5 results on page 1, got %d", len(results1))
	}

	// Should have paging with a next cursor.
	assertPaging(t, page1)
	paging := assertIsObject(t, page1, "paging")
	next := assertIsObject(t, paging, "next")
	after := assertIsString(t, next, "after")
	if after == "" {
		t.Fatal("expected non-empty after cursor")
	}

	// Second page using the after cursor.
	resp2 := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=5&after="+after, nil)
	mustStatus(t, resp2, http.StatusOK)
	page2 := readJSON(t, resp2)

	results2 := assertIsArray(t, page2, "results")
	if len(results2) != 5 {
		t.Fatalf("expected 5 results on page 2, got %d", len(results2))
	}
}

func TestListContactsProperties(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{
		"email":     "props@example.com",
		"firstname": "Props",
		"lastname":  "Test",
	})

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?properties=email,firstname", nil)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	results := assertIsArray(t, result, "results")
	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}

	obj := toObject(t, results[0])
	props := assertIsObject(t, obj, "properties")

	// The requested properties should be present.
	assertFieldPresent(t, props, "email")
	assertFieldPresent(t, props, "firstname")
}

func TestUpdateContact(t *testing.T) {
	resetServer(t)

	created := createContact(t, map[string]string{
		"email":     "update@example.com",
		"firstname": "Before",
	})
	id := assertIsString(t, created, "id")

	body := map[string]any{
		"properties": map[string]string{
			"firstname": "After",
		},
	}
	resp := doRequest(t, http.MethodPatch, "/crm/v3/objects/contacts/"+id, body)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertSimplePublicObject(t, result)
	props := assertIsObject(t, result, "properties")
	assertStringField(t, props, "firstname", "After")

	// updatedAt should be a valid timestamp (and typically different from createdAt).
	updatedAt := assertIsString(t, result, "updatedAt")
	assertISOTimestamp(t, updatedAt)
}

func TestArchiveContact(t *testing.T) {
	resetServer(t)

	created := createContact(t, map[string]string{
		"email": "archive@example.com",
	})
	id := assertIsString(t, created, "id")

	// Delete (archive) the contact.
	resp := doRequest(t, http.MethodDelete, "/crm/v3/objects/contacts/"+id, nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// GET returns the object with archived=true.
	resp2 := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+id, nil)
	mustStatus(t, resp2, http.StatusOK)
	archivedObj := readJSON(t, resp2)
	assertBoolField(t, archivedObj, "archived", true)
	assertFieldPresent(t, archivedObj, "archivedAt")
}

func TestBatchCreate(t *testing.T) {
	resetServer(t)

	body := map[string]any{
		"inputs": []map[string]any{
			{"properties": map[string]string{"email": "batch1@example.com", "firstname": "B1"}},
			{"properties": map[string]string{"email": "batch2@example.com", "firstname": "B2"}},
			{"properties": map[string]string{"email": "batch3@example.com", "firstname": "B3"}},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", body)
	mustStatus(t, resp, http.StatusCreated)
	result := readJSON(t, resp)

	assertStringField(t, result, "status", "COMPLETE")
	results := assertIsArray(t, result, "results")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		obj := toObject(t, r)
		assertSimplePublicObject(t, obj)
	}
}

func TestBatchRead(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"email": "br1@example.com"})
	c2 := createContact(t, map[string]string{"email": "br2@example.com"})
	id1 := assertIsString(t, c1, "id")
	id2 := assertIsString(t, c2, "id")

	body := map[string]any{
		"inputs": []map[string]string{
			{"id": id1},
			{"id": id2},
		},
		"properties": []string{"email"},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/read", body)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertStringField(t, result, "status", "COMPLETE")
	results := assertIsArray(t, result, "results")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		obj := toObject(t, r)
		assertSimplePublicObject(t, obj)
		props := assertIsObject(t, obj, "properties")
		assertFieldPresent(t, props, "email")
	}
}

func TestBatchUpdate(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"email": "bu1@example.com", "firstname": "Old1"})
	id1 := assertIsString(t, c1, "id")

	body := map[string]any{
		"inputs": []map[string]any{
			{
				"id":         id1,
				"properties": map[string]string{"firstname": "New1"},
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/update", body)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertStringField(t, result, "status", "COMPLETE")
	results := assertIsArray(t, result, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	obj := toObject(t, results[0])
	props := assertIsObject(t, obj, "properties")
	assertStringField(t, props, "firstname", "New1")
}

func TestBatchUpsert(t *testing.T) {
	resetServer(t)

	// Create an existing contact.
	existing := createContact(t, map[string]string{
		"email":     "upsert-existing@example.com",
		"firstname": "Existing",
	})
	existingID := assertIsString(t, existing, "id")

	// Upsert: update the existing one (matched by email) and create a new one.
	body := map[string]any{
		"inputs": []map[string]any{
			{
				"id":         "upsert-existing@example.com",
				"idProperty": "email",
				"properties": map[string]string{
					"email":     "upsert-existing@example.com",
					"firstname": "Updated",
				},
			},
			{
				"properties": map[string]string{
					"email":     "upsert-new@example.com",
					"firstname": "New",
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/upsert", body)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertStringField(t, result, "status", "COMPLETE")
	results := assertIsArray(t, result, "results")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify the existing contact was updated (same ID).
	found := false
	for _, r := range results {
		obj := toObject(t, r)
		if assertIsString(t, obj, "id") == existingID {
			props := assertIsObject(t, obj, "properties")
			assertStringField(t, props, "firstname", "Updated")
			found = true
		}
	}
	if !found {
		t.Error("existing contact not found in upsert results")
	}
}

func TestBatchArchive(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"email": "ba1@example.com"})
	c2 := createContact(t, map[string]string{"email": "ba2@example.com"})
	id1 := assertIsString(t, c1, "id")
	id2 := assertIsString(t, c2, "id")

	body := map[string]any{
		"inputs": []map[string]string{
			{"id": id1},
			{"id": id2},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/archive", body)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// Verify both contacts are now archived.
	for _, id := range []string{id1, id2} {
		r := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+id, nil)
		mustStatus(t, r, http.StatusOK)
		obj := readJSON(t, r)
		assertBoolField(t, obj, "archived", true)
	}
}

func TestMergeObjects(t *testing.T) {
	resetServer(t)

	primary := createContact(t, map[string]string{
		"email":     "primary@example.com",
		"firstname": "Primary",
	})
	toMerge := createContact(t, map[string]string{
		"email":     "tomerge@example.com",
		"firstname": "ToMerge",
	})
	primaryID := assertIsString(t, primary, "id")
	mergeID := assertIsString(t, toMerge, "id")

	body := map[string]any{
		"primaryObjectId": primaryID,
		"objectIdToMerge": mergeID,
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/merge", body)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertSimplePublicObject(t, result)
	assertStringField(t, result, "id", primaryID)

	// The merged object should be archived.
	resp2 := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+mergeID, nil)
	mustStatus(t, resp2, http.StatusOK)
	mergedObj := readJSON(t, resp2)
	assertBoolField(t, mergedObj, "archived", true)
}

func TestCreateObjectAllTypes(t *testing.T) {
	// Only test seeded object types (contacts tested elsewhere).
	objectTypes := []struct {
		name  string
		props map[string]string
	}{
		{"companies", map[string]string{"name": "Test Corp", "domain": "testcorp.com"}},
		{"deals", map[string]string{"dealname": "Test Deal", "amount": "1000"}},
		{"tickets", map[string]string{"subject": "Test Ticket"}},
	}

	for _, tc := range objectTypes {
		t.Run(tc.name, func(t *testing.T) {
			resetServer(t)

			body := map[string]any{"properties": tc.props}
			resp := doRequest(t, http.MethodPost, "/crm/v3/objects/"+tc.name, body)
			mustStatus(t, resp, http.StatusCreated)
			created := readJSON(t, resp)
			assertSimplePublicObject(t, created)
			id := assertIsString(t, created, "id")

			// GET should succeed.
			resp2 := doRequest(t, http.MethodGet, "/crm/v3/objects/"+tc.name+"/"+id, nil)
			mustStatus(t, resp2, http.StatusOK)
			got := readJSON(t, resp2)
			assertSimplePublicObject(t, got)
			assertStringField(t, got, "id", id)

			// DELETE should succeed.
			resp3 := doRequest(t, http.MethodDelete, "/crm/v3/objects/"+tc.name+"/"+id, nil)
			mustStatus(t, resp3, http.StatusNoContent)
			_ = resp3.Body.Close()

			// GET after DELETE returns archived object.
			resp4 := doRequest(t, http.MethodGet, "/crm/v3/objects/"+tc.name+"/"+id, nil)
			mustStatus(t, resp4, http.StatusOK)
			archivedObj := readJSON(t, resp4)
			assertBoolField(t, archivedObj, "archived", true)
		})
	}
}

func TestCreateObjectInvalidType(t *testing.T) {
	resetServer(t)

	body := map[string]any{
		"properties": map[string]string{"name": "test"},
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/nonexistent_type", body)
	mustStatus(t, resp, http.StatusNotFound)
	result := readJSON(t, resp)
	assertHubSpotError(t, result, "OBJECT_NOT_FOUND")
}

func TestGetObjectNotFound(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/999999", nil)
	mustStatus(t, resp, http.StatusNotFound)
	result := readJSON(t, resp)

	assertHubSpotError(t, result, "OBJECT_NOT_FOUND")
	assertFieldPresent(t, result, "status")
	assertFieldPresent(t, result, "message")
	assertFieldPresent(t, result, "correlationId")
}

func TestBatchLimitExceeded(t *testing.T) {
	resetServer(t)

	// Build 101 inputs to exceed the max batch size of 100.
	inputs := make([]map[string]any, 101)
	for i := range inputs {
		inputs[i] = map[string]any{
			"properties": map[string]string{
				"email": fmt.Sprintf("limit%d@example.com", i),
			},
		}
	}

	body := map[string]any{"inputs": inputs}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", body)
	mustStatus(t, resp, http.StatusBadRequest)
	result := readJSON(t, resp)
	assertHubSpotError(t, result, "VALIDATION_ERROR")
}

// patchContact updates a single contact property and returns the response body.
func patchContact(t *testing.T, id string, props map[string]string) map[string]any {
	t.Helper()
	body := map[string]any{"properties": props}
	resp := doRequest(t, http.MethodPatch, "/crm/v3/objects/contacts/"+id, body)
	mustStatus(t, resp, http.StatusOK)
	return readJSON(t, resp)
}

// assertHistoryEntry validates a single ValueWithTimestamp entry has the required
// trio (value/timestamp/sourceType) and returns its value.
func assertHistoryEntry(t *testing.T, entry map[string]any) string {
	t.Helper()
	value := assertIsString(t, entry, "value")
	assertISOTimestamp(t, assertIsString(t, entry, "timestamp"))
	assertStringField(t, entry, "sourceType", "API")
	return value
}

func TestGetContactPropertiesWithHistory(t *testing.T) {
	resetServer(t)

	// Set the tracked property at create time, then PATCH it twice with distinct values.
	created := createContact(t, map[string]string{
		"email":                        "history@example.com",
		"recent_conversion_event_name": "signup-form",
	})
	id := assertIsString(t, created, "id")

	patchContact(t, id, map[string]string{"recent_conversion_event_name": "demo-request"})
	patchContact(t, id, map[string]string{"recent_conversion_event_name": "pricing-page"})

	// propertiesWithHistory is an independent selector: request the property ONLY in
	// propertiesWithHistory (not in properties) and confirm it still appears.
	resp := doRequest(t, http.MethodGet,
		"/crm/v3/objects/contacts/"+id+"?properties=email&propertiesWithHistory=recent_conversion_event_name,does_not_exist", nil)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	assertSimplePublicObject(t, result)

	history := assertIsObject(t, result, "propertiesWithHistory")

	// Unknown property named in the param is silently ignored — no key, no error.
	if _, ok := history["does_not_exist"]; ok {
		t.Errorf("expected unknown property to be absent from propertiesWithHistory")
	}

	entries := assertIsArray(t, history, "recent_conversion_event_name")
	// One create-time row + two PATCH rows = 3 versions.
	if len(entries) != 3 {
		t.Fatalf("expected 3 history versions, got %d", len(entries))
	}

	// Newest-first: index 0 is the most recent write, last is the earliest.
	values := make([]string, len(entries))
	for i, e := range entries {
		values[i] = assertHistoryEntry(t, toObject(t, e))
	}
	if values[0] != "pricing-page" {
		t.Errorf("expected newest entry to be %q, got %q", "pricing-page", values[0])
	}
	if values[len(values)-1] != "signup-form" {
		t.Errorf("expected oldest entry to be %q, got %q", "signup-form", values[len(values)-1])
	}
	if values[1] != "demo-request" {
		t.Errorf("expected middle entry to be %q, got %q", "demo-request", values[1])
	}
}

func TestGetContactWithoutHistoryOmitsField(t *testing.T) {
	resetServer(t)

	created := createContact(t, map[string]string{"email": "nohistory@example.com"})
	id := assertIsString(t, created, "id")

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+id+"?properties=email", nil)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	// When propertiesWithHistory is not requested, the field is omitted entirely.
	if _, ok := result["propertiesWithHistory"]; ok {
		t.Errorf("expected propertiesWithHistory to be absent when not requested")
	}
}

func TestBatchReadPropertiesWithHistory(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"email": "bh1@example.com", "lifecyclestage": "subscriber"})
	c2 := createContact(t, map[string]string{"email": "bh2@example.com", "lifecyclestage": "subscriber"})
	id1 := assertIsString(t, c1, "id")
	id2 := assertIsString(t, c2, "id")

	patchContact(t, id1, map[string]string{"lifecyclestage": "lead"})
	patchContact(t, id2, map[string]string{"lifecyclestage": "lead"})

	body := map[string]any{
		"inputs": []map[string]string{
			{"id": id1},
			{"id": id2},
		},
		"properties":            []string{"email"},
		"propertiesWithHistory": []string{"lifecyclestage"},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/read", body)
	mustStatus(t, resp, http.StatusOK)
	result := readJSON(t, resp)

	results := assertIsArray(t, result, "results")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		obj := toObject(t, r)
		assertSimplePublicObject(t, obj)
		history := assertIsObject(t, obj, "propertiesWithHistory")
		entries := assertIsArray(t, history, "lifecyclestage")
		// create-time "subscriber" + PATCH "lead" = 2 versions, newest-first.
		if len(entries) != 2 {
			t.Fatalf("expected 2 history versions, got %d", len(entries))
		}
		if v := assertHistoryEntry(t, toObject(t, entries[0])); v != "lead" {
			t.Errorf("expected newest entry to be %q, got %q", "lead", v)
		}
		if v := assertHistoryEntry(t, toObject(t, entries[1])); v != "subscriber" {
			t.Errorf("expected oldest entry to be %q, got %q", "subscriber", v)
		}
	}
}

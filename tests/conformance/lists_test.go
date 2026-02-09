package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

// createList is a helper that creates a MANUAL list and returns the response body.
func createList(t *testing.T, name string) map[string]any {
	t.Helper()
	body := map[string]any{
		"name":           name,
		"objectTypeId":   "0-1",
		"processingType": "MANUAL",
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/lists", body)
	mustStatus(t, resp, http.StatusOK)
	return readJSON(t, resp)
}

func TestCreateList(t *testing.T) {
	resetServer(t)

	list := createList(t, "My Test List")

	assertIsString(t, list, "listId")
	assertStringField(t, list, "name", "My Test List")
	assertStringField(t, list, "objectTypeId", "0-1")
	assertStringField(t, list, "processingType", "MANUAL")
	assertFieldPresent(t, list, "createdAt")
	assertFieldPresent(t, list, "updatedAt")
	assertFieldPresent(t, list, "listVersion")
	assertFieldPresent(t, list, "size")
}

func TestGetList(t *testing.T) {
	resetServer(t)

	created := createList(t, "Get Test List")
	listID := assertIsString(t, created, "listId")

	resp := doRequest(t, http.MethodGet, "/crm/v3/lists/"+listID, nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "listId", listID)
	assertStringField(t, body, "name", "Get Test List")
	assertStringField(t, body, "objectTypeId", "0-1")
	assertStringField(t, body, "processingType", "MANUAL")

	// Non-existent list returns 404.
	resp = doRequest(t, http.MethodGet, "/crm/v3/lists/999999", nil)
	mustStatus(t, resp, http.StatusNotFound)
	errBody := readJSON(t, resp)
	assertHubSpotError(t, errBody, "OBJECT_NOT_FOUND")
}

func TestDeleteList(t *testing.T) {
	resetServer(t)

	created := createList(t, "Delete Test List")
	listID := assertIsString(t, created, "listId")

	// Delete returns 204.
	resp := doRequest(t, http.MethodDelete, "/crm/v3/lists/"+listID, nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// GET after delete returns 404.
	resp = doRequest(t, http.MethodGet, "/crm/v3/lists/"+listID, nil)
	mustStatus(t, resp, http.StatusNotFound)
	_ = resp.Body.Close()

	// Deleting non-existent list returns 404.
	resp = doRequest(t, http.MethodDelete, "/crm/v3/lists/999999", nil)
	mustStatus(t, resp, http.StatusNotFound)
	_ = resp.Body.Close()
}

func TestRestoreList(t *testing.T) {
	resetServer(t)

	created := createList(t, "Restore Test List")
	listID := assertIsString(t, created, "listId")

	// Delete then restore.
	resp := doRequest(t, http.MethodDelete, "/crm/v3/lists/"+listID, nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	resp = doRequest(t, http.MethodPut, "/crm/v3/lists/"+listID+"/restore", nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// GET should now succeed.
	resp = doRequest(t, http.MethodGet, "/crm/v3/lists/"+listID, nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	assertStringField(t, body, "name", "Restore Test List")
}

func TestUpdateListName(t *testing.T) {
	resetServer(t)

	created := createList(t, "Old Name")
	listID := assertIsString(t, created, "listId")

	resp := doRequest(t, http.MethodPut, "/crm/v3/lists/"+listID+"/update-list-name", map[string]any{
		"name": "New Name",
	})
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "New Name")
	assertStringField(t, body, "listId", listID)

	// Verify via GET.
	resp = doRequest(t, http.MethodGet, "/crm/v3/lists/"+listID, nil)
	mustStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	assertStringField(t, body, "name", "New Name")
}

func TestUpdateListFilters(t *testing.T) {
	resetServer(t)

	created := createList(t, "Filter Test List")
	listID := assertIsString(t, created, "listId")

	filterBranch := map[string]any{
		"filterBranchType":     "OR",
		"filterBranchOperator": "OR",
		"filterBranches": []any{
			map[string]any{
				"filterBranchType":     "AND",
				"filterBranchOperator": "AND",
				"filters": []any{
					map[string]any{
						"filterType": "PROPERTY",
						"property":   "firstname",
						"operation": map[string]any{
							"operationType":                "MULTISTRING",
							"operator":                     "IS_EQUAL_TO",
							"values":                       []any{"John"},
							"includeObjectsWithNoValueSet": false,
						},
					},
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPut, "/crm/v3/lists/"+listID+"/update-list-filters", map[string]any{
		"filterBranch": filterBranch,
	})
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "listId", listID)
	assertFieldPresent(t, body, "filterBranch")
}

func TestSearchLists(t *testing.T) {
	resetServer(t)

	createList(t, "Alpha List")
	createList(t, "Beta List")
	createList(t, "Alpha Second")

	// Search for "Alpha" — should match 2.
	resp := doRequest(t, http.MethodPost, "/crm/v3/lists/search", map[string]any{
		"query": "Alpha",
		"count": 10,
	})
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	lists := assertIsArray(t, body, "lists")
	if len(lists) != 2 {
		t.Errorf("expected 2 results for 'Alpha', got %d", len(lists))
	}
	assertFieldPresent(t, body, "offset")
	assertFieldPresent(t, body, "hasMore")
	assertFieldPresent(t, body, "total")

	// Search with no query — should return all.
	resp = doRequest(t, http.MethodPost, "/crm/v3/lists/search", map[string]any{
		"query": "",
		"count": 25,
	})
	mustStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	lists = assertIsArray(t, body, "lists")
	if len(lists) < 3 {
		t.Errorf("expected at least 3 results, got %d", len(lists))
	}
}

func TestAddMembers(t *testing.T) {
	resetServer(t)

	list := createList(t, "Add Members Test")
	listID := assertIsString(t, list, "listId")

	c1 := createContact(t, map[string]string{"email": "add1@test.com"})
	contactID := assertIsString(t, c1, "id")

	resp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/add", listID), []string{contactID})
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertFieldPresent(t, body, "recordIdsAdded")
	added := assertIsArray(t, body, "recordIdsAdded")
	if len(added) != 1 {
		t.Errorf("expected 1 added, got %d", len(added))
	}
	// HubSpot includes typo'd field.
	assertFieldPresent(t, body, "recordsIdsAdded")
}

func TestRemoveMembers(t *testing.T) {
	resetServer(t)

	list := createList(t, "Remove Members Test")
	listID := assertIsString(t, list, "listId")

	c1 := createContact(t, map[string]string{"email": "rm1@test.com"})
	contactID := assertIsString(t, c1, "id")

	// Add first.
	resp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/add", listID), []string{contactID})
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Remove.
	resp = doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/remove", listID), []string{contactID})
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertFieldPresent(t, body, "recordIdsRemoved")
	removed := assertIsArray(t, body, "recordIdsRemoved")
	if len(removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(removed))
	}
	assertFieldPresent(t, body, "recordsIdsRemoved")
}

func TestAddAndRemoveMembers(t *testing.T) {
	resetServer(t)

	list := createList(t, "Add Remove Members Test")
	listID := assertIsString(t, list, "listId")

	c1 := createContact(t, map[string]string{"email": "ar1@test.com"})
	c1ID := assertIsString(t, c1, "id")
	c2 := createContact(t, map[string]string{"email": "ar2@test.com"})
	c2ID := assertIsString(t, c2, "id")

	// Add c1 first.
	resp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/add", listID), []string{c1ID})
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Add c2, remove c1 in a single call.
	resp = doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/add-and-remove", listID), map[string]any{
		"recordIdsToAdd":    []string{c2ID},
		"recordIdsToRemove": []string{c1ID},
	})
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	added := assertIsArray(t, body, "recordIdsAdded")
	if len(added) != 1 {
		t.Errorf("expected 1 added, got %d", len(added))
	}
	removed := assertIsArray(t, body, "recordIdsRemoved")
	if len(removed) != 1 {
		t.Errorf("expected 1 removed, got %d", len(removed))
	}
}

func TestGetMemberships(t *testing.T) {
	resetServer(t)

	list := createList(t, "Get Memberships Test")
	listID := assertIsString(t, list, "listId")

	c1 := createContact(t, map[string]string{"email": "gm1@test.com"})
	c1ID := assertIsString(t, c1, "id")
	c2 := createContact(t, map[string]string{"email": "gm2@test.com"})
	c2ID := assertIsString(t, c2, "id")

	resp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/add", listID), []string{c1ID, c2ID})
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Get memberships — cursor pagination.
	resp = doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/lists/%s/memberships", listID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 2 {
		t.Errorf("expected 2 memberships, got %d", len(results))
	}

	// Validate each membership entry has required fields.
	for i, r := range results {
		m := toObject(t, r)
		assertIsString(t, m, "recordId")
		assertIsString(t, m, "listId")
		assertFieldPresent(t, m, "addedAt")
		if got := assertIsString(t, m, "listId"); got != listID {
			t.Errorf("membership[%d] listId: expected %s, got %s", i, listID, got)
		}
	}

	// Pagination with limit=1 should return paging.
	resp = doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/lists/%s/memberships?limit=1", listID), nil)
	mustStatus(t, resp, http.StatusOK)
	body = readJSON(t, resp)
	results = assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(results))
	}
	assertPaging(t, body)
}

func TestRemoveAllMembers(t *testing.T) {
	resetServer(t)

	list := createList(t, "Remove All Members Test")
	listID := assertIsString(t, list, "listId")

	c1 := createContact(t, map[string]string{"email": "rall1@test.com"})
	c1ID := assertIsString(t, c1, "id")
	c2 := createContact(t, map[string]string{"email": "rall2@test.com"})
	c2ID := assertIsString(t, c2, "id")

	resp := doRequest(t, http.MethodPut, fmt.Sprintf("/crm/v3/lists/%s/memberships/add", listID), []string{c1ID, c2ID})
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Remove all members.
	resp = doRequest(t, http.MethodDelete, fmt.Sprintf("/crm/v3/lists/%s/memberships", listID), nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// Verify memberships are empty.
	resp = doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/lists/%s/memberships", listID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if len(results) != 0 {
		t.Errorf("expected 0 memberships after remove all, got %d", len(results))
	}
}

func TestUniqueListNames(t *testing.T) {
	resetServer(t)

	createList(t, "Unique Name Test")

	// Creating a second list with the same name should fail with 409.
	resp := doRequest(t, http.MethodPost, "/crm/v3/lists", map[string]any{
		"name":           "Unique Name Test",
		"objectTypeId":   "0-1",
		"processingType": "MANUAL",
	})
	mustStatus(t, resp, http.StatusConflict)
	body := readJSON(t, resp)
	assertHubSpotError(t, body, "CONFLICT")
}

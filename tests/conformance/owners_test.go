package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestOwners_ListAll(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response")
	}

	for i, r := range results {
		owner := toObject(t, r)
		assertIsString(t, owner, "id")
		assertFieldPresent(t, owner, "email")
		assertFieldPresent(t, owner, "type")
		_ = i
	}
}

func TestOwners_ListEmpty(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response, even if empty")
	}
}

func TestOwners_GetByID(t *testing.T) {
	resetServer(t)

	// List owners first to get a valid ID.
	listResp := doRequest(t, http.MethodGet, "/crm/v3/owners/", nil)
	mustStatus(t, listResp, http.StatusOK)

	listBody := readJSON(t, listResp)
	results := assertIsArray(t, listBody, "results")
	if len(results) == 0 {
		t.Skip("no owners available to test GET by ID")
	}

	first := toObject(t, results[0])
	ownerID := assertIsString(t, first, "id")

	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/owners/%s", ownerID), nil)
	mustStatus(t, resp, http.StatusOK)

	owner := readJSON(t, resp)
	assertStringField(t, owner, "id", ownerID)
	assertFieldPresent(t, owner, "email")
	assertFieldPresent(t, owner, "type")
}

func TestOwners_GetNonExistent(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/999999999", nil)
	mustStatus(t, resp, http.StatusNotFound)
}

func TestOwners_ListWithPagination(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/?limit=1", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response")
	}

	if len(results) > 1 {
		t.Errorf("expected at most 1 result with limit=1, got %d", len(results))
	}

	// If there are more results, paging should be present.
	if len(results) == 1 {
		if _, ok := body["paging"]; ok {
			assertPaging(t, body)
		}
	}
}

func TestOwners_FilterByEmail(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/?email=nonexistent@test.com", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if results == nil {
		t.Fatal("expected results array in response")
	}

	// With a nonexistent email, we expect no results.
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent email filter, got %d", len(results))
	}
}

func TestOwners_ResponseStructure(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	results := assertIsArray(t, body, "results")
	if len(results) == 0 {
		t.Skip("no owners available to validate response structure")
	}

	for i, r := range results {
		owner := toObject(t, r)

		// Required fields per spec: id, type, archived, createdAt, updatedAt
		id := assertIsString(t, owner, "id")
		if id == "" {
			t.Errorf("owner[%d]: id should be non-empty", i)
		}

		ownerType := assertIsString(t, owner, "type")
		if ownerType != "PERSON" && ownerType != "QUEUE" {
			t.Errorf("owner[%d]: type should be PERSON or QUEUE, got %q", i, ownerType)
		}

		assertBoolField(t, owner, "archived", false)

		createdAt := assertIsString(t, owner, "createdAt")
		assertISOTimestamp(t, createdAt)

		updatedAt := assertIsString(t, owner, "updatedAt")
		assertISOTimestamp(t, updatedAt)

		// Optional but expected fields: email, firstName, lastName
		assertFieldPresent(t, owner, "email")
		assertFieldPresent(t, owner, "firstName")
		assertFieldPresent(t, owner, "lastName")

		// teams should be an array
		assertIsArray(t, owner, "teams")
	}
}

func TestOwners_ListArchived(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/owners/?archived=true", nil)
	mustStatus(t, resp, http.StatusOK)

	body := readJSON(t, resp)
	assertIsArray(t, body, "results")
}

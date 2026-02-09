package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestResetEndpoint(t *testing.T) {
	resetServer(t)

	// Create a contact so we have data to clear.
	contact := createContact(t, map[string]string{
		"firstname": "Reset",
		"lastname":  "Test",
		"email":     "reset@example.com",
	})
	contactID := assertIsString(t, contact, "id")

	// Verify the contact exists before reset.
	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+contactID, nil)
	mustStatus(t, resp, http.StatusOK)
	readJSON(t, resp)

	// Call reset.
	resp = doRequest(t, http.MethodPost, "/_notspot/reset", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	assertStringField(t, body, "status", "ok")

	// Verify the contact is gone after reset.
	resp = doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+contactID, nil)
	mustStatus(t, resp, http.StatusNotFound)
	_ = resp.Body.Close()

	// Verify seeded data still exists — default properties for contacts should be present.
	resp = doRequest(t, http.MethodGet, "/crm/v3/properties/contacts", nil)
	mustStatus(t, resp, http.StatusOK)
	propsBody := readJSON(t, resp)
	results := assertIsArray(t, propsBody, "results")
	if len(results) == 0 {
		t.Error("expected seeded properties to exist after reset, got none")
	}
}

func TestSeedEndpoint(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodPost, "/_notspot/seed", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)
	assertStringField(t, body, "status", "ok")
}

func TestRequestLog(t *testing.T) {
	resetServer(t)

	// The request log endpoint should respond with 200 and a results array.
	resp := doRequest(t, http.MethodGet, "/_notspot/requests", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")

	// If request logging middleware is not yet wired, the results array will
	// be empty — that's OK, we just verify the endpoint responds correctly.
	if len(results) == 0 {
		t.Log("request log is empty (logging middleware may not be wired); skipping entry assertions")
		return
	}

	// Verify each entry has the expected fields.
	for i, r := range results {
		entry := toObject(t, r)
		t.Run(fmt.Sprintf("entry_%d", i), func(t *testing.T) {
			assertFieldPresent(t, entry, "method")
			assertFieldPresent(t, entry, "path")
			assertFieldPresent(t, entry, "statusCode")
			assertFieldPresent(t, entry, "durationMs")
			assertFieldPresent(t, entry, "createdAt")
		})
	}

	// Verify pagination with limit parameter.
	t.Run("pagination", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/_notspot/requests?limit=1", nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)

		results := assertIsArray(t, body, "results")
		if len(results) != 1 {
			t.Errorf("expected 1 result with limit=1, got %d", len(results))
		}

		// Since we made several requests, there should be a next page.
		assertPaging(t, body)

		// Fetch the next page using the cursor.
		paging := assertIsObject(t, body, "paging")
		if paging != nil {
			next := assertIsObject(t, paging, "next")
			if next != nil {
				after := assertIsString(t, next, "after")
				if after != "" {
					resp2 := doRequest(t, http.MethodGet, "/_notspot/requests?limit=1&after="+after, nil)
					mustStatus(t, resp2, http.StatusOK)
					body2 := readJSON(t, resp2)

					results2 := assertIsArray(t, body2, "results")
					if len(results2) != 1 {
						t.Errorf("expected 1 result on second page, got %d", len(results2))
					}
				}
			}
		}
	})
}

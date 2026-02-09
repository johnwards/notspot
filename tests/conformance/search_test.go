package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

// searchContacts performs a POST /crm/v3/objects/contacts/search request.
func searchContacts(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/search", body)
	mustStatus(t, resp, http.StatusOK)
	return readJSON(t, resp)
}

// searchContactIDs performs a search and returns just the IDs from the results.
func searchContactIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	result := searchContacts(t, body)
	results := assertIsArray(t, result, "results")
	ids := make([]string, len(results))
	for i, r := range results {
		obj := toObject(t, r)
		ids[i] = assertIsString(t, obj, "id")
	}
	return ids
}

// filterBody builds a search request body with a single filter group containing one filter.
func filterBody(propertyName, operator, value string) map[string]any {
	f := map[string]any{
		"propertyName": propertyName,
		"operator":     operator,
	}
	if value != "" {
		f["value"] = value
	}
	return map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{f},
			},
		},
	}
}

// containsID checks if an ID is in the list.
func containsID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func TestSearchEQ(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice", "lastname": "Smith"})
	createContact(t, map[string]string{"firstname": "Bob", "lastname": "Jones"})

	ids := searchContactIDs(t, filterBody("firstname", "EQ", "Alice"))

	if len(ids) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ids))
	}
	if ids[0] != assertIsString(t, c1, "id") {
		t.Errorf("expected id %s, got %s", assertIsString(t, c1, "id"), ids[0])
	}
}

func TestSearchNEQ(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice", "lastname": "Smith"})
	c2 := createContact(t, map[string]string{"firstname": "Bob", "lastname": "Jones"})

	ids := searchContactIDs(t, filterBody("firstname", "NEQ", "Alice"))

	if !containsID(ids, assertIsString(t, c2, "id")) {
		t.Errorf("expected Bob (id=%s) in results", assertIsString(t, c2, "id"))
	}
	if containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("did not expect Alice (id=%s) in results", assertIsString(t, c1, "id"))
	}
}

func TestSearchLT(t *testing.T) {
	resetServer(t)

	// Use alphabetically-ordered string values (SQLite compares as strings).
	createContact(t, map[string]string{"firstname": "B", "lastname": "Beta"})
	c2 := createContact(t, map[string]string{"firstname": "A", "lastname": "Alpha"})
	createContact(t, map[string]string{"firstname": "C", "lastname": "Charlie"})

	ids := searchContactIDs(t, filterBody("lastname", "LT", "Beta"))

	if len(ids) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ids))
	}
	if ids[0] != assertIsString(t, c2, "id") {
		t.Errorf("expected id %s, got %s", assertIsString(t, c2, "id"), ids[0])
	}
}

func TestSearchLTE(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "B", "lastname": "Beta"})
	c2 := createContact(t, map[string]string{"firstname": "A", "lastname": "Alpha"})
	createContact(t, map[string]string{"firstname": "C", "lastname": "Charlie"})

	ids := searchContactIDs(t, filterBody("lastname", "LTE", "Beta"))

	if len(ids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected B (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c2, "id")) {
		t.Errorf("expected A (id=%s) in results", assertIsString(t, c2, "id"))
	}
}

func TestSearchGT(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "B", "lastname": "Beta"})
	createContact(t, map[string]string{"firstname": "A", "lastname": "Alpha"})
	c3 := createContact(t, map[string]string{"firstname": "C", "lastname": "Charlie"})

	ids := searchContactIDs(t, filterBody("lastname", "GT", "Beta"))

	if len(ids) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ids))
	}
	if ids[0] != assertIsString(t, c3, "id") {
		t.Errorf("expected id %s, got %s", assertIsString(t, c3, "id"), ids[0])
	}
}

func TestSearchGTE(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "B", "lastname": "Beta"})
	createContact(t, map[string]string{"firstname": "A", "lastname": "Alpha"})
	c3 := createContact(t, map[string]string{"firstname": "C", "lastname": "Charlie"})

	ids := searchContactIDs(t, filterBody("lastname", "GTE", "Beta"))

	if len(ids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected B (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("expected C (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

func TestSearchBETWEEN(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "A", "annualrevenue": "100"})
	createContact(t, map[string]string{"firstname": "B", "annualrevenue": "50"})
	c3 := createContact(t, map[string]string{"firstname": "C", "annualrevenue": "200"})
	createContact(t, map[string]string{"firstname": "D", "annualrevenue": "300"})

	body := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "annualrevenue",
						"operator":     "BETWEEN",
						"value":        "100",
						"highValue":    "200",
					},
				},
			},
		},
	}

	ids := searchContactIDs(t, body)

	if len(ids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected A (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("expected C (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

func TestSearchIN(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice"})
	createContact(t, map[string]string{"firstname": "Bob"})
	c3 := createContact(t, map[string]string{"firstname": "Charlie"})

	body := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "IN",
						"values":       []any{"Alice", "Charlie"},
					},
				},
			},
		},
	}

	ids := searchContactIDs(t, body)

	if len(ids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected Alice (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("expected Charlie (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

func TestSearchNOT_IN(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice"})
	c2 := createContact(t, map[string]string{"firstname": "Bob"})
	c3 := createContact(t, map[string]string{"firstname": "Charlie"})

	body := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "NOT_IN",
						"values":       []any{"Alice", "Charlie"},
					},
				},
			},
		},
	}

	ids := searchContactIDs(t, body)

	if !containsID(ids, assertIsString(t, c2, "id")) {
		t.Errorf("expected Bob (id=%s) in results", assertIsString(t, c2, "id"))
	}
	if containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("did not expect Alice (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("did not expect Charlie (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

func TestSearchHAS_PROPERTY(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice", "company": "Acme"})
	createContact(t, map[string]string{"firstname": "Bob"})

	ids := searchContactIDs(t, filterBody("company", "HAS_PROPERTY", ""))

	if len(ids) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ids))
	}
	if ids[0] != assertIsString(t, c1, "id") {
		t.Errorf("expected id %s, got %s", assertIsString(t, c1, "id"), ids[0])
	}
}

func TestSearchNOT_HAS_PROPERTY(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "Alice", "company": "Acme"})
	c2 := createContact(t, map[string]string{"firstname": "Bob"})

	ids := searchContactIDs(t, filterBody("company", "NOT_HAS_PROPERTY", ""))

	if !containsID(ids, assertIsString(t, c2, "id")) {
		t.Errorf("expected Bob (id=%s) in results", assertIsString(t, c2, "id"))
	}
}

func TestSearchCONTAINS_TOKEN(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alexander"})
	createContact(t, map[string]string{"firstname": "Bob"})
	c3 := createContact(t, map[string]string{"firstname": "Alexandra"})

	ids := searchContactIDs(t, filterBody("firstname", "CONTAINS_TOKEN", "Alex"))

	if len(ids) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected Alexander (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("expected Alexandra (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

func TestSearchNOT_CONTAINS_TOKEN(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "Alexander"})
	c2 := createContact(t, map[string]string{"firstname": "Bob"})
	createContact(t, map[string]string{"firstname": "Alexandra"})

	ids := searchContactIDs(t, filterBody("firstname", "NOT_CONTAINS_TOKEN", "Alex"))

	if !containsID(ids, assertIsString(t, c2, "id")) {
		t.Errorf("expected Bob (id=%s) in results", assertIsString(t, c2, "id"))
	}
	// Alexander and Alexandra should not be in results.
	if len(ids) != 1 {
		t.Errorf("expected 1 result (Bob only), got %d", len(ids))
	}
}

func TestSearchMultipleFilters(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice", "lastname": "Smith"})
	createContact(t, map[string]string{"firstname": "Alice", "lastname": "Jones"})
	createContact(t, map[string]string{"firstname": "Bob", "lastname": "Smith"})

	body := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "EQ",
						"value":        "Alice",
					},
					map[string]any{
						"propertyName": "lastname",
						"operator":     "EQ",
						"value":        "Smith",
					},
				},
			},
		},
	}

	ids := searchContactIDs(t, body)

	if len(ids) != 1 {
		t.Fatalf("expected 1 result (AND of filters), got %d", len(ids))
	}
	if ids[0] != assertIsString(t, c1, "id") {
		t.Errorf("expected Alice Smith (id=%s), got %s", assertIsString(t, c1, "id"), ids[0])
	}
}

func TestSearchMultipleFilterGroups(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice", "lastname": "Smith"})
	createContact(t, map[string]string{"firstname": "Bob", "lastname": "Jones"})
	c3 := createContact(t, map[string]string{"firstname": "Charlie", "lastname": "Brown"})

	body := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "EQ",
						"value":        "Alice",
					},
				},
			},
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "EQ",
						"value":        "Charlie",
					},
				},
			},
		},
	}

	ids := searchContactIDs(t, body)

	if len(ids) != 2 {
		t.Fatalf("expected 2 results (OR of groups), got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected Alice (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("expected Charlie (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

func TestSearchSorting(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "Charlie"})
	createContact(t, map[string]string{"firstname": "Alice"})
	createContact(t, map[string]string{"firstname": "Bob"})

	// Sort ascending by firstname, request properties to get them in response.
	body := map[string]any{
		"sorts": []any{
			map[string]any{
				"propertyName": "firstname",
				"direction":    "ASCENDING",
			},
		},
		"properties": []string{"firstname"},
	}

	result := searchContacts(t, body)
	results := assertIsArray(t, result, "results")

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	expected := []string{"Alice", "Bob", "Charlie"}
	for i, r := range results {
		obj := toObject(t, r)
		props := assertIsObject(t, obj, "properties")
		fn := assertIsString(t, props, "firstname")
		if fn != expected[i] {
			t.Errorf("result[%d]: expected firstname %q, got %q", i, expected[i], fn)
		}
	}

	// Sort descending.
	body["sorts"] = []any{
		map[string]any{
			"propertyName": "firstname",
			"direction":    "DESCENDING",
		},
	}

	result = searchContacts(t, body)
	results = assertIsArray(t, result, "results")

	expectedDesc := []string{"Charlie", "Bob", "Alice"}
	for i, r := range results {
		obj := toObject(t, r)
		props := assertIsObject(t, obj, "properties")
		fn := assertIsString(t, props, "firstname")
		if fn != expectedDesc[i] {
			t.Errorf("descending result[%d]: expected firstname %q, got %q", i, expectedDesc[i], fn)
		}
	}
}

func TestSearchPagination(t *testing.T) {
	resetServer(t)

	// Create 5 contacts.
	createdIDs := make([]string, 5)
	for i := range 5 {
		c := createContact(t, map[string]string{"firstname": fmt.Sprintf("Contact%d", i)})
		createdIDs[i] = assertIsString(t, c, "id")
	}

	// Page 1: limit 2.
	body := map[string]any{
		"limit": 2,
	}
	result := searchContacts(t, body)
	results := assertIsArray(t, result, "results")
	if len(results) != 2 {
		t.Fatalf("page 1: expected 2 results, got %d", len(results))
	}

	// Check total.
	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected total to be a number")
	}
	if int(total) != 5 {
		t.Errorf("expected total=5, got %d", int(total))
	}

	// Should have paging.
	assertPaging(t, result)
	paging := assertIsObject(t, result, "paging")
	next := assertIsObject(t, paging, "next")
	after := assertIsString(t, next, "after")
	if after == "" {
		t.Fatal("expected non-empty after cursor")
	}

	// Page 2.
	body["after"] = after
	result = searchContacts(t, body)
	results2 := assertIsArray(t, result, "results")
	if len(results2) != 2 {
		t.Fatalf("page 2: expected 2 results, got %d", len(results2))
	}

	// Page 2 results should be different from page 1.
	page1IDs := make(map[string]bool)
	for _, r := range results {
		obj := toObject(t, r)
		page1IDs[assertIsString(t, obj, "id")] = true
	}
	for _, r := range results2 {
		obj := toObject(t, r)
		id := assertIsString(t, obj, "id")
		if page1IDs[id] {
			t.Errorf("page 2 contains id %s which was already in page 1", id)
		}
	}

	// Page 3: should have 1 result and no paging.
	paging2 := assertIsObject(t, result, "paging")
	next2 := assertIsObject(t, paging2, "next")
	after2 := assertIsString(t, next2, "after")

	body["after"] = after2
	result = searchContacts(t, body)
	results3 := assertIsArray(t, result, "results")
	if len(results3) != 1 {
		t.Fatalf("page 3: expected 1 result, got %d", len(results3))
	}

	// No paging on last page.
	if _, hasPaging := result["paging"]; hasPaging {
		t.Error("expected no paging on the last page")
	}
}

func TestSearchProperties(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{
		"firstname": "Alice",
		"lastname":  "Smith",
		"email":     "alice@example.com",
		"company":   "Acme",
	})

	// Request only firstname and email.
	body := map[string]any{
		"properties": []any{"firstname", "email"},
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "EQ",
						"value":        "Alice",
					},
				},
			},
		},
	}

	result := searchContacts(t, body)
	results := assertIsArray(t, result, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	obj := toObject(t, results[0])
	props := assertIsObject(t, obj, "properties")

	// Requested properties should be present.
	assertFieldPresent(t, props, "firstname")
	assertFieldPresent(t, props, "email")

	// Default properties (hs_object_id, etc.) are always included.
	assertFieldPresent(t, props, "hs_object_id")

	// Non-requested, non-default properties should not be present.
	if _, hasCompany := props["company"]; hasCompany {
		t.Error("expected 'company' to NOT be in response when not requested")
	}
}

func TestSearchResponseShape(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "Alice"})

	body := map[string]any{
		"filterGroups": []any{
			map[string]any{
				"filters": []any{
					map[string]any{
						"propertyName": "firstname",
						"operator":     "EQ",
						"value":        "Alice",
					},
				},
			},
		},
	}

	result := searchContacts(t, body)

	// Must have "total" as a number.
	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected 'total' to be a number")
	}
	if int(total) != 1 {
		t.Errorf("expected total=1, got %d", int(total))
	}

	// Must have "results" as an array.
	results := assertIsArray(t, result, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Each result should be a SimplePublicObject.
	obj := toObject(t, results[0])
	assertSimplePublicObject(t, obj)

	// With only 1 result, there should be no paging.
	if _, hasPaging := result["paging"]; hasPaging {
		t.Error("expected no paging when all results fit in one page")
	}
}

func TestSearchQueryFullText(t *testing.T) {
	resetServer(t)

	c1 := createContact(t, map[string]string{"firstname": "Alice", "email": "alice@example.com"})
	createContact(t, map[string]string{"firstname": "Bob", "email": "bob@test.com"})
	c3 := createContact(t, map[string]string{"firstname": "Charlie", "email": "charlie@example.com"})

	// Search by query string — should match across searchable props.
	body := map[string]any{
		"query": "example.com",
	}

	ids := searchContactIDs(t, body)

	if len(ids) != 2 {
		t.Fatalf("expected 2 results for query 'example.com', got %d", len(ids))
	}
	if !containsID(ids, assertIsString(t, c1, "id")) {
		t.Errorf("expected Alice (id=%s) in results", assertIsString(t, c1, "id"))
	}
	if !containsID(ids, assertIsString(t, c3, "id")) {
		t.Errorf("expected Charlie (id=%s) in results", assertIsString(t, c3, "id"))
	}
}

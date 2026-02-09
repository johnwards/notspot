package conformance_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestEdge_EmptySearchResults(t *testing.T) {
	resetServer(t)

	result := searchContacts(t, filterBody("firstname", "EQ", "zzz_nonexistent_name_xyz"))

	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected 'total' to be a number")
	}
	if int(total) != 0 {
		t.Errorf("expected total=0, got %d", int(total))
	}

	results := assertIsArray(t, result, "results")
	if len(results) != 0 {
		t.Errorf("expected empty results array, got %d items", len(results))
	}
}

func TestEdge_SingleResultNoPaging(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "Solo"})

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=10", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if _, hasPaging := body["paging"]; hasPaging {
		t.Error("expected no paging field when only 1 result returned with limit=10")
	}
}

func TestEdge_ExactPageBoundary(t *testing.T) {
	resetServer(t)

	for i := range 10 {
		createContact(t, map[string]string{"firstname": fmt.Sprintf("Boundary%d", i)})
	}

	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=10", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}

	// At exact page boundary, HubSpot may or may not include a next cursor.
	// We just log the behavior for documentation purposes.
	if _, hasPaging := body["paging"]; hasPaging {
		t.Logf("paging IS present at exact page boundary (limit=10, results=10)")
	} else {
		t.Logf("paging is NOT present at exact page boundary (limit=10, results=10)")
	}
}

func TestEdge_PaginationCursorStability(t *testing.T) {
	resetServer(t)

	// Create initial 10 contacts.
	for i := range 10 {
		createContact(t, map[string]string{"firstname": fmt.Sprintf("Stable%d", i)})
	}

	// List with limit=3, save cursor.
	resp := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=3", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 3 {
		t.Fatalf("expected 3 results on first page, got %d", len(results))
	}

	paging := assertIsObject(t, body, "paging")
	if paging == nil {
		t.Fatal("expected paging on first page with limit=3 and 10 contacts")
	}
	next := assertIsObject(t, paging, "next")
	cursor := assertIsString(t, next, "after")
	if cursor == "" {
		t.Fatal("expected non-empty cursor")
	}

	// Create 5 more contacts while holding the cursor.
	for i := range 5 {
		createContact(t, map[string]string{"firstname": fmt.Sprintf("NewContact%d", i)})
	}

	// Use saved cursor to get next page.
	resp2 := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts?limit=3&after=%s", cursor), nil)
	mustStatus(t, resp2, http.StatusOK)
	body2 := readJSON(t, resp2)

	results2 := assertIsArray(t, body2, "results")
	if len(results2) == 0 {
		t.Fatal("expected results on second page after using saved cursor")
	}

	// Verify no overlap with first page.
	page1IDs := make(map[string]bool)
	for _, r := range results {
		obj := toObject(t, r)
		page1IDs[assertIsString(t, obj, "id")] = true
	}
	for _, r := range results2 {
		obj := toObject(t, r)
		id := assertIsString(t, obj, "id")
		if page1IDs[id] {
			t.Errorf("cursor page 2 contains id %s from page 1 — cursor not stable", id)
		}
	}
}

func TestEdge_SearchLargePageLimit(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "LargePageTest"})

	// limit=200 is the max for search — should succeed.
	body200 := map[string]any{"limit": 200}
	result := searchContacts(t, body200)
	assertIsArray(t, result, "results")

	// limit=201 — should either error or clamp to 200.
	body201 := map[string]any{"limit": 201}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/search", body201)
	if resp.StatusCode == http.StatusOK {
		respBody := readJSON(t, resp)
		assertIsArray(t, respBody, "results")
		t.Log("limit=201 was accepted (likely clamped to 200)")
	} else {
		respBody := readJSON(t, resp)
		t.Logf("limit=201 returned status %d: %v", resp.StatusCode, respBody)
	}
}

func TestEdge_ListLargePageLimit(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "ListPageTest"})

	// limit=100 is the max for list — should succeed.
	resp100 := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=100", nil)
	mustStatus(t, resp100, http.StatusOK)
	body100 := readJSON(t, resp100)
	assertIsArray(t, body100, "results")

	// limit=101 — should either error or clamp.
	resp101 := doRequest(t, http.MethodGet, "/crm/v3/objects/contacts?limit=101", nil)
	if resp101.StatusCode == http.StatusOK {
		body101 := readJSON(t, resp101)
		assertIsArray(t, body101, "results")
		t.Log("limit=101 for list was accepted (likely clamped to 100)")
	} else {
		body101 := readJSON(t, resp101)
		t.Logf("limit=101 for list returned status %d: %v", resp101.StatusCode, body101)
	}
}

func TestEdge_PropertiesSelectionNonExistent(t *testing.T) {
	resetServer(t)

	c := createContact(t, map[string]string{"firstname": "PropTest"})
	id := assertIsString(t, c, "id")

	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts/%s?properties=firstname,nonexistent_prop", id), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	props := assertIsObject(t, body, "properties")
	assertStringField(t, props, "firstname", "PropTest")

	// nonexistent_prop should be either absent or null.
	if val, exists := props["nonexistent_prop"]; exists {
		if val != nil {
			t.Errorf("expected nonexistent_prop to be nil or absent, got %v", val)
		}
	}
	// If absent, that's also acceptable — no error needed.
}

func TestEdge_NullVsEmptyVsMissingProperties(t *testing.T) {
	resetServer(t)

	// Create contact without setting firstname — only set email.
	c := createContact(t, map[string]string{"email": "noname@example.com"})
	id := assertIsString(t, c, "id")

	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts/%s?properties=firstname,email", id), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	props := assertIsObject(t, body, "properties")
	assertStringField(t, props, "email", "noname@example.com")

	// firstname was never set — check what value comes back.
	if val, exists := props["firstname"]; exists {
		if val == nil {
			t.Log("unset firstname returned as null")
		} else if s, ok := val.(string); ok && s == "" {
			t.Log("unset firstname returned as empty string")
		} else {
			t.Logf("unset firstname returned as: %v (%T)", val, val)
		}
	} else {
		t.Log("unset firstname is absent from response")
	}
}

func TestEdge_ArchivedObjectRetrieval(t *testing.T) {
	resetServer(t)

	c := createContact(t, map[string]string{"firstname": "ToArchive"})
	id := assertIsString(t, c, "id")

	// Delete (archive) the contact.
	delResp := doRequest(t, http.MethodDelete, fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil)
	if delResp.StatusCode != http.StatusNoContent && delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 204 or 200 for delete, got %d", delResp.StatusCode)
	}
	_ = delResp.Body.Close()

	// GET archived object without ?archived=true — should return 404 or archived:true.
	getResp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil)
	switch getResp.StatusCode {
	case http.StatusNotFound:
		_ = getResp.Body.Close()
		t.Log("archived object returns 404 without ?archived=true")
	case http.StatusOK:
		body := readJSON(t, getResp)
		if archived, ok := body["archived"].(bool); ok && archived {
			t.Log("archived object returns 200 with archived:true")
		} else {
			t.Error("archived object returned 200 but archived field is not true")
		}
	default:
		_ = getResp.Body.Close()
		t.Errorf("unexpected status %d for archived object GET", getResp.StatusCode)
	}

	// GET with ?archived=true — should return the archived object.
	getArchResp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts/%s?archived=true", id), nil)
	if getArchResp.StatusCode == http.StatusOK {
		body := readJSON(t, getArchResp)
		assertBoolField(t, body, "archived", true)
		t.Log("GET with ?archived=true returns archived object with archived:true")
	} else {
		_ = getArchResp.Body.Close()
		t.Logf("GET with ?archived=true returned status %d", getArchResp.StatusCode)
	}
}

func TestEdge_DoubleArchive(t *testing.T) {
	resetServer(t)

	c := createContact(t, map[string]string{"firstname": "DoubleDelete"})
	id := assertIsString(t, c, "id")

	// First delete.
	resp1 := doRequest(t, http.MethodDelete, fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil)
	if resp1.StatusCode != http.StatusNoContent && resp1.StatusCode != http.StatusOK {
		t.Fatalf("first delete: expected 204 or 200, got %d", resp1.StatusCode)
	}
	_ = resp1.Body.Close()

	// Second delete — check behavior.
	resp2 := doRequest(t, http.MethodDelete, fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil)
	_ = resp2.Body.Close()
	switch resp2.StatusCode {
	case http.StatusNoContent:
		t.Log("double archive returns 204 (idempotent)")
	case http.StatusNotFound:
		t.Log("double archive returns 404")
	default:
		t.Logf("double archive returns status %d", resp2.StatusCode)
	}
}

func TestEdge_SpecialCharactersInProperties(t *testing.T) {
	resetServer(t)

	tests := []struct {
		name  string
		value string
	}{
		{"unicode", "Ñoño"},
		{"emoji", "John 🚀"},
		{"long_string", strings.Repeat("a", 500)},
		{"html", "<b>bold</b>"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := createContact(t, map[string]string{"firstname": tc.value})
			id := assertIsString(t, c, "id")

			resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts/%s?properties=firstname", id), nil)
			mustStatus(t, resp, http.StatusOK)
			body := readJSON(t, resp)

			props := assertIsObject(t, body, "properties")
			assertStringField(t, props, "firstname", tc.value)
		})
	}
}

func TestEdge_CaseSensitiveSearch(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "alice"})

	// Search for different cases.
	cases := []string{"ALICE", "Alice", "alice"}
	for _, query := range cases {
		t.Run(query, func(t *testing.T) {
			ids := searchContactIDs(t, filterBody("firstname", "EQ", query))
			switch len(ids) {
			case 1:
				t.Logf("EQ %q matched lowercase 'alice'", query)
			case 0:
				t.Logf("EQ %q did NOT match lowercase 'alice'", query)
			default:
				t.Logf("EQ %q returned %d results", query, len(ids))
			}
		})
	}
}

func TestEdge_BatchCreateDuplicateEmails(t *testing.T) {
	resetServer(t)

	body := map[string]any{
		"inputs": []any{
			map[string]any{
				"properties": map[string]string{
					"firstname": "Dup1",
					"email":     "duplicate@example.com",
				},
			},
			map[string]any{
				"properties": map[string]string{
					"firstname": "Dup2",
					"email":     "duplicate@example.com",
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", body)
	respBody := readJSON(t, resp)

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		results := assertIsArray(t, respBody, "results")
		t.Logf("batch create with duplicate emails succeeded with %d results", len(results))
	case http.StatusMultiStatus:
		t.Log("batch create with duplicate emails returned 207 (multi-status)")
	case http.StatusConflict:
		t.Log("batch create with duplicate emails returned 409 (conflict)")
	default:
		t.Logf("batch create with duplicate emails returned status %d", resp.StatusCode)
	}
}

func TestEdge_AssociationPagination(t *testing.T) {
	resetServer(t)

	company := createCompany(t, map[string]string{"name": "AssocPaginationCo"})
	companyID := assertIsString(t, company, "id")

	// Create 15 contacts and associate each with the company.
	contactIDs := make([]string, 15)
	for i := range 15 {
		c := createContact(t, map[string]string{"firstname": fmt.Sprintf("AssocContact%d", i)})
		cID := assertIsString(t, c, "id")
		contactIDs[i] = cID

		// Create association: company -> contact.
		assocBody := []any{
			map[string]any{
				"associationCategory": "HUBSPOT_DEFINED",
				"associationTypeId":   2,
			},
		}
		assocResp := doRequest(t, http.MethodPut,
			fmt.Sprintf("/crm/v4/objects/companies/%s/associations/contacts/%s", companyID, cID),
			assocBody)
		if assocResp.StatusCode != http.StatusOK && assocResp.StatusCode != http.StatusCreated {
			t.Fatalf("associate contact %d: status=%d", i, assocResp.StatusCode)
		}
		_ = assocResp.Body.Close()
	}

	// Paginate through associations with limit=5.
	allAssocIDs := make(map[string]bool)
	cursor := ""
	pages := 0
	for {
		pages++
		if pages > 10 {
			t.Fatal("too many pages — possible infinite loop")
		}

		path := fmt.Sprintf("/crm/v4/objects/companies/%s/associations/contacts?limit=5", companyID)
		if cursor != "" {
			path += "&after=" + cursor
		}

		resp := doRequest(t, http.MethodGet, path, nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)

		results := assertIsArray(t, body, "results")
		for _, r := range results {
			obj := toObject(t, r)
			toID := assertIsString(t, obj, "toObjectId")
			allAssocIDs[toID] = true
		}

		paging, hasPaging := body["paging"]
		if !hasPaging {
			break
		}
		pagingObj, ok := paging.(map[string]any)
		if !ok {
			break
		}
		nextObj, hasNext := pagingObj["next"]
		if !hasNext {
			break
		}
		nextMap, ok := nextObj.(map[string]any)
		if !ok {
			break
		}
		after, ok := nextMap["after"].(string)
		if !ok || after == "" {
			break
		}
		cursor = after
	}

	if len(allAssocIDs) != 15 {
		t.Errorf("expected 15 associated contacts, found %d across %d pages", len(allAssocIDs), pages)
	}

	// Verify all created contact IDs are found.
	for _, cID := range contactIDs {
		if !allAssocIDs[cID] {
			t.Errorf("contact %s not found in paginated associations", cID)
		}
	}
}

func TestEdge_SearchSortStability(t *testing.T) {
	resetServer(t)

	firstnames := []string{"Eve", "Dave", "Charlie", "Bob", "Alice"}
	for _, fn := range firstnames {
		createContact(t, map[string]string{"firstname": fn, "lastname": "Smith"})
	}

	body := map[string]any{
		"sorts": []any{
			map[string]any{
				"propertyName": "lastname",
				"direction":    "ASCENDING",
			},
		},
		"properties": []string{"firstname", "lastname"},
	}

	result := searchContacts(t, body)
	results := assertIsArray(t, result, "results")

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	// All should have lastname "Smith".
	for i, r := range results {
		obj := toObject(t, r)
		props := assertIsObject(t, obj, "properties")
		ln := assertIsString(t, props, "lastname")
		if ln != "Smith" {
			t.Errorf("result[%d]: expected lastname 'Smith', got %q", i, ln)
		}
	}

	// Run the same query again and verify order is deterministic.
	result2 := searchContacts(t, body)
	results2 := assertIsArray(t, result2, "results")

	if len(results2) != 5 {
		t.Fatalf("second query: expected 5 results, got %d", len(results2))
	}

	for i := range results {
		obj1 := toObject(t, results[i])
		obj2 := toObject(t, results2[i])
		id1 := assertIsString(t, obj1, "id")
		id2 := assertIsString(t, obj2, "id")
		if id1 != id2 {
			t.Errorf("result[%d]: first query id=%s, second query id=%s — sort not stable", i, id1, id2)
		}
	}
}

func TestEdge_EmptyBatchCreate(t *testing.T) {
	resetServer(t)

	body := map[string]any{
		"inputs": []any{},
	}

	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", body)
	respBody := readJSON(t, resp)

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		results := assertIsArray(t, respBody, "results")
		if len(results) != 0 {
			t.Errorf("expected 0 results for empty batch, got %d", len(results))
		}
		t.Log("empty batch create returns success with empty results")
	case http.StatusBadRequest:
		t.Log("empty batch create returns 400 (bad request)")
	default:
		t.Logf("empty batch create returns status %d", resp.StatusCode)
	}
}

func TestEdge_GetObjectWithAllProperties(t *testing.T) {
	resetServer(t)

	c := createContact(t, map[string]string{
		"firstname": "AllProps",
		"lastname":  "Test",
		"email":     "allprops@example.com",
		"company":   "TestCo",
	})
	id := assertIsString(t, c, "id")

	// GET without specifying properties — should return default properties.
	resp := doRequest(t, http.MethodGet, fmt.Sprintf("/crm/v3/objects/contacts/%s", id), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	props := assertIsObject(t, body, "properties")

	// hs_object_id should always be present as a default property.
	assertFieldPresent(t, props, "hs_object_id")

	// Log which properties are returned by default.
	t.Logf("default properties returned: %v", mapKeys(props))
}

func TestEdge_SearchWithEmptyQuery(t *testing.T) {
	resetServer(t)

	createContact(t, map[string]string{"firstname": "Empty1"})
	createContact(t, map[string]string{"firstname": "Empty2"})
	createContact(t, map[string]string{"firstname": "Empty3"})

	// Search with empty body — should return all contacts.
	result := searchContacts(t, map[string]any{})

	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected 'total' to be a number")
	}
	if int(total) < 3 {
		t.Errorf("expected total >= 3, got %d", int(total))
	}

	results := assertIsArray(t, result, "results")
	if len(results) < 3 {
		t.Errorf("expected at least 3 results, got %d", len(results))
	}
}

func TestEdge_PaginationWalkAll(t *testing.T) {
	resetServer(t)

	// Create 12 contacts.
	createdIDs := make(map[string]bool)
	for i := range 12 {
		c := createContact(t, map[string]string{"firstname": fmt.Sprintf("Walk%d", i)})
		createdIDs[assertIsString(t, c, "id")] = true
	}

	// Paginate with limit=5, collecting all IDs.
	allIDs := make(map[string]bool)
	cursor := ""
	pages := 0
	for {
		pages++
		if pages > 10 {
			t.Fatal("too many pages — possible infinite loop")
		}

		path := "/crm/v3/objects/contacts?limit=5"
		if cursor != "" {
			path += "&after=" + cursor
		}

		resp := doRequest(t, http.MethodGet, path, nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)

		results := assertIsArray(t, body, "results")
		for _, r := range results {
			obj := toObject(t, r)
			id := assertIsString(t, obj, "id")
			if allIDs[id] {
				t.Errorf("duplicate id %s found across pages", id)
			}
			allIDs[id] = true
		}

		paging, hasPaging := body["paging"]
		if !hasPaging {
			break
		}
		pagingObj, ok := paging.(map[string]any)
		if !ok {
			break
		}
		nextObj, hasNext := pagingObj["next"]
		if !hasNext {
			break
		}
		nextMap, ok := nextObj.(map[string]any)
		if !ok {
			break
		}
		after, ok := nextMap["after"].(string)
		if !ok || after == "" {
			break
		}
		cursor = after
	}

	if len(allIDs) != 12 {
		t.Errorf("expected 12 unique IDs across all pages, got %d (in %d pages)", len(allIDs), pages)
	}

	// Verify all created IDs were found.
	for id := range createdIDs {
		if !allIDs[id] {
			t.Errorf("created contact %s not found in paginated results", id)
		}
	}
}

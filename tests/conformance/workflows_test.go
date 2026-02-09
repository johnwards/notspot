package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

// TestWorkflow_CreateSearchVerify creates contacts and searches for one by firstname.
func TestWorkflow_CreateSearchVerify(t *testing.T) {
	resetServer(t)

	// Create 3 contacts with distinct firstnames.
	c1 := createContact(t, map[string]string{"firstname": "Zara", "lastname": "Alpha"})
	createContact(t, map[string]string{"firstname": "Yuki", "lastname": "Beta"})
	createContact(t, map[string]string{"firstname": "Xander", "lastname": "Gamma"})

	c1ID := assertIsString(t, c1, "id")

	// Search by firstname EQ for "Zara".
	result := searchContacts(t, filterBody("firstname", "EQ", "Zara"))
	results := assertIsArray(t, result, "results")

	if len(results) != 1 {
		t.Fatalf("expected exactly 1 search result, got %d", len(results))
	}

	obj := toObject(t, results[0])
	assertSimplePublicObject(t, obj)
	assertStringField(t, obj, "id", c1ID)

	props := assertIsObject(t, obj, "properties")
	assertStringField(t, props, "firstname", "Zara")
	assertStringField(t, props, "lastname", "Alpha")

	// Verify total count.
	total, ok := result["total"].(float64)
	if !ok {
		t.Fatal("expected 'total' to be a number")
	}
	if int(total) != 1 {
		t.Errorf("expected total=1, got %d", int(total))
	}
}

// TestWorkflow_CreateAssociateQuery creates objects and verifies bidirectional associations.
func TestWorkflow_CreateAssociateQuery(t *testing.T) {
	resetServer(t)

	// Create a company and 3 contacts.
	company := createCompany(t, map[string]string{"name": "Workflow Corp"})
	companyID := assertIsString(t, company, "id")

	contactIDs := make([]string, 3)
	for i := range 3 {
		c := createContact(t, map[string]string{
			"firstname": fmt.Sprintf("Assoc%d", i),
			"email":     fmt.Sprintf("assoc%d@workflow.com", i),
		})
		contactIDs[i] = assertIsString(t, c, "id")
	}

	// Associate each contact to the company using PUT with typed association body.
	for _, cID := range contactIDs {
		resp := doRequest(t, http.MethodPut,
			fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies/%s", cID, companyID),
			[]map[string]any{
				{"associationCategory": "HUBSPOT_DEFINED", "associationTypeId": 1},
			})
		mustStatus(t, resp, http.StatusOK)
		_ = resp.Body.Close()
	}

	// Query associations from contact→company for each contact.
	for _, cID := range contactIDs {
		resp := doRequest(t, http.MethodGet,
			fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", cID), nil)
		mustStatus(t, resp, http.StatusOK)
		body := readJSON(t, resp)

		results := assertIsArray(t, body, "results")
		if len(results) != 1 {
			t.Fatalf("contact %s: expected 1 association to company, got %d", cID, len(results))
		}
		first := toObject(t, results[0])
		assertStringField(t, first, "toObjectId", companyID)
	}

	// Query associations from company→contacts (reverse direction).
	resp := doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/companies/%s/associations/contacts", companyID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 3 {
		t.Fatalf("company: expected 3 associations to contacts, got %d", len(results))
	}

	// Verify all 3 contact IDs are present in the results.
	foundIDs := make(map[string]bool)
	for _, r := range results {
		obj := toObject(t, r)
		toID := assertIsString(t, obj, "toObjectId")
		foundIDs[toID] = true
	}
	for _, cID := range contactIDs {
		if !foundIDs[cID] {
			t.Errorf("expected contact %s in company→contacts associations", cID)
		}
	}
}

// TestWorkflow_CustomPropertyLifecycle tests creating, using, searching, and updating a custom property.
func TestWorkflow_CustomPropertyLifecycle(t *testing.T) {
	resetServer(t)

	// Create a custom property on contacts.
	propInput := map[string]any{
		"name":      "favorite_color",
		"label":     "Favorite Color",
		"type":      "string",
		"fieldType": "text",
		"groupName": "contactinformation",
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", propInput)
	mustStatus(t, resp, http.StatusCreated)
	propBody := readJSON(t, resp)
	assertStringField(t, propBody, "name", "favorite_color")

	// Create a contact using that custom property.
	c1 := createContact(t, map[string]string{
		"firstname":      "ColorFan",
		"favorite_color": "blue",
	})
	c1ID := assertIsString(t, c1, "id")

	// Create another contact with a different value for contrast.
	createContact(t, map[string]string{
		"firstname":      "OtherFan",
		"favorite_color": "red",
	})

	// Search by the custom property.
	ids := searchContactIDs(t, filterBody("favorite_color", "EQ", "blue"))
	if len(ids) != 1 {
		t.Fatalf("expected 1 result searching favorite_color=blue, got %d", len(ids))
	}
	if ids[0] != c1ID {
		t.Errorf("expected contact %s, got %s", c1ID, ids[0])
	}

	// Update the contact's custom property value.
	updateBody := map[string]any{
		"properties": map[string]string{"favorite_color": "green"},
	}
	resp = doRequest(t, http.MethodPatch, "/crm/v3/objects/contacts/"+c1ID, updateBody)
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Search for the updated value.
	ids = searchContactIDs(t, filterBody("favorite_color", "EQ", "green"))
	if len(ids) != 1 {
		t.Fatalf("expected 1 result searching favorite_color=green, got %d", len(ids))
	}
	if ids[0] != c1ID {
		t.Errorf("expected contact %s, got %s", c1ID, ids[0])
	}

	// Old value should no longer match.
	ids = searchContactIDs(t, filterBody("favorite_color", "EQ", "blue"))
	if len(ids) != 0 {
		t.Errorf("expected 0 results searching favorite_color=blue after update, got %d", len(ids))
	}
}

// TestWorkflow_PipelineWorkflow tests creating a deal in a pipeline stage and moving it.
func TestWorkflow_PipelineWorkflow(t *testing.T) {
	resetServer(t)

	// Get the default deals pipeline ID and its stages.
	pipelineID := getDefaultDealsPipelineID(t)

	resp := doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v3/pipelines/deals/%s/stages", pipelineID), nil)
	mustStatus(t, resp, http.StatusOK)
	stagesBody := readJSON(t, resp)

	stages := assertIsArray(t, stagesBody, "results")
	if len(stages) < 2 {
		t.Fatalf("expected at least 2 stages, got %d", len(stages))
	}

	// Get the first and second stage IDs.
	stage0 := toObject(t, stages[0])
	stage0ID := assertIsString(t, stage0, "id")
	stage1 := toObject(t, stages[1])
	stage1ID := assertIsString(t, stage1, "id")

	// Create a deal in the first stage.
	dealBody := map[string]any{
		"properties": map[string]string{
			"dealname":  "Pipeline Test Deal",
			"dealstage": stage0ID,
			"pipeline":  pipelineID,
		},
	}
	resp = doRequest(t, http.MethodPost, "/crm/v3/objects/deals", dealBody)
	mustStatus(t, resp, http.StatusCreated)
	deal := readJSON(t, resp)
	dealID := assertIsString(t, deal, "id")

	// Verify the deal is in the first stage.
	resp = doRequest(t, http.MethodGet, "/crm/v3/objects/deals/"+dealID+"?properties=dealstage", nil)
	mustStatus(t, resp, http.StatusOK)
	dealGet := readJSON(t, resp)
	props := assertIsObject(t, dealGet, "properties")
	assertStringField(t, props, "dealstage", stage0ID)

	// Update the deal to move to the next stage.
	updateBody := map[string]any{
		"properties": map[string]string{"dealstage": stage1ID},
	}
	resp = doRequest(t, http.MethodPatch, "/crm/v3/objects/deals/"+dealID, updateBody)
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// GET the deal and verify the stage property changed.
	resp = doRequest(t, http.MethodGet, "/crm/v3/objects/deals/"+dealID+"?properties=dealstage", nil)
	mustStatus(t, resp, http.StatusOK)
	dealGet2 := readJSON(t, resp)
	props2 := assertIsObject(t, dealGet2, "properties")
	assertStringField(t, props2, "dealstage", stage1ID)
}

// TestWorkflow_BulkLoadPaginateSearch batch creates contacts, paginates through all, then searches.
func TestWorkflow_BulkLoadPaginateSearch(t *testing.T) {
	resetServer(t)

	// Batch create 25 contacts.
	inputs := make([]map[string]any, 25)
	for i := range 25 {
		inputs[i] = map[string]any{
			"properties": map[string]string{
				"firstname": fmt.Sprintf("Bulk%02d", i),
				"email":     fmt.Sprintf("bulk%02d@workflow.com", i),
			},
		}
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/batch/create", map[string]any{
		"inputs": inputs,
	})
	mustStatus(t, resp, http.StatusCreated)
	batchResult := readJSON(t, resp)
	assertStringField(t, batchResult, "status", "COMPLETE")

	batchResults := assertIsArray(t, batchResult, "results")
	if len(batchResults) != 25 {
		t.Fatalf("expected 25 batch results, got %d", len(batchResults))
	}

	// Collect all created IDs.
	createdIDs := make(map[string]bool)
	for _, r := range batchResults {
		obj := toObject(t, r)
		id := assertIsString(t, obj, "id")
		createdIDs[id] = true
	}

	// List contacts with limit=5, walk all pages collecting IDs.
	collectedIDs := make(map[string]bool)
	path := "/crm/v3/objects/contacts?limit=5"
	pages := 0
	maxPages := 20 // safety limit

	for pages < maxPages {
		resp = doRequest(t, http.MethodGet, path, nil)
		mustStatus(t, resp, http.StatusOK)
		page := readJSON(t, resp)

		results := assertIsArray(t, page, "results")
		for _, r := range results {
			obj := toObject(t, r)
			id := assertIsString(t, obj, "id")
			if collectedIDs[id] {
				t.Errorf("duplicate ID %s found during pagination", id)
			}
			collectedIDs[id] = true
		}
		pages++

		// Check for paging.next.after cursor.
		paging, hasPaging := page["paging"]
		if !hasPaging {
			break
		}
		pagingObj, ok := paging.(map[string]any)
		if !ok {
			break
		}
		nextObj, ok := pagingObj["next"].(map[string]any)
		if !ok {
			break
		}
		after, ok := nextObj["after"].(string)
		if !ok || after == "" {
			break
		}
		path = fmt.Sprintf("/crm/v3/objects/contacts?limit=5&after=%s", after)
	}

	// Verify all 25 IDs are collected.
	for id := range createdIDs {
		if !collectedIDs[id] {
			t.Errorf("created contact %s not found during pagination", id)
		}
	}
	if len(collectedIDs) < 25 {
		t.Errorf("expected at least 25 contacts collected, got %d", len(collectedIDs))
	}

	// Search with a filter to find a subset (contacts with firstname starting with "Bulk0").
	searchResult := searchContacts(t, filterBody("firstname", "CONTAINS_TOKEN", "Bulk0"))
	searchResults := assertIsArray(t, searchResult, "results")
	if len(searchResults) < 1 {
		t.Error("expected at least 1 result searching for 'Bulk0'")
	}
}

// TestWorkflow_SchemaObjectsAssociations creates a custom schema, objects, and associations.
func TestWorkflow_SchemaObjectsAssociations(t *testing.T) {
	resetServer(t)

	// Create a custom object schema with a property and associated object.
	schemaInput := map[string]any{
		"name":   "vehicles",
		"labels": map[string]any{"singular": "Vehicle", "plural": "Vehicles"},
		"properties": []map[string]any{
			{
				"name":      "make",
				"label":     "Make",
				"type":      "string",
				"fieldType": "text",
			},
		},
		"associatedObjects":      []string{"contacts"},
		"primaryDisplayProperty": "make",
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/schemas", schemaInput)
	mustStatus(t, resp, http.StatusCreated)
	schema := readJSON(t, resp)
	assertStringField(t, schema, "name", "vehicles")

	// Create an object of the custom type.
	resp = doRequest(t, http.MethodPost, "/crm/v3/objects/vehicles", map[string]any{
		"properties": map[string]string{"make": "Tesla"},
	})
	mustStatus(t, resp, http.StatusCreated)
	vehicle := readJSON(t, resp)
	vehicleID := assertIsString(t, vehicle, "id")
	assertSimplePublicObject(t, vehicle)

	// Create a contact to associate with.
	contact := createContact(t, map[string]string{
		"firstname": "Driver",
		"email":     "driver@workflow.com",
	})
	contactID := assertIsString(t, contact, "id")

	// Associate the vehicle with the contact using default association.
	resp = doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/vehicles/%s/associations/default/contacts/%s", vehicleID, contactID), nil)
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Query the association from vehicle→contacts.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/vehicles/%s/associations/contacts", vehicleID), nil)
	mustStatus(t, resp, http.StatusOK)
	assocBody := readJSON(t, resp)

	results := assertIsArray(t, assocBody, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 association, got %d", len(results))
	}
	first := toObject(t, results[0])
	assertStringField(t, first, "toObjectId", contactID)
}

// TestWorkflow_MergeVerification tests merging two contacts and verifying property consolidation.
func TestWorkflow_MergeVerification(t *testing.T) {
	resetServer(t)

	// Create two contacts with different properties.
	primary := createContact(t, map[string]string{
		"firstname": "PrimaryFirst",
		"email":     "primary@merge.com",
	})
	primaryID := assertIsString(t, primary, "id")

	secondary := createContact(t, map[string]string{
		"lastname": "SecondaryLast",
		"email":    "secondary@merge.com",
	})
	secondaryID := assertIsString(t, secondary, "id")

	// Merge: secondary into primary.
	mergeBody := map[string]any{
		"primaryObjectId": primaryID,
		"objectIdToMerge": secondaryID,
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts/merge", mergeBody)
	mustStatus(t, resp, http.StatusOK)
	mergeResult := readJSON(t, resp)

	assertSimplePublicObject(t, mergeResult)
	assertStringField(t, mergeResult, "id", primaryID)

	// GET the surviving contact — verify it has properties from both.
	resp = doRequest(t, http.MethodGet,
		"/crm/v3/objects/contacts/"+primaryID+"?properties=firstname,lastname,email", nil)
	mustStatus(t, resp, http.StatusOK)
	survivor := readJSON(t, resp)

	survivorProps := assertIsObject(t, survivor, "properties")
	assertStringField(t, survivorProps, "firstname", "PrimaryFirst")
	// The secondary's lastname should have been merged into the primary.
	assertStringField(t, survivorProps, "lastname", "SecondaryLast")

	// GET the merged contact — expect archived.
	resp = doRequest(t, http.MethodGet, "/crm/v3/objects/contacts/"+secondaryID, nil)
	mustStatus(t, resp, http.StatusOK)
	mergedObj := readJSON(t, resp)
	assertBoolField(t, mergedObj, "archived", true)

	// Search — verify merged contact doesn't appear in search results.
	ids := searchContactIDs(t, filterBody("email", "EQ", "secondary@merge.com"))
	if containsID(ids, secondaryID) {
		t.Error("merged (archived) contact should not appear in search results")
	}
}

// TestWorkflow_ArchiveCascade tests archiving a contact and checking association behavior.
func TestWorkflow_ArchiveCascade(t *testing.T) {
	resetServer(t)

	// Create contact and company.
	contact := createContact(t, map[string]string{
		"firstname": "Archive",
		"email":     "archive@cascade.com",
	})
	contactID := assertIsString(t, contact, "id")

	company := createCompany(t, map[string]string{"name": "Cascade Corp"})
	companyID := assertIsString(t, company, "id")

	// Associate them.
	resp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/default/companies/%s", contactID, companyID), nil)
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Verify association exists before archive.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", contactID), nil)
	mustStatus(t, resp, http.StatusOK)
	beforeBody := readJSON(t, resp)
	beforeResults := assertIsArray(t, beforeBody, "results")
	if len(beforeResults) != 1 {
		t.Fatalf("expected 1 association before archive, got %d", len(beforeResults))
	}

	// Archive the contact.
	resp = doRequest(t, http.MethodDelete, "/crm/v3/objects/contacts/"+contactID, nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// Try to GET associations for the archived contact.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", contactID), nil)
	// The server may return 200 with empty results, or it may still show associations.
	// We record the behavior either way.
	if resp.StatusCode == http.StatusOK {
		body := readJSON(t, resp)
		results := assertIsArray(t, body, "results")
		// After archiving, associations should ideally be cleaned up.
		if len(results) != 0 {
			t.Logf("archived contact still has %d associations (may be expected)", len(results))
		}
	} else {
		_ = resp.Body.Close()
	}

	// Try to GET associations from the company side.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/companies/%s/associations/contacts", companyID), nil)
	mustStatus(t, resp, http.StatusOK)
	companyAssocBody := readJSON(t, resp)
	companyResults := assertIsArray(t, companyAssocBody, "results")

	// After archiving the contact, the company should no longer have an association to it.
	for _, r := range companyResults {
		obj := toObject(t, r)
		toID := assertIsString(t, obj, "toObjectId")
		if toID == contactID {
			t.Error("archived contact should not appear in company's associations")
		}
	}
}

// TestWorkflow_ListMembershipWorkflow tests the full list membership add/remove lifecycle.
func TestWorkflow_ListMembershipWorkflow(t *testing.T) {
	resetServer(t)

	// Create a list.
	list := createList(t, "Workflow Members List")
	listID := assertIsString(t, list, "listId")

	// Create 5 contacts.
	contactIDs := make([]string, 5)
	for i := range 5 {
		c := createContact(t, map[string]string{
			"email": fmt.Sprintf("member%d@workflow.com", i),
		})
		contactIDs[i] = assertIsString(t, c, "id")
	}

	// Add all 5 as members.
	resp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v3/lists/%s/memberships/add", listID), contactIDs)
	mustStatus(t, resp, http.StatusOK)
	addBody := readJSON(t, resp)

	added := assertIsArray(t, addBody, "recordIdsAdded")
	if len(added) != 5 {
		t.Errorf("expected 5 added, got %d", len(added))
	}

	// GET memberships — verify all 5 are present.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v3/lists/%s/memberships", listID), nil)
	mustStatus(t, resp, http.StatusOK)
	membersBody := readJSON(t, resp)
	memberResults := assertIsArray(t, membersBody, "results")
	if len(memberResults) != 5 {
		t.Fatalf("expected 5 memberships, got %d", len(memberResults))
	}

	// Verify each membership has the expected fields.
	memberRecordIDs := make(map[string]bool)
	for _, r := range memberResults {
		m := toObject(t, r)
		recordID := assertIsString(t, m, "recordId")
		memberRecordIDs[recordID] = true
		assertIsString(t, m, "listId")
		assertFieldPresent(t, m, "addedAt")
	}
	for _, cID := range contactIDs {
		if !memberRecordIDs[cID] {
			t.Errorf("contact %s not found in list memberships", cID)
		}
	}

	// Remove 2 contacts.
	toRemove := contactIDs[:2]
	resp = doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v3/lists/%s/memberships/remove", listID), toRemove)
	mustStatus(t, resp, http.StatusOK)
	removeBody := readJSON(t, resp)
	removed := assertIsArray(t, removeBody, "recordIdsRemoved")
	if len(removed) != 2 {
		t.Errorf("expected 2 removed, got %d", len(removed))
	}

	// GET memberships again — verify only 3 remain.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v3/lists/%s/memberships", listID), nil)
	mustStatus(t, resp, http.StatusOK)
	finalBody := readJSON(t, resp)
	finalResults := assertIsArray(t, finalBody, "results")
	if len(finalResults) != 3 {
		t.Errorf("expected 3 memberships after removing 2, got %d", len(finalResults))
	}

	// Verify the removed contacts are no longer present.
	finalRecordIDs := make(map[string]bool)
	for _, r := range finalResults {
		m := toObject(t, r)
		finalRecordIDs[assertIsString(t, m, "recordId")] = true
	}
	for _, removedID := range toRemove {
		if finalRecordIDs[removedID] {
			t.Errorf("contact %s should have been removed but is still in list", removedID)
		}
	}
	// Verify the remaining 3 are still there.
	for _, remainID := range contactIDs[2:] {
		if !finalRecordIDs[remainID] {
			t.Errorf("contact %s should still be in list but was not found", remainID)
		}
	}
}

package conformance_test

import (
	"net/http"
	"testing"
)

// TestListDefaultProperties verifies that the seeded default contact properties
// include email, firstname, and lastname.
func TestListDefaultProperties(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) == 0 {
		t.Fatal("expected seeded properties, got empty results")
	}

	// Build a set of property names.
	names := make(map[string]bool)
	for _, r := range results {
		obj := toObject(t, r)
		name := assertIsString(t, obj, "name")
		names[name] = true
	}

	for _, expected := range []string{"email", "firstname", "lastname"} {
		if !names[expected] {
			t.Errorf("expected seeded property %q not found in results", expected)
		}
	}
}

// TestListPropertiesNoPaging verifies that the property list response has no
// paging field — properties are returned as a flat collection.
func TestListPropertiesNoPaging(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertIsArray(t, body, "results")

	if _, ok := body["paging"]; ok {
		t.Error("expected no paging field in properties response, but it was present")
	}
}

// TestCreateProperty verifies creating a property and inspecting the response.
func TestCreateProperty(t *testing.T) {
	resetServer(t)

	input := map[string]any{
		"name":      "custom_text",
		"label":     "Custom Text",
		"type":      "string",
		"fieldType": "text",
		"groupName": "contactinformation",
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "custom_text")
	assertStringField(t, body, "label", "Custom Text")
	assertStringField(t, body, "type", "string")
	assertStringField(t, body, "fieldType", "text")
	assertStringField(t, body, "groupName", "contactinformation")
	assertFieldPresent(t, body, "createdAt")
	assertFieldPresent(t, body, "updatedAt")
}

// TestCreateEnumerationProperty verifies creating a property with options and
// that the options round-trip correctly.
func TestCreateEnumerationProperty(t *testing.T) {
	resetServer(t)

	input := map[string]any{
		"name":      "custom_select",
		"label":     "Custom Select",
		"type":      "enumeration",
		"fieldType": "select",
		"groupName": "contactinformation",
		"options": []map[string]any{
			{"label": "Option A", "value": "a", "displayOrder": 0, "hidden": false},
			{"label": "Option B", "value": "b", "displayOrder": 1, "hidden": false},
		},
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "custom_select")
	assertStringField(t, body, "type", "enumeration")

	options := assertIsArray(t, body, "options")
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}

	opt0 := toObject(t, options[0])
	assertStringField(t, opt0, "label", "Option A")
	assertStringField(t, opt0, "value", "a")

	opt1 := toObject(t, options[1])
	assertStringField(t, opt1, "label", "Option B")
	assertStringField(t, opt1, "value", "b")
}

// TestGetProperty verifies retrieving a single property by name.
func TestGetProperty(t *testing.T) {
	resetServer(t)

	// email is a seeded default property.
	resp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts/email", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "email")
	assertStringField(t, body, "label", "Email")
	assertStringField(t, body, "type", "string")
	assertStringField(t, body, "fieldType", "text")
	assertFieldPresent(t, body, "createdAt")
	assertFieldPresent(t, body, "updatedAt")
}

// TestUpdateProperty verifies PATCH updates to label and description.
func TestUpdateProperty(t *testing.T) {
	resetServer(t)

	// Create a property to update.
	createInput := map[string]any{
		"name":      "updatable_prop",
		"label":     "Original Label",
		"type":      "string",
		"fieldType": "text",
		"groupName": "contactinformation",
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", createInput)
	mustStatus(t, createResp, http.StatusCreated)
	readJSON(t, createResp) // consume body

	// Update label and description.
	patchInput := map[string]any{
		"label":       "Updated Label",
		"description": "A test description",
	}
	resp := doRequest(t, http.MethodPatch, "/crm/v3/properties/contacts/updatable_prop", patchInput)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "updatable_prop")
	assertStringField(t, body, "label", "Updated Label")
	assertStringField(t, body, "description", "A test description")
}

// TestArchiveProperty verifies that DELETE archives a property and a subsequent
// GET returns 404.
func TestArchiveProperty(t *testing.T) {
	resetServer(t)

	// Create a property to archive.
	createInput := map[string]any{
		"name":      "to_archive",
		"label":     "To Archive",
		"type":      "string",
		"fieldType": "text",
		"groupName": "contactinformation",
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts", createInput)
	mustStatus(t, createResp, http.StatusCreated)
	readJSON(t, createResp) // consume body

	// Archive.
	delResp := doRequest(t, http.MethodDelete, "/crm/v3/properties/contacts/to_archive", nil)
	defer func() { _ = delResp.Body.Close() }()
	mustStatus(t, delResp, http.StatusNoContent)

	// Verify GET returns the property with archived=true.
	getResp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts/to_archive", nil)
	mustStatus(t, getResp, http.StatusOK)
	archivedProp := readJSON(t, getResp)
	assertBoolField(t, archivedProp, "archived", true)
}

// TestBatchCreateProperties verifies batch creation of multiple properties.
func TestBatchCreateProperties(t *testing.T) {
	resetServer(t)

	input := map[string]any{
		"inputs": []map[string]any{
			{"name": "batch_prop_a", "label": "Batch A", "type": "string", "fieldType": "text", "groupName": "contactinformation"},
			{"name": "batch_prop_b", "label": "Batch B", "type": "string", "fieldType": "text", "groupName": "contactinformation"},
		},
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts/batch/create", input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	names := make(map[string]bool)
	for _, r := range results {
		obj := toObject(t, r)
		names[assertIsString(t, obj, "name")] = true
	}
	if !names["batch_prop_a"] || !names["batch_prop_b"] {
		t.Errorf("expected both batch_prop_a and batch_prop_b in results, got %v", names)
	}
}

// TestBatchReadProperties verifies batch reading properties by name.
func TestBatchReadProperties(t *testing.T) {
	resetServer(t)

	// email and firstname are seeded defaults — batch read them.
	input := map[string]any{
		"inputs": []map[string]any{
			{"name": "email"},
			{"name": "firstname"},
		},
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts/batch/read", input)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	names := make(map[string]bool)
	for _, r := range results {
		obj := toObject(t, r)
		names[assertIsString(t, obj, "name")] = true
	}
	if !names["email"] || !names["firstname"] {
		t.Errorf("expected email and firstname in results, got %v", names)
	}
}

// TestListPropertyGroups verifies that listing property groups returns the
// seeded default group for contacts.
func TestListPropertyGroups(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts/groups", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) == 0 {
		t.Fatal("expected at least one property group, got empty results")
	}

	// Verify the seeded group is present.
	found := false
	for _, r := range results {
		obj := toObject(t, r)
		if assertIsString(t, obj, "name") == "contactinformation" {
			found = true
			assertStringField(t, obj, "label", "Contact Information")
			break
		}
	}
	if !found {
		t.Error("expected seeded group 'contactinformation' not found")
	}
}

// TestCreatePropertyGroup verifies creating a new property group.
func TestCreatePropertyGroup(t *testing.T) {
	resetServer(t)

	input := map[string]any{
		"name":  "customgroup",
		"label": "Custom Group",
	}
	resp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts/groups", input)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "customgroup")
	assertStringField(t, body, "label", "Custom Group")
}

// TestGetPropertyGroup verifies retrieving a single property group by name.
func TestGetPropertyGroup(t *testing.T) {
	resetServer(t)

	// contactinformation is a seeded default group.
	resp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts/groups/contactinformation", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "contactinformation")
	assertStringField(t, body, "label", "Contact Information")
}

// TestUpdatePropertyGroup verifies PATCH updates to label and displayOrder.
func TestUpdatePropertyGroup(t *testing.T) {
	resetServer(t)

	// Create a group to update.
	createInput := map[string]any{
		"name":  "updatablegroup",
		"label": "Updatable Group",
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts/groups", createInput)
	mustStatus(t, createResp, http.StatusCreated)
	readJSON(t, createResp) // consume body

	// Patch the group.
	patchInput := map[string]any{
		"label":        "Updated Group Label",
		"displayOrder": 5,
	}
	resp := doRequest(t, http.MethodPatch, "/crm/v3/properties/contacts/groups/updatablegroup", patchInput)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "name", "updatablegroup")
	assertStringField(t, body, "label", "Updated Group Label")

	// displayOrder should be 5.
	order, ok := body["displayOrder"]
	if !ok {
		t.Error("expected displayOrder field to be present")
	} else if order != float64(5) {
		t.Errorf("expected displayOrder=5, got %v", order)
	}
}

// TestArchivePropertyGroup verifies that DELETE archives a group and a
// subsequent GET returns 404.
func TestArchivePropertyGroup(t *testing.T) {
	resetServer(t)

	// Create a group to archive.
	createInput := map[string]any{
		"name":  "archivablegroup",
		"label": "Archivable Group",
	}
	createResp := doRequest(t, http.MethodPost, "/crm/v3/properties/contacts/groups", createInput)
	mustStatus(t, createResp, http.StatusCreated)
	readJSON(t, createResp) // consume body

	// Archive.
	delResp := doRequest(t, http.MethodDelete, "/crm/v3/properties/contacts/groups/archivablegroup", nil)
	defer func() { _ = delResp.Body.Close() }()
	mustStatus(t, delResp, http.StatusNoContent)

	// Verify GET returns the group with archived=true.
	getResp := doRequest(t, http.MethodGet, "/crm/v3/properties/contacts/groups/archivablegroup", nil)
	mustStatus(t, getResp, http.StatusOK)
	archivedGroup := readJSON(t, getResp)
	assertBoolField(t, archivedGroup, "archived", true)
}

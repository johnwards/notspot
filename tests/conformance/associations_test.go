package conformance_test

import (
	"fmt"
	"net/http"
	"testing"
)

func TestAssociateDefault(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Assoc", "lastname": "Test"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Assoc Corp"})
	companyID := assertIsString(t, company, "id")

	// PUT default association.
	resp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/default/companies/%s", contactID, companyID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	first := toObject(t, results[0])
	assertStringField(t, first, "toObjectId", companyID)
	types := assertIsArray(t, first, "associationTypes")
	if len(types) == 0 {
		t.Fatal("expected at least one association type")
	}
}

func TestGetAssociations(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Get", "lastname": "Assoc"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Get Assoc Corp"})
	companyID := assertIsString(t, company, "id")

	// Create association first.
	resp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/default/companies/%s", contactID, companyID), nil)
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// GET associations.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", contactID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	first := toObject(t, results[0])
	assertStringField(t, first, "toObjectId", companyID)
	assertFieldPresent(t, first, "associationTypes")
}

func TestGetAssociations_Empty(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Empty", "lastname": "Assoc"})
	contactID := assertIsString(t, contact, "id")

	resp := doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", contactID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestRemoveAssociation(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Remove", "lastname": "Assoc"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Remove Corp"})
	companyID := assertIsString(t, company, "id")

	// Create association.
	resp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/default/companies/%s", contactID, companyID), nil)
	mustStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Delete it.
	resp = doRequest(t, http.MethodDelete,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies/%s", contactID, companyID), nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// Verify removed.
	resp = doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", contactID), nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	if len(results) != 0 {
		t.Fatalf("expected 0 results after removal, got %d", len(results))
	}
}

func TestBatchAssociateDefault(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Batch", "lastname": "Default"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Batch Default Corp"})
	companyID := assertIsString(t, company, "id")

	reqBody := map[string]any{
		"inputs": []map[string]any{
			{
				"from": map[string]string{"id": contactID},
				"to":   map[string]string{"id": companyID},
			},
		},
	}

	resp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/batch/associate/default", reqBody)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "status", "COMPLETE")
	results := assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	first := toObject(t, results[0])
	from := assertIsObject(t, first, "from")
	assertStringField(t, from, "id", contactID)
	to := assertIsArray(t, first, "to")
	if len(to) == 0 {
		t.Fatal("expected at least one 'to' entry")
	}
	toFirst := toObject(t, to[0])
	assertStringField(t, toFirst, "toObjectId", companyID)
}

func TestBatchCreateAssociations(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Batch", "lastname": "Create"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Batch Create Corp"})
	companyID := assertIsString(t, company, "id")

	reqBody := map[string]any{
		"inputs": []map[string]any{
			{
				"from": map[string]string{"id": contactID},
				"to":   map[string]string{"id": companyID},
				"types": []map[string]any{
					{"associationCategory": "HUBSPOT_DEFINED", "associationTypeId": 279},
				},
			},
		},
	}

	resp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/batch/create", reqBody)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "status", "COMPLETE")
	results := assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	first := toObject(t, results[0])
	from := assertIsObject(t, first, "from")
	assertStringField(t, from, "id", contactID)
	to := assertIsArray(t, first, "to")
	if len(to) == 0 {
		t.Fatal("expected at least one 'to' entry")
	}
	toFirst := toObject(t, to[0])
	assertStringField(t, toFirst, "toObjectId", companyID)
	assertFieldPresent(t, toFirst, "associationTypes")
}

func TestBatchReadAssociations(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Batch", "lastname": "Read"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Batch Read Corp"})
	companyID := assertIsString(t, company, "id")

	// Create an association first.
	assocResp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/default/companies/%s", contactID, companyID), nil)
	mustStatus(t, assocResp, http.StatusOK)
	_ = assocResp.Body.Close()

	// Batch read.
	reqBody := map[string]any{
		"inputs": []map[string]string{
			{"id": contactID},
		},
	}

	resp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/batch/read", reqBody)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "status", "COMPLETE")
	results := assertIsArray(t, body, "results")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	first := toObject(t, results[0])
	from := assertIsObject(t, first, "from")
	assertStringField(t, from, "id", contactID)
	to := assertIsArray(t, first, "to")
	if len(to) != 1 {
		t.Fatalf("expected 1 to entry, got %d", len(to))
	}
	toFirst := toObject(t, to[0])
	assertStringField(t, toFirst, "toObjectId", companyID)
}

func TestBatchArchiveAssociations(t *testing.T) {
	resetServer(t)

	contact := createContact(t, map[string]string{"firstname": "Batch", "lastname": "Archive"})
	contactID := assertIsString(t, contact, "id")
	company := createCompany(t, map[string]string{"name": "Batch Archive Corp"})
	companyID := assertIsString(t, company, "id")

	// Create an association first.
	assocResp := doRequest(t, http.MethodPut,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/default/companies/%s", contactID, companyID), nil)
	mustStatus(t, assocResp, http.StatusOK)
	_ = assocResp.Body.Close()

	// Batch archive.
	reqBody := map[string]any{
		"inputs": []map[string]any{
			{
				"from": map[string]string{"id": contactID},
				"to":   map[string]string{"id": companyID},
			},
		},
	}

	resp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/batch/archive", reqBody)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// Verify association was removed.
	getResp := doRequest(t, http.MethodGet,
		fmt.Sprintf("/crm/v4/objects/contacts/%s/associations/companies", contactID), nil)
	mustStatus(t, getResp, http.StatusOK)
	body := readJSON(t, getResp)

	results := assertIsArray(t, body, "results")
	if len(results) != 0 {
		t.Fatalf("expected 0 results after batch archive, got %d", len(results))
	}
}

func TestListAssociationLabels(t *testing.T) {
	resetServer(t)

	resp := doRequest(t, http.MethodGet,
		"/crm/v4/associations/contacts/companies/labels", nil)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	results := assertIsArray(t, body, "results")
	// Seeded data should include at least the default contact-to-company labels.
	if len(results) < 2 {
		t.Fatalf("expected at least 2 seeded labels, got %d", len(results))
	}

	// Verify structure of first label.
	first := toObject(t, results[0])
	assertFieldPresent(t, first, "typeId")
	assertFieldPresent(t, first, "category")
	assertFieldPresent(t, first, "label")
}

func TestCreateAssociationLabel(t *testing.T) {
	resetServer(t)

	reqBody := map[string]string{
		"label":               "Custom Test Label",
		"associationCategory": "USER_DEFINED",
	}

	resp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/labels", reqBody)
	mustStatus(t, resp, http.StatusCreated)
	body := readJSON(t, resp)

	assertStringField(t, body, "label", "Custom Test Label")
	assertStringField(t, body, "category", "USER_DEFINED")
	assertFieldPresent(t, body, "typeId")

	// Verify it appears in the list.
	listResp := doRequest(t, http.MethodGet,
		"/crm/v4/associations/contacts/companies/labels", nil)
	mustStatus(t, listResp, http.StatusOK)
	listBody := readJSON(t, listResp)

	results := assertIsArray(t, listBody, "results")
	found := false
	for _, r := range results {
		obj := toObject(t, r)
		if l, ok := obj["label"].(string); ok && l == "Custom Test Label" {
			found = true
			break
		}
	}
	if !found {
		t.Error("created label not found in label list")
	}
}

func TestCreateAssociationLabel_MissingLabel(t *testing.T) {
	resetServer(t)

	reqBody := map[string]string{
		"associationCategory": "USER_DEFINED",
	}

	resp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/labels", reqBody)
	mustStatus(t, resp, http.StatusBadRequest)
	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

func TestUpdateAssociationLabel(t *testing.T) {
	resetServer(t)

	// Create a label first.
	createBody := map[string]string{
		"label":               "Before Update",
		"associationCategory": "USER_DEFINED",
	}
	createResp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/labels", createBody)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)

	typeID, _ := created["typeId"].(float64)

	// Update label.
	updateBody := map[string]any{
		"associationTypeId": int(typeID),
		"label":             "After Update",
	}
	resp := doRequest(t, http.MethodPut,
		"/crm/v4/associations/contacts/companies/labels", updateBody)
	mustStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	assertStringField(t, body, "label", "After Update")
	assertFieldPresent(t, body, "typeId")
}

func TestUpdateAssociationLabel_MissingTypeID(t *testing.T) {
	resetServer(t)

	updateBody := map[string]any{
		"label": "No TypeID",
	}
	resp := doRequest(t, http.MethodPut,
		"/crm/v4/associations/contacts/companies/labels", updateBody)
	mustStatus(t, resp, http.StatusBadRequest)
	body := readJSON(t, resp)
	assertHubSpotError(t, body, "VALIDATION_ERROR")
}

func TestDeleteAssociationLabel(t *testing.T) {
	resetServer(t)

	// Create a label to delete.
	createBody := map[string]string{
		"label":               "To Delete",
		"associationCategory": "USER_DEFINED",
	}
	createResp := doRequest(t, http.MethodPost,
		"/crm/v4/associations/contacts/companies/labels", createBody)
	mustStatus(t, createResp, http.StatusCreated)
	created := readJSON(t, createResp)

	typeIDFloat, _ := created["typeId"].(float64)
	typeID := int(typeIDFloat)

	// Delete it.
	resp := doRequest(t, http.MethodDelete,
		fmt.Sprintf("/crm/v4/associations/contacts/companies/labels/%d", typeID), nil)
	mustStatus(t, resp, http.StatusNoContent)
	_ = resp.Body.Close()

	// Verify it's gone from the list.
	listResp := doRequest(t, http.MethodGet,
		"/crm/v4/associations/contacts/companies/labels", nil)
	mustStatus(t, listResp, http.StatusOK)
	listBody := readJSON(t, listResp)

	results := assertIsArray(t, listBody, "results")
	for _, r := range results {
		obj := toObject(t, r)
		if id, ok := obj["typeId"].(float64); ok && int(id) == typeID {
			t.Errorf("label with typeId %d should have been deleted but still present", typeID)
		}
	}
}

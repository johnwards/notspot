package conformance_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// doRequest makes an HTTP request to the test server and returns the response.
// The caller is responsible for closing the response body.
func doRequest(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, serverURL+path, bodyReader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// readJSON reads the response body and unmarshals it into a map.
func readJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("unmarshal response (status %d): body=%s err=%v", resp.StatusCode, string(b), err)
	}
	return result
}

// mustStatus asserts the HTTP response has the expected status code.
func mustStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d; body=%s", expected, resp.StatusCode, string(b))
	}
}

// resetServer calls POST /_notspot/reset to return the server to its seeded state.
func resetServer(t *testing.T) {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/_notspot/reset", nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("reset server failed: status=%d body=%s", resp.StatusCode, string(b))
	}
}

// assertHubSpotError validates the response matches the standard HubSpot error format.
func assertHubSpotError(t *testing.T, body map[string]any, expectedCategory string) {
	t.Helper()
	assertStringField(t, body, "status", "error")
	assertFieldPresent(t, body, "message")
	assertFieldPresent(t, body, "correlationId")
	if expectedCategory != "" {
		assertStringField(t, body, "category", expectedCategory)
	}
}

// assertFieldPresent checks that a key exists in the map.
func assertFieldPresent(t *testing.T, m map[string]any, key string) {
	t.Helper()
	if _, ok := m[key]; !ok {
		t.Errorf("expected field %q to be present, got keys: %v", key, mapKeys(m))
	}
}

// assertStringField checks that a key exists and has the expected string value.
func assertStringField(t *testing.T, m map[string]any, key, expected string) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("expected field %q to be present", key)
		return
	}
	s, ok := v.(string)
	if !ok {
		t.Errorf("expected field %q to be string, got %T", key, v)
		return
	}
	if s != expected {
		t.Errorf("field %q: expected %q, got %q", key, expected, s)
	}
}

// assertBoolField checks that a key exists and has the expected boolean value.
func assertBoolField(t *testing.T, m map[string]any, key string, expected bool) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("expected field %q to be present", key)
		return
	}
	b, ok := v.(bool)
	if !ok {
		t.Errorf("expected field %q to be bool, got %T", key, v)
		return
	}
	if b != expected {
		t.Errorf("field %q: expected %v, got %v", key, expected, b)
	}
}

// assertIsString checks that a field is a non-empty string and returns its value.
func assertIsString(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("expected field %q to be present", key)
		return ""
	}
	s, ok := v.(string)
	if !ok {
		t.Errorf("expected field %q to be string, got %T", key, v)
		return ""
	}
	return s
}

// assertIsArray checks that a field is a JSON array and returns it.
func assertIsArray(t *testing.T, m map[string]any, key string) []any {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("expected field %q to be present", key)
		return nil
	}
	a, ok := v.([]any)
	if !ok {
		t.Errorf("expected field %q to be array, got %T", key, v)
		return nil
	}
	return a
}

// assertIsObject checks that a field is a JSON object and returns it.
func assertIsObject(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("expected field %q to be present", key)
		return nil
	}
	o, ok := v.(map[string]any)
	if !ok {
		t.Errorf("expected field %q to be object, got %T", key, v)
		return nil
	}
	return o
}

// assertISOTimestamp checks that a string value is a valid ISO 8601 timestamp.
func assertISOTimestamp(t *testing.T, value string) {
	t.Helper()
	if value == "" {
		t.Error("expected non-empty ISO timestamp")
		return
	}
	formats := []string{
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, f := range formats {
		if _, err := time.Parse(f, value); err == nil {
			return
		}
	}
	t.Errorf("value %q is not a valid ISO 8601 timestamp", value)
}

// assertPaging checks that the response has a valid paging structure.
func assertPaging(t *testing.T, body map[string]any) {
	t.Helper()
	paging := assertIsObject(t, body, "paging")
	if paging == nil {
		return
	}
	next := assertIsObject(t, paging, "next")
	if next == nil {
		return
	}
	assertFieldPresent(t, next, "after")
}

// assertSimplePublicObject validates the core fields of a CRM object response.
func assertSimplePublicObject(t *testing.T, obj map[string]any) {
	t.Helper()
	assertIsString(t, obj, "id")
	assertIsObject(t, obj, "properties")
	assertISOTimestamp(t, assertIsString(t, obj, "createdAt"))
	assertISOTimestamp(t, assertIsString(t, obj, "updatedAt"))
	assertBoolField(t, obj, "archived", false)
}

// toObject converts a slice element to a map.
func toObject(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", v)
	}
	return m
}

// createContact is a helper that creates a contact and returns the response body.
func createContact(t *testing.T, props map[string]string) map[string]any {
	t.Helper()
	body := map[string]any{"properties": props}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/contacts", body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create contact: status=%d body=%s", resp.StatusCode, string(b))
	}
	return readJSON(t, resp)
}

// createCompany is a helper that creates a company and returns the response body.
func createCompany(t *testing.T, props map[string]string) map[string]any {
	t.Helper()
	body := map[string]any{"properties": props}
	resp := doRequest(t, http.MethodPost, "/crm/v3/objects/companies", body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("create company: status=%d body=%s", resp.StatusCode, string(b))
	}
	return readJSON(t, resp)
}

// mapKeys returns the keys of a map for diagnostic output.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

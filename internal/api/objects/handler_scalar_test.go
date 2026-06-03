package objects_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/johnwards/hubspot/internal/domain"
)

// HubSpot accepts property values as JSON strings, numbers, or booleans on writes and coerces them
// all to strings. A strict map[string]string decode used to reject any non-string value with a 400
// ("Invalid input JSON"). Verify the tolerant domain.Properties decode accepts scalars and stores
// their string form. (Regression test for the numeric-property fix.)
func TestCreate_CoercesScalarPropertyValues(t *testing.T) {
	srv := setupServer(t)
	defer srv.Close()

	body := `{"properties":{"email":"scalar@example.com","step_number":7,"ratio":3.5,"vip":true,"note":null}}`
	resp, err := http.Post(srv.URL+"/crm/v3/objects/contacts", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201; body = %s", resp.StatusCode, b)
	}

	var obj domain.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for k, want := range map[string]string{"step_number": "7", "ratio": "3.5", "vip": "true", "note": ""} {
		if got := obj.Properties[k]; got != want {
			t.Errorf("property %q = %q, want %q", k, got, want)
		}
	}
}

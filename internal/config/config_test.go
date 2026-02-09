package config_test

import (
	"testing"

	"github.com/johnwards/hubspot/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	// Unset any env vars that might be set.
	t.Setenv("NOTSPOT_ADDR", "")
	t.Setenv("NOTSPOT_DB", "")
	t.Setenv("NOTSPOT_AUTH_TOKEN", "")

	cfg := config.Load()

	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.DBPath != "notspot.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "notspot.db")
	}
	if cfg.AuthToken != "" {
		t.Errorf("AuthToken = %q, want empty", cfg.AuthToken)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("NOTSPOT_ADDR", ":9090")
	t.Setenv("NOTSPOT_DB", "/tmp/test.db")
	t.Setenv("NOTSPOT_AUTH_TOKEN", "secret-token")

	cfg := config.Load()

	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":9090")
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/test.db")
	}
	if cfg.AuthToken != "secret-token" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "secret-token")
	}
}

package config

import "os"

// Config holds application configuration loaded from environment variables.
type Config struct {
	Addr      string // NOTSPOT_ADDR, default ":8080"
	DBPath    string // NOTSPOT_DB, default "notspot.db"
	AuthToken string // NOTSPOT_AUTH_TOKEN, optional
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		Addr:      envOr("NOTSPOT_ADDR", ":8080"),
		DBPath:    envOr("NOTSPOT_DB", "notspot.db"),
		AuthToken: os.Getenv("NOTSPOT_AUTH_TOKEN"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

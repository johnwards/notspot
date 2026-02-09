package admin

import (
	"database/sql"
	"net/http"
)

// RegisterRoutes registers all admin API endpoints on the mux.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	h := &Handler{db: db}

	mux.HandleFunc("POST /_notspot/reset", h.Reset)
	mux.HandleFunc("GET /_notspot/requests", h.Requests)
	mux.HandleFunc("POST /_notspot/seed", h.SeedData)
}

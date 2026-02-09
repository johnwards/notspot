package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/johnwards/hubspot/internal/api"
	"github.com/johnwards/hubspot/internal/api/admin"
	"github.com/johnwards/hubspot/internal/api/associations"
	"github.com/johnwards/hubspot/internal/api/exports"
	"github.com/johnwards/hubspot/internal/api/imports"
	"github.com/johnwards/hubspot/internal/api/lists"
	"github.com/johnwards/hubspot/internal/api/objects"
	"github.com/johnwards/hubspot/internal/api/owners"
	"github.com/johnwards/hubspot/internal/api/pipelines"
	"github.com/johnwards/hubspot/internal/api/properties"
	"github.com/johnwards/hubspot/internal/api/schemas"
	"github.com/johnwards/hubspot/internal/api/ui"
	"github.com/johnwards/hubspot/internal/config"
	"github.com/johnwards/hubspot/internal/database"
	"github.com/johnwards/hubspot/internal/seed"
	"github.com/johnwards/hubspot/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := database.Migrate(ctx, db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	if err := seed.Seed(ctx, db); err != nil {
		return fmt.Errorf("seed data: %w", err)
	}

	s := store.New(db)

	mux := http.NewServeMux()

	// CRM API routes
	objects.RegisterRoutes(mux, s)
	properties.RegisterRoutes(mux, s.DB)
	pipelines.RegisterRoutes(mux, s.DB)
	associations.RegisterRoutes(mux, s.DB)
	schemas.RegisterRoutes(mux, s.DB)
	imports.RegisterRoutes(mux, s)
	exports.RegisterRoutes(mux, s)
	owners.RegisterRoutes(mux, s)
	lists.RegisterRoutes(mux, s)

	// Admin API
	admin.RegisterRoutes(mux, s.DB)

	// Web UI
	ui.RegisterRoutes(mux)

	// Catch-all: return 404 in HubSpot error format.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		corrID := api.CorrelationID(r.Context())
		api.WriteError(w, http.StatusNotFound, api.NewNotFoundError(
			fmt.Sprintf("No route found for %s %s", r.Method, r.URL.Path),
			corrID,
		))
	})

	handler := api.Chain(mux,
		api.Recovery(),
		api.RequestID(),
		api.Auth(cfg.AuthToken),
		api.JSONContentType(),
		api.Logging(),
	)

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: handler,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutting down server")
		if err := srv.Shutdown(context.Background()); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("starting notspot server", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}

	return nil
}

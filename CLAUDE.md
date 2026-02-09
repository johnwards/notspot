# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this?

Notspot is a clean-room HubSpot API mock server — a drop-in replacement for `api.hubapi.com` in integration tests. It uses Go stdlib `net/http` with Go 1.22+ pattern matching and SQLite (pure Go via `modernc.org/sqlite`) with an EAV (Entity-Attribute-Value) pattern for schemaless CRM properties.

## Build & Test Commands

```bash
make check          # Run all checks: fmt, vet, lint, test
make build          # Build binary to build/hubspot
make test           # Unit tests with race detector
make lint           # golangci-lint (11 linters, see .golangci.yml)
make conformance    # Black-box conformance tests (builds server, starts it, runs HTTP tests)

# Run a single unit test
go test -race -run TestFunctionName ./internal/store/...

# Run a single conformance test
go test -v -run TestName ./tests/conformance/...
```

## Architecture

### Layer structure

```
cmd/hubspot/main.go          → Entry point: DB open → migrate → seed → register routes → serve
internal/config/              → Env-based config (NOTSPOT_ADDR, NOTSPOT_DB, NOTSPOT_AUTH_TOKEN)
internal/database/            → SQLite open (WAL, FK, single conn), versioned migrations
internal/seed/                → Idempotent seed data (properties, pipelines, associations, owners)
internal/domain/              → Pure data types (Object, Property, Pipeline, Association, etc.)
internal/store/               → SQLite-backed data access; Store struct aggregates sub-stores
internal/api/                 → Middleware (auth, request ID, logging, recovery, JSON content type)
internal/api/{feature}/       → Each feature has routes.go (RegisterRoutes) + handler.go + handler_test.go
internal/testhelpers/         → NewTestDB() — in-memory SQLite for unit tests
tests/conformance/            → Independent black-box tests against a running server
```

### Key patterns

- **Route registration**: Each API feature package exports `RegisterRoutes(mux, store/db)`. Handlers are methods on a `Handler` struct that holds the store dependency.
- **Middleware chain**: `api.Chain(mux, Recovery(), RequestID(), Auth(), JSONContentType(), Logging())` — outermost middleware listed first.
- **Error responses**: All errors use HubSpot-compatible format via `api.WriteError()` with `api.Error` struct containing `status`, `message`, `correlationId`, `category`.
- **Store layer**: Some handlers receive `*store.Store` (composite), others receive `*sql.DB` directly. The `Store` struct aggregates `ObjectStore`, `SearchStore`, `ImportStore`, `ExportStore`, `OwnerStore`, `ListStore`.
- **Conformance tests**: `tests/conformance/main_test.go` builds the binary, starts the server on a random port with `:memory:` DB, then runs HTTP tests against it. Tests use a shared `serverURL` package variable.
- **Admin API**: `/_notspot/reset` endpoint for test control (reset state, re-seed).

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `NOTSPOT_ADDR` | `:8080` | Listen address |
| `NOTSPOT_DB` | `notspot.db` | SQLite database path (`:memory:` for tests) |
| `NOTSPOT_AUTH_TOKEN` | (empty) | If set, requires `Bearer <token>` auth |

## HubSpot API conventions to follow

- CRM objects share unified CRUD at `/crm/v3/objects/{objectType}`
- Properties endpoints return all results (no pagination) in `{"results": [...]}`
- Cursor-based pagination with `after` parameter and `paging.next.after` in responses
- Standard error format: `{status, message, correlationId, category, errors[]}`

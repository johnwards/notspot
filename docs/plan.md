# Notspot Implementation Plan

**A clean-room HubSpot API mock server — standalone Go binary backed by SQLite.**

Purpose: Drop-in replacement for `api.hubapi.com` in integration tests. Avoids rate limits, enables offline testing, provides deterministic state control.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Scope & Prioritization](#2-scope--prioritization)
3. [Phase 1: Foundation](#3-phase-1-foundation)
4. [Phase 2: CRM Objects Core](#4-phase-2-crm-objects-core)
5. [Phase 3: Properties & Pipelines](#5-phase-3-properties--pipelines)
6. [Phase 4: Associations v4](#6-phase-4-associations-v4)
7. [Phase 5: Search](#7-phase-5-search)
8. [Phase 6: Lists & Memberships](#8-phase-6-lists--memberships)
9. [Phase 7: Custom Object Schemas](#9-phase-7-custom-object-schemas)
10. [Phase 8: Imports & Exports](#10-phase-8-imports--exports)
11. [Phase 9: Owners & Users](#11-phase-9-owners--users)
12. [Phase 10: Extended APIs](#12-phase-10-extended-apis)
13. [Future: Web UI](#13-future-web-ui)
14. [SQLite Schema](#14-sqlite-schema)
15. [Go Package Layout](#15-go-package-layout)
16. [Cross-Cutting Concerns](#16-cross-cutting-concerns)
17. [Agent Task Breakdown](#17-agent-task-breakdown)

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────┐
│                  notspot binary                  │
│                                                  │
│  ┌────────────┐  ┌───────────┐  ┌────────────┐  │
│  │  HTTP/Router│  │  Handlers │  │  Middleware │  │
│  │  (stdlib)   │──│  per-API  │──│  auth/err   │  │
│  └────────────┘  └───────────┘  └────────────┘  │
│                       │                          │
│  ┌────────────────────┴──────────────────────┐   │
│  │              Store Layer                   │   │
│  │  (interfaces for each domain)              │   │
│  └────────────────────┬──────────────────────┘   │
│                       │                          │
│  ┌────────────────────┴──────────────────────┐   │
│  │          SQLite (modernc.org/sqlite)       │   │
│  └───────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### Key Design Decisions

- **No external framework**: Use `net/http` stdlib router (Go 1.22+ pattern matching). Keeps the binary small and dependency-free.
- **Single binary**: Everything in one `go build` output. No containers needed.
- **SQLite via modernc.org/sqlite**: Pure Go, no CGO, cross-platform.
- **EAV for properties**: Match HubSpot's schemaless property model using Entity-Attribute-Value pattern.
- **JSON columns for complex structures**: List filter branches, pipeline stage metadata, etc. stored as JSON text in SQLite.
- **Deterministic IDs**: Auto-incrementing integers, cast to strings to match HubSpot's string IDs.
- **Future web UI**: All state lives in SQLite, so a future web UI just reads the same DB. Keep handlers as thin REST wrappers over the store layer.

---

## 2. Scope & Prioritization

### In scope (Phases 1-9)

These are the APIs most commonly used in integration testing:

| Priority | API Area | Why |
|----------|----------|-----|
| P0 | CRM Objects CRUD (all types) | Core of every HubSpot integration |
| P0 | Properties & Property Groups | Required for object creation/validation |
| P0 | Associations v4 | Required for any relational data |
| P0 | CRM Search | Most integrations search for records |
| P1 | Pipelines & Stages | Deal/ticket workflows |
| P1 | Lists & Memberships | Segmentation testing |
| P1 | Custom Object Schemas | Custom object testing |
| P2 | Imports & Exports | Bulk data testing |
| P2 | Owners | User assignment testing |

### Out of scope (Phase 10 — implement on demand)

| API Area | Notes |
|----------|-------|
| CMS (Blog, Pages, HubDB, Source Code) | Rarely used in API integrations |
| Marketing (Emails, Forms, Events) | Implement when needed |
| Automation (Workflows, Sequences) | Complex, rarely mocked |
| Conversations | Niche |
| Webhooks | Would need a push mechanism |
| OAuth flow | Tests typically use static Bearer tokens |
| Commerce (Payments, Invoices) | Implement when needed |
| Files API | Implement when needed |

### What we're NOT implementing

- Rate limiting enforcement (defeats the purpose of a test server)
- OAuth token exchange/refresh (accept any Bearer token)
- Webhook delivery
- Calculated/rollup properties evaluation
- Dynamic list filter evaluation (store filters as JSON, don't evaluate)
- GDPR delete compliance

---

## 3. Phase 1: Foundation

**Goal**: HTTP server skeleton, middleware, error handling, database migrations, configuration.

### 3.1 Configuration

```go
type Config struct {
    Addr       string // listen address, default ":8080"
    DBPath     string // SQLite path, default "notspot.db"
    AuthToken  string // optional: if set, validate Bearer tokens match
}
```

Load from environment variables: `NOTSPOT_ADDR`, `NOTSPOT_DB`, `NOTSPOT_AUTH_TOKEN`.

### 3.2 HTTP Server & Router

Use Go 1.22+ `http.ServeMux` with method+pattern matching:

```go
mux.HandleFunc("GET /crm/v3/objects/{objectType}", h.ListObjects)
mux.HandleFunc("POST /crm/v3/objects/{objectType}", h.CreateObject)
// etc.
```

### 3.3 Middleware Stack

1. **Recovery**: Catch panics, return 500 with HubSpot error format
2. **Auth**: Check `Authorization: Bearer {token}` header exists. If `NOTSPOT_AUTH_TOKEN` is set, validate it matches. Always pass through if not configured (test convenience).
3. **Request ID**: Generate `correlationId` for every request (UUID v4)
4. **JSON Content-Type**: Set `Content-Type: application/json` on all responses
5. **Logging**: Structured logging (slog) with method, path, status, duration

### 3.4 Error Response Format

Every error response must match HubSpot's format exactly:

```json
{
  "status": "error",
  "message": "Human readable message",
  "correlationId": "uuid-here",
  "category": "VALIDATION_ERROR",
  "errors": [
    {
      "message": "Specific error detail",
      "code": "INVALID_INTEGER",
      "in": "fieldname",
      "context": {"propertyName": ["discount"]},
      "subCategory": "string"
    }
  ]
}
```

Implement as a Go struct + helper functions:

```go
type APIError struct {
    Status        string        `json:"status"`
    Message       string        `json:"message"`
    CorrelationID string        `json:"correlationId"`
    Category      string        `json:"category"`
    SubCategory   string        `json:"subCategory,omitempty"`
    Errors        []ErrorDetail `json:"errors,omitempty"`
}
```

Standard categories: `VALIDATION_ERROR`, `OBJECT_NOT_FOUND`, `CONFLICT`, `RATE_LIMITS`.

### 3.5 Database Migrations

Extend the existing migration framework in `internal/database/database.go`. Add numbered migration files that create all tables (see [SQLite Schema](#14-sqlite-schema)).

### 3.6 Files to create/modify

| File | Action |
|------|--------|
| `cmd/hubspot/main.go` | Rewrite: parse config, set up router, register all handlers |
| `internal/config/config.go` | New: configuration loading |
| `internal/api/errors.go` | New: error types and helpers |
| `internal/api/middleware.go` | New: middleware stack |
| `internal/api/response.go` | New: JSON response helpers (write, paginate) |
| `internal/database/database.go` | Modify: add migration SQL |

### 3.7 Deliverable

A running server that returns `404` HubSpot-formatted errors for all routes, with auth middleware and logging.

---

## 4. Phase 2: CRM Objects Core

**Goal**: Implement the unified CRM object CRUD that serves ALL object types through a single code path.

### 4.1 Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/crm/v3/objects/{objectType}` | ListObjects |
| `POST` | `/crm/v3/objects/{objectType}` | CreateObject |
| `GET` | `/crm/v3/objects/{objectType}/{objectId}` | GetObject |
| `PATCH` | `/crm/v3/objects/{objectType}/{objectId}` | UpdateObject |
| `DELETE` | `/crm/v3/objects/{objectType}/{objectId}` | ArchiveObject |
| `POST` | `/crm/v3/objects/{objectType}/batch/create` | BatchCreate |
| `POST` | `/crm/v3/objects/{objectType}/batch/read` | BatchRead |
| `POST` | `/crm/v3/objects/{objectType}/batch/update` | BatchUpdate |
| `POST` | `/crm/v3/objects/{objectType}/batch/upsert` | BatchUpsert |
| `POST` | `/crm/v3/objects/{objectType}/batch/archive` | BatchArchive |
| `POST` | `/crm/v3/objects/{objectType}/merge` | MergeObjects |

Also register type-specific aliases that route to the same handler:
- `/crm/v3/objects/contacts/{contactId}` → same as `/crm/v3/objects/contacts/{objectId}`
- etc. for companies, deals, tickets, and all other types

### 4.2 Object Type Registry

Maintain an in-memory registry of all known object types, initialized from a seed list on startup:

```go
var standardTypes = map[string]ObjectTypeDef{
    "contacts":       {ID: "0-1", Singular: "Contact", Plural: "Contacts"},
    "companies":      {ID: "0-2", Singular: "Company", Plural: "Companies"},
    "deals":          {ID: "0-3", Singular: "Deal", Plural: "Deals"},
    "tickets":        {ID: "0-5", Singular: "Ticket", Plural: "Tickets"},
    "products":       {ID: "0-7", Singular: "Product", Plural: "Products"},
    "line_items":     {ID: "0-8", Singular: "Line Item", Plural: "Line Items"},
    "quotes":         {ID: "0-14", Singular: "Quote", Plural: "Quotes"},
    "calls":          {ID: "0-48", Singular: "Call", Plural: "Calls"},
    "emails":         {ID: "0-49", Singular: "Email", Plural: "Emails"},
    "meetings":       {ID: "0-47", Singular: "Meeting", Plural: "Meetings"},
    "notes":          {ID: "0-46", Singular: "Note", Plural: "Notes"},
    "tasks":          {ID: "0-27", Singular: "Task", Plural: "Tasks"},
    "communications": {ID: "0-18", Singular: "Communication", Plural: "Communications"},
    "postal_mail":    {ID: "0-116", Singular: "Postal Mail", Plural: "Postal Mails"},
    "leads":          {ID: "0-136", Singular: "Lead", Plural: "Leads"},
    "goals":          {ID: "0-74", Singular: "Goal", Plural: "Goals"},
    "orders":         {ID: "0-123", Singular: "Order", Plural: "Orders"},
    "carts":          {ID: "0-142", Singular: "Cart", Plural: "Carts"},
    "invoices":       {ID: "0-53", Singular: "Invoice", Plural: "Invoices"},
    "feedback_submissions": {ID: "0-19", Singular: "Feedback Submission", Plural: "Feedback Submissions"},
}
```

Custom objects get registered dynamically when schemas are created (Phase 7).

### 4.3 Request/Response Shapes

**SimplePublicObject** (response for single reads):
```json
{
  "id": "12345",
  "properties": {
    "hs_object_id": "12345",
    "hs_createdate": "2024-11-20T20:12:09.236Z",
    "hs_lastmodifieddate": "2024-11-20T20:12:10.610Z",
    "email": "test@example.com"
  },
  "createdAt": "2024-11-20T20:12:09.236Z",
  "updatedAt": "2024-11-20T20:12:10.610Z",
  "archived": false
}
```

**Create request** (`SimplePublicObjectInputForCreate`):
```json
{
  "properties": {"email": "test@example.com", "firstname": "Test"},
  "associations": [
    {
      "to": {"id": "67890"},
      "types": [{"associationCategory": "HUBSPOT_DEFINED", "associationTypeId": 1}]
    }
  ]
}
```

**List response** (paginated):
```json
{
  "results": [SimplePublicObject, ...],
  "paging": {
    "next": {"after": "100", "link": "..."}
  }
}
```

**Batch response**:
```json
{
  "status": "COMPLETE",
  "results": [SimplePublicObject, ...],
  "startedAt": "...",
  "completedAt": "...",
  "errors": [],
  "numErrors": 0
}
```

### 4.4 Query Parameters

| Parameter | Endpoints | Description |
|-----------|-----------|-------------|
| `properties` | GET list, GET single | Comma-separated property names to include |
| `propertiesWithHistory` | GET list, GET single | Include historical values |
| `associations` | GET list, GET single | Comma-separated object types to include associations for |
| `limit` | GET list | Page size (default 10, max 100) |
| `after` | GET list | Cursor for pagination |
| `archived` | GET list, GET single | Include archived records |

### 4.5 Behavior Notes

- **Default properties**: Only `hs_object_id`, `hs_createdate`, `hs_lastmodifieddate` returned unless `properties` param specified
- **Batch limits**: Max 100 items per batch request
- **Upsert**: Match by `idProperty` (defaults to `hs_object_id` for most types, `email` for contacts)
- **Archive**: Soft-delete (set `archived=true`, `archivedAt=now`)
- **Merge**: Combine two records — `primaryObjectId` survives, `objectIdToMerge` gets archived. Store merged IDs in `hs_merged_object_ids` property.
- **ID format**: String representation of auto-increment integers
- **Create response difference**: Generic `/crm/v3/objects/{objectType}` returns `200` with `{"createdResourceId", "entity", "location"}` wrapper. Type-specific endpoints (e.g., `/crm/v3/objects/contacts`) return `201` with `SimplePublicObject` directly. Our handler should use the generic wrapper since most clients use the generic endpoint.
- **idProperty param**: Objects can be fetched/updated by any unique property (e.g., `?idProperty=email`) not just by ID
- **All property values are strings**: Even numbers, dates, booleans — always `map[string]string`

### 4.6 Store Interface

```go
type ObjectStore interface {
    Create(ctx context.Context, objectType string, properties map[string]string) (*Object, error)
    Get(ctx context.Context, objectType string, id string, props []string) (*Object, error)
    List(ctx context.Context, objectType string, opts ListOpts) (*ObjectPage, error)
    Update(ctx context.Context, objectType string, id string, properties map[string]string) (*Object, error)
    Archive(ctx context.Context, objectType string, id string) error
    BatchCreate(ctx context.Context, objectType string, inputs []CreateInput) (*BatchResult, error)
    BatchRead(ctx context.Context, objectType string, ids []string, props []string) (*BatchResult, error)
    BatchUpdate(ctx context.Context, objectType string, inputs []UpdateInput) (*BatchResult, error)
    BatchUpsert(ctx context.Context, objectType string, inputs []UpsertInput, idProperty string) (*BatchResult, error)
    BatchArchive(ctx context.Context, objectType string, ids []string) error
    Merge(ctx context.Context, objectType string, primaryID, mergeID string) (*Object, error)
}
```

### 4.7 Files to create

| File | Purpose |
|------|---------|
| `internal/api/objects/handler.go` | HTTP handlers for all object endpoints |
| `internal/api/objects/routes.go` | Route registration |
| `internal/store/objects.go` | ObjectStore interface + SQLite implementation |
| `internal/domain/object.go` | Domain types (Object, CreateInput, etc.) |

---

## 5. Phase 3: Properties & Pipelines

### 5.1 Properties API Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/crm/v3/properties/{objectType}` | ListProperties |
| `POST` | `/crm/v3/properties/{objectType}` | CreateProperty |
| `GET` | `/crm/v3/properties/{objectType}/{propertyName}` | GetProperty |
| `PATCH` | `/crm/v3/properties/{objectType}/{propertyName}` | UpdateProperty |
| `DELETE` | `/crm/v3/properties/{objectType}/{propertyName}` | ArchiveProperty |
| `POST` | `/crm/v3/properties/{objectType}/batch/create` | BatchCreateProperties |
| `POST` | `/crm/v3/properties/{objectType}/batch/read` | BatchReadProperties |
| `POST` | `/crm/v3/properties/{objectType}/batch/archive` | BatchArchiveProperties |
| `GET` | `/crm/v3/properties/{objectType}/groups` | ListGroups |
| `POST` | `/crm/v3/properties/{objectType}/groups` | CreateGroup |
| `GET` | `/crm/v3/properties/{objectType}/groups/{groupName}` | GetGroup |
| `PATCH` | `/crm/v3/properties/{objectType}/groups/{groupName}` | UpdateGroup |
| `DELETE` | `/crm/v3/properties/{objectType}/groups/{groupName}` | ArchiveGroup |

### 5.2 Property Model

```go
type Property struct {
    Name              string   `json:"name"`
    Label             string   `json:"label"`
    Type              string   `json:"type"`              // bool, enumeration, date, datetime, string, number
    FieldType         string   `json:"fieldType"`         // text, select, number, date, etc.
    GroupName         string   `json:"groupName"`
    Description       string   `json:"description,omitempty"`
    Options           []Option `json:"options,omitempty"`
    DisplayOrder      int      `json:"displayOrder"`
    HasUniqueValue    bool     `json:"hasUniqueValue"`
    Hidden            bool     `json:"hidden"`
    FormField         bool     `json:"formField"`
    Calculated        bool     `json:"calculated"`
    ExternalOptions   bool     `json:"externalOptions"`
    Archived          bool     `json:"archived"`
    HubspotDefined    bool     `json:"hubspotDefined"`
    CreatedAt         string   `json:"createdAt,omitempty"`
    UpdatedAt         string   `json:"updatedAt,omitempty"`
    ArchivedAt        string   `json:"archivedAt,omitempty"`
    ModificationMetadata *ModificationMetadata `json:"modificationMetadata,omitempty"`
}
```

**No pagination** on property list — returns all properties for the object type in `{ "results": [...] }`.

### 5.3 Default Properties Seeding

On first boot (or via migration), seed default properties for standard object types. At minimum:

- **All types**: `hs_object_id`, `hs_createdate`, `hs_lastmodifieddate`
- **Contacts**: `email`, `firstname`, `lastname`, `phone`, `company`, `lifecyclestage`
- **Companies**: `name`, `domain`, `industry`, `lifecyclestage`
- **Deals**: `dealname`, `dealstage`, `pipeline`, `amount`, `closedate`
- **Tickets**: `subject`, `content`, `hs_pipeline`, `hs_pipeline_stage`, `hs_ticket_priority`

Store seed data as embedded JSON or Go maps. Mark seeded properties as `hubspotDefined: true`.

### 5.4 Pipelines API Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/crm/v3/pipelines/{objectType}` | ListPipelines |
| `POST` | `/crm/v3/pipelines/{objectType}` | CreatePipeline |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | GetPipeline |
| `PATCH` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | UpdatePipeline |
| `PUT` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | ReplacePipeline |
| `DELETE` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | DeletePipeline |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages` | ListStages |
| `POST` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages` | CreateStage |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | GetStage |
| `PATCH` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | UpdateStage |
| `PUT` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | ReplaceStage |
| `DELETE` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | DeleteStage |

### 5.5 Pipeline Model

```go
type Pipeline struct {
    ID           string          `json:"id"`
    Label        string          `json:"label"`
    DisplayOrder int             `json:"displayOrder"`
    Stages       []PipelineStage `json:"stages"`
    Archived     bool            `json:"archived"`
    CreatedAt    string          `json:"createdAt"`
    UpdatedAt    string          `json:"updatedAt"`
}

type PipelineStage struct {
    ID           string            `json:"id"`
    Label        string            `json:"label"`
    DisplayOrder int               `json:"displayOrder"`
    Metadata     map[string]string `json:"metadata"` // probability, isClosed, ticketState
    Archived     bool              `json:"archived"`
    CreatedAt    string            `json:"createdAt"`
    UpdatedAt    string            `json:"updatedAt"`
}
```

**No pagination** — pipeline list returns all pipelines for the object type.

Seed default pipelines:
- **Deals**: "Sales Pipeline" with stages: Appointment Scheduled (0.2), Qualified To Buy (0.3), Presentation Scheduled (0.4), Decision Maker Bought-In (0.6), Contract Sent (0.8), Closed Won (1.0, isClosed), Closed Lost (0.0, isClosed)
- **Tickets**: "Support Pipeline" with stages: New (OPEN), Waiting on contact (OPEN), Waiting on us (OPEN), Closed (CLOSED)

### 5.6 Files to create

| File | Purpose |
|------|---------|
| `internal/api/properties/handler.go` | Property CRUD handlers |
| `internal/api/properties/routes.go` | Route registration |
| `internal/api/pipelines/handler.go` | Pipeline CRUD handlers |
| `internal/api/pipelines/routes.go` | Route registration |
| `internal/store/properties.go` | PropertyStore interface + SQLite impl |
| `internal/store/pipelines.go` | PipelineStore interface + SQLite impl |
| `internal/domain/property.go` | Property domain types |
| `internal/domain/pipeline.go` | Pipeline domain types |
| `internal/seed/properties.go` | Default property definitions |
| `internal/seed/pipelines.go` | Default pipeline definitions |

---

## 6. Phase 4: Associations v4

### 6.1 Record-Level Association Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `PUT` | `/crm/v4/objects/{from}/{fromId}/associations/default/{to}/{toId}` | AssociateDefault |
| `PUT` | `/crm/v4/objects/{from}/{fromId}/associations/{to}/{toId}` | AssociateWithLabels |
| `GET` | `/crm/v4/objects/{from}/{fromId}/associations/{to}` | GetAssociations |
| `DELETE` | `/crm/v4/objects/{from}/{fromId}/associations/{to}/{toId}` | RemoveAssociations |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/associate/default` | BatchAssociateDefault |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/create` | BatchCreateAssociations |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/read` | BatchReadAssociations |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/archive` | BatchArchiveAssociations |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/labels/archive` | BatchArchiveLabels |

### 6.2 Schema/Label Management Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/crm/v4/associations/{from}/{to}/labels` | ListLabels |
| `POST` | `/crm/v4/associations/{from}/{to}/labels` | CreateLabel |
| `PUT` | `/crm/v4/associations/{from}/{to}/labels` | UpdateLabel |
| `DELETE` | `/crm/v4/associations/{from}/{to}/labels/{typeId}` | DeleteLabel |

### 6.3 Association Model

```go
type Association struct {
    FromObjectType string
    FromObjectID   string
    ToObjectType   string
    ToObjectID     string
    TypeID         int
    Category       string // HUBSPOT_DEFINED, USER_DEFINED, INTEGRATOR_DEFINED
    Label          string // nullable
}
```

### 6.4 Behavior Notes

- Associations are **directional** — A→B has a different typeId than B→A
- Creating a labeled association also implicitly creates an unlabeled one
- "Primary" associations (e.g., primary company) have special typeIds (1 for Contact→Company)
- Batch limits: 2,000 for create, 1,000 for read
- Seed standard association type definitions (Contact↔Company, Contact↔Deal, Deal↔Company, etc.)

### 6.5 Files to create

| File | Purpose |
|------|---------|
| `internal/api/associations/handler.go` | Association handlers |
| `internal/api/associations/routes.go` | Route registration |
| `internal/store/associations.go` | AssociationStore interface + SQLite impl |
| `internal/domain/association.go` | Association domain types |
| `internal/seed/associations.go` | Default association type definitions |

---

## 7. Phase 5: Search

### 7.1 Endpoint

| Method | Path | Handler |
|--------|------|---------|
| `POST` | `/crm/v3/objects/{objectType}/search` | SearchObjects |

### 7.2 Request Shape

```json
{
  "query": "full-text search string",
  "filterGroups": [
    {
      "filters": [
        {"propertyName": "email", "operator": "EQ", "value": "test@example.com"}
      ]
    }
  ],
  "sorts": [{"propertyName": "createdate", "direction": "DESCENDING"}],
  "properties": ["email", "firstname"],
  "limit": 100,
  "after": "0"
}
```

### 7.3 Filter Operators to Implement

| Operator | SQL Equivalent |
|----------|---------------|
| `EQ` | `= value` |
| `NEQ` | `!= value` |
| `LT` | `< value` |
| `LTE` | `<= value` |
| `GT` | `> value` |
| `GTE` | `>= value` |
| `BETWEEN` | `BETWEEN value AND highValue` |
| `IN` | `IN (values...)` |
| `NOT_IN` | `NOT IN (values...)` |
| `HAS_PROPERTY` | `IS NOT NULL` |
| `NOT_HAS_PROPERTY` | `IS NULL` |
| `CONTAINS_TOKEN` | `LIKE %value%` (with wildcard support) |
| `NOT_CONTAINS_TOKEN` | `NOT LIKE %value%` |

### 7.4 Search Implementation

Build dynamic SQL queries from the filter structure:
- Filters within a group → `AND`
- Multiple filter groups → `OR`
- Max 5 filter groups, 6 filters per group
- Max 200 results per page, 10,000 total cap
- Join on `property_values` table for each filtered property

The `query` field does full-text search across "searchable" properties for the object type (typically name, email, phone, etc.). Implement as `LIKE '%query%'` across searchable columns.

### 7.5 Response Shape

```json
{
  "total": 1,
  "results": [SimplePublicObject, ...],
  "paging": {"next": {"after": "100"}}
}
```

### 7.6 Files to create

| File | Purpose |
|------|---------|
| `internal/api/objects/search.go` | Search handler (in objects package) |
| `internal/store/search.go` | Search query builder + executor |

---

## 8. Phase 6: Lists & Memberships

### 8.1 Endpoints (core subset)

| Method | Path | Handler |
|--------|------|---------|
| `POST` | `/crm/v3/lists` | CreateList |
| `GET` | `/crm/v3/lists/{listId}` | GetList |
| `DELETE` | `/crm/v3/lists/{listId}` | DeleteList |
| `PUT` | `/crm/v3/lists/{listId}/restore` | RestoreList |
| `PUT` | `/crm/v3/lists/{listId}/update-list-name` | UpdateListName |
| `PUT` | `/crm/v3/lists/{listId}/update-list-filters` | UpdateListFilters |
| `POST` | `/crm/v3/lists/search` | SearchLists |
| `GET` | `/crm/v3/lists/` | GetMultipleLists |
| `GET` | `/crm/v3/lists/{listId}/memberships` | GetMemberships |
| `PUT` | `/crm/v3/lists/{listId}/memberships/add` | AddMembers |
| `PUT` | `/crm/v3/lists/{listId}/memberships/remove` | RemoveMembers |
| `PUT` | `/crm/v3/lists/{listId}/memberships/add-and-remove` | AddAndRemoveMembers |
| `DELETE` | `/crm/v3/lists/{listId}/memberships` | RemoveAllMembers |

### 8.2 Behavior Notes

- **Filter branches stored as opaque JSON** — we do NOT evaluate dynamic list criteria
- **Membership mutations only on MANUAL/SNAPSHOT lists**
- **List names globally unique**
- **Membership pagination**: bi-directional cursors, max 250 per page
- **List search uses offset-based pagination** (not cursor-based)
- **Folder management**: defer to later; not commonly used in tests
- **HubSpot typo in spec**: `MembershipsUpdateResponse` uses `recordsIdsAdded` (extra 's') — we must match this for compatibility
- **Schemas API path**: actual spec uses `/crm-object-schemas/v3/schemas` not `/crm/v3/schemas` — support both
- **Association input/output asymmetry**: input uses `associationCategory`/`associationTypeId`, output uses `category`/`typeId`/`label`

### 8.3 Files to create

| File | Purpose |
|------|---------|
| `internal/api/lists/handler.go` | List CRUD handlers |
| `internal/api/lists/routes.go` | Route registration |
| `internal/store/lists.go` | ListStore interface + SQLite impl |
| `internal/domain/list.go` | List domain types |

---

## 9. Phase 7: Custom Object Schemas

### 9.1 Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/crm/v3/schemas` | ListSchemas |
| `POST` | `/crm/v3/schemas` | CreateSchema |
| `GET` | `/crm/v3/schemas/{objectType}` | GetSchema |
| `PATCH` | `/crm/v3/schemas/{objectType}` | UpdateSchema |
| `DELETE` | `/crm/v3/schemas/{objectType}` | ArchiveSchema |
| `POST` | `/crm/v3/schemas/{objectType}/associations` | CreateSchemaAssociation |
| `DELETE` | `/crm/v3/schemas/{objectType}/associations/{id}` | DeleteSchemaAssociation |

Note: The actual spec path prefix is `/crm-object-schemas/v3/schemas` but HubSpot docs reference `/crm/v3/schemas`. We should support both.

### 9.2 Behavior

When a schema is created:
1. Generate an objectTypeId in format `2-{autoincrement}`
2. Generate a fullyQualifiedName in format `p0_{name}` (using portal 0 since we're a mock)
3. Register the new type in the object type registry
4. Create default property definitions for the schema
5. The new type is now accessible via `/crm/v3/objects/{objectType}` using either the name or ID

### 9.3 Files to create

| File | Purpose |
|------|---------|
| `internal/api/schemas/handler.go` | Schema CRUD handlers |
| `internal/api/schemas/routes.go` | Route registration |
| `internal/store/schemas.go` | SchemaStore interface + SQLite impl |
| `internal/domain/schema.go` | Schema domain types |

---

## 10. Phase 8: Imports & Exports

### 10.1 Imports Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `POST` | `/crm/v3/imports` | StartImport |
| `GET` | `/crm/v3/imports` | ListImports |
| `GET` | `/crm/v3/imports/{importId}` | GetImport |
| `POST` | `/crm/v3/imports/{importId}/cancel` | CancelImport |
| `GET` | `/crm/v3/imports/{importId}/errors` | GetImportErrors |

### 10.2 Exports Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `POST` | `/crm/v3/exports/export/async` | StartExport |
| `GET` | `/crm/v3/exports/export/async/tasks/{taskId}/status` | GetExportStatus |

### 10.3 Behavior

**Imports**: Accept multipart/form-data with CSV file + JSON config. Parse the CSV synchronously (we're a test server, no need for true async). Create/update/upsert objects based on the import config. Track errors.

**Exports**: Accept the export request, immediately generate a CSV/XLSX in memory, store it, return a task ID. Status endpoint returns the download URL when ready.

### 10.4 Files to create

| File | Purpose |
|------|---------|
| `internal/api/imports/handler.go` | Import handlers |
| `internal/api/exports/handler.go` | Export handlers |
| `internal/store/imports.go` | ImportStore |
| `internal/store/exports.go` | ExportStore |

---

## 11. Phase 9: Owners & Users

### 11.1 Endpoints

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/crm/v3/owners` | ListOwners |
| `GET` | `/crm/v3/owners/{ownerId}` | GetOwner |

Owners are read-only in HubSpot's API. For testing, seed some default owners or provide a notspot-specific admin endpoint to create them.

### 11.2 Files to create

| File | Purpose |
|------|---------|
| `internal/api/owners/handler.go` | Owner handlers |
| `internal/store/owners.go` | OwnerStore |
| `internal/seed/owners.go` | Default owners |

---

## 12. Phase 10: Extended APIs

Implement on-demand as testing needs arise. Each follows the same pattern: read spec → create handler + store + domain types.

Candidates in priority order:
1. **Files API** (`/files/v3`) — upload/download/manage files
2. **Marketing Forms** (`/marketing/v3/forms`) — form CRUD + submissions
3. **Communication Preferences** (`/communication-preferences/v3`) — subscription management
4. **CMS HubDB** (`/cms/v3/hubdb`) — structured data tables

---

## 13. Future: Web UI

Design decisions to keep in mind now:

- **All state in SQLite**: The web UI will read from the same database
- **Store layer is the interface boundary**: UI can either call store functions directly (if embedded) or use the REST API
- **Consider adding a notspot admin API** at a different path prefix (e.g., `/_notspot/`) for:
  - Resetting all data
  - Seeding test fixtures
  - Viewing request/response logs
  - Managing owners/users
- **Request logging table**: Log all incoming requests to SQLite for debugging/inspection via UI

---

## 14. SQLite Schema

```sql
-- ============================================================
-- Object Type Registry
-- ============================================================
CREATE TABLE object_types (
    id TEXT PRIMARY KEY,                    -- "0-1", "2-12345"
    name TEXT UNIQUE NOT NULL,              -- "contacts", "p0_my_object"
    label_singular TEXT NOT NULL,
    label_plural TEXT NOT NULL,
    primary_display_property TEXT,
    is_custom BOOLEAN NOT NULL DEFAULT FALSE,
    fully_qualified_name TEXT,              -- "p0_my_object" for custom
    description TEXT,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- ============================================================
-- Property Definitions
-- ============================================================
CREATE TABLE property_definitions (
    object_type_id TEXT NOT NULL,
    name TEXT NOT NULL,
    label TEXT NOT NULL,
    type TEXT NOT NULL,                     -- bool, enumeration, date, datetime, string, number
    field_type TEXT NOT NULL,               -- text, select, number, date, etc.
    group_name TEXT NOT NULL DEFAULT 'contactinformation',
    description TEXT DEFAULT '',
    display_order INTEGER NOT NULL DEFAULT 0,
    has_unique_value BOOLEAN NOT NULL DEFAULT FALSE,
    hidden BOOLEAN NOT NULL DEFAULT FALSE,
    form_field BOOLEAN NOT NULL DEFAULT FALSE,
    calculated BOOLEAN NOT NULL DEFAULT FALSE,
    external_options BOOLEAN NOT NULL DEFAULT FALSE,
    hubspot_defined BOOLEAN NOT NULL DEFAULT FALSE,
    options TEXT,                            -- JSON array of {label, value, displayOrder, hidden}
    calculation_formula TEXT,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (object_type_id, name)
);

-- ============================================================
-- Property Groups
-- ============================================================
CREATE TABLE property_groups (
    object_type_id TEXT NOT NULL,
    name TEXT NOT NULL,
    label TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (object_type_id, name)
);

-- ============================================================
-- Objects (records)
-- ============================================================
CREATE TABLE objects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    object_type_id TEXT NOT NULL,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    archived_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    merged_into_id INTEGER                  -- set when this record was merged into another
);

CREATE INDEX idx_objects_type ON objects(object_type_id, archived);
CREATE INDEX idx_objects_type_created ON objects(object_type_id, created_at);

-- ============================================================
-- Property Values (EAV)
-- ============================================================
CREATE TABLE property_values (
    object_id INTEGER NOT NULL,
    property_name TEXT NOT NULL,
    value TEXT,
    updated_at TEXT NOT NULL,
    source TEXT DEFAULT 'API',
    source_id TEXT,
    PRIMARY KEY (object_id, property_name),
    FOREIGN KEY (object_id) REFERENCES objects(id)
);

CREATE INDEX idx_property_values_value ON property_values(property_name, value);

-- ============================================================
-- Property Value History
-- ============================================================
CREATE TABLE property_value_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    object_id INTEGER NOT NULL,
    property_name TEXT NOT NULL,
    value TEXT,
    timestamp TEXT NOT NULL,
    source TEXT DEFAULT 'API',
    source_id TEXT,
    FOREIGN KEY (object_id) REFERENCES objects(id)
);

CREATE INDEX idx_prop_history ON property_value_history(object_id, property_name, timestamp);

-- ============================================================
-- Association Type Definitions
-- ============================================================
CREATE TABLE association_types (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_object_type TEXT NOT NULL,
    to_object_type TEXT NOT NULL,
    category TEXT NOT NULL,                 -- HUBSPOT_DEFINED, USER_DEFINED, INTEGRATOR_DEFINED
    label TEXT,                             -- nullable for unlabeled
    inverse_label TEXT,                     -- for paired labels
    UNIQUE(from_object_type, to_object_type, category, label)
);

-- ============================================================
-- Association Records
-- ============================================================
CREATE TABLE associations (
    from_object_id INTEGER NOT NULL,
    to_object_id INTEGER NOT NULL,
    association_type_id INTEGER NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (from_object_id, to_object_id, association_type_id),
    FOREIGN KEY (from_object_id) REFERENCES objects(id),
    FOREIGN KEY (to_object_id) REFERENCES objects(id),
    FOREIGN KEY (association_type_id) REFERENCES association_types(id)
);

CREATE INDEX idx_assoc_from ON associations(from_object_id, association_type_id);
CREATE INDEX idx_assoc_to ON associations(to_object_id);

-- ============================================================
-- Pipelines
-- ============================================================
CREATE TABLE pipelines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    object_type_id TEXT NOT NULL,
    label TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- ============================================================
-- Pipeline Stages
-- ============================================================
CREATE TABLE pipeline_stages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pipeline_id INTEGER NOT NULL,
    label TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    metadata TEXT DEFAULT '{}',             -- JSON: {probability, isClosed, ticketState}
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (pipeline_id) REFERENCES pipelines(id)
);

-- ============================================================
-- Lists
-- ============================================================
CREATE TABLE lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    object_type_id TEXT NOT NULL,
    processing_type TEXT NOT NULL,           -- MANUAL, DYNAMIC, SNAPSHOT
    processing_status TEXT NOT NULL DEFAULT 'COMPLETE',
    filter_branch TEXT,                      -- JSON (opaque, not evaluated)
    list_version INTEGER NOT NULL DEFAULT 1,
    folder_id INTEGER,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- ============================================================
-- List Memberships
-- ============================================================
CREATE TABLE list_memberships (
    list_id INTEGER NOT NULL,
    object_id INTEGER NOT NULL,
    added_at TEXT NOT NULL,
    PRIMARY KEY (list_id, object_id),
    FOREIGN KEY (list_id) REFERENCES lists(id),
    FOREIGN KEY (object_id) REFERENCES objects(id)
);

-- ============================================================
-- Imports
-- ============================================================
CREATE TABLE imports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    state TEXT NOT NULL DEFAULT 'STARTED',
    source TEXT DEFAULT 'API',
    opt_out_import BOOLEAN NOT NULL DEFAULT FALSE,
    request_json TEXT,                       -- JSON of the import configuration
    metadata TEXT DEFAULT '{}',              -- JSON of counters, fileIds, objectLists
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- ============================================================
-- Import Errors
-- ============================================================
CREATE TABLE import_errors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    import_id INTEGER NOT NULL,
    error_type TEXT NOT NULL,
    error_message TEXT,
    invalid_value TEXT,
    object_type TEXT,
    line_number INTEGER,
    created_at TEXT NOT NULL,
    FOREIGN KEY (import_id) REFERENCES imports(id)
);

-- ============================================================
-- Exports
-- ============================================================
CREATE TABLE exports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    state TEXT NOT NULL DEFAULT 'ENQUEUED',
    export_type TEXT NOT NULL,               -- VIEW, LIST
    object_type TEXT NOT NULL,
    object_properties TEXT NOT NULL,          -- JSON array
    request_json TEXT,                        -- JSON of export config
    result_data BLOB,                        -- the generated file
    record_count INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- ============================================================
-- Owners
-- ============================================================
CREATE TABLE owners (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    first_name TEXT,
    last_name TEXT,
    user_id INTEGER,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- ============================================================
-- Request Log (for debugging / future web UI)
-- ============================================================
CREATE TABLE request_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    request_body TEXT,
    response_body TEXT,
    duration_ms INTEGER,
    correlation_id TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX idx_request_log_time ON request_log(created_at);
```

---

## 15. Go Package Layout

```
cmd/
  hubspot/
    main.go                     # Entry point: config, DB, router, server

internal/
  config/
    config.go                   # Configuration loading from env

  database/
    database.go                 # SQLite open, migrate (existing)
    database_test.go            # Existing tests
    migrations.go               # Migration SQL embedded strings

  domain/
    object.go                   # Object, CreateInput, UpdateInput, etc.
    property.go                 # Property, PropertyGroup, Option
    pipeline.go                 # Pipeline, PipelineStage
    association.go              # Association, AssociationType
    list.go                     # List, ListMembership
    schema.go                   # ObjectSchema
    import.go                   # Import, ImportError
    export.go                   # Export
    owner.go                    # Owner
    common.go                   # Paging, BatchResult, etc.

  api/
    errors.go                   # APIError type and helpers
    middleware.go               # Auth, recovery, logging, request ID
    response.go                 # JSON response writers
    router.go                   # Route registration orchestrator

    objects/
      handler.go                # CRM object CRUD handlers
      search.go                 # Search handler
      routes.go                 # Route registration

    properties/
      handler.go
      routes.go

    pipelines/
      handler.go
      routes.go

    associations/
      handler.go
      routes.go

    lists/
      handler.go
      routes.go

    schemas/
      handler.go
      routes.go

    imports/
      handler.go
      routes.go

    exports/
      handler.go
      routes.go

    owners/
      handler.go
      routes.go

  store/
    store.go                    # Top-level Store struct composing all sub-stores
    objects.go                  # ObjectStore interface + SQLite impl
    properties.go               # PropertyStore
    pipelines.go                # PipelineStore
    associations.go             # AssociationStore
    lists.go                    # ListStore
    schemas.go                  # SchemaStore
    imports.go                  # ImportStore
    exports.go                  # ExportStore
    owners.go                   # OwnerStore
    search.go                   # Search query builder

  seed/
    seed.go                     # Orchestrator: run all seeders
    object_types.go             # Standard object type definitions
    properties.go               # Default properties per type
    pipelines.go                # Default pipelines
    associations.go             # Standard association type defs
    owners.go                   # Default test owners

  testhelpers/
    testhelpers.go              # Existing test utilities

docs/
  research.md                   # Existing research doc
  plan.md                       # This file
  specs/                        # Downloaded OpenAPI specs
    CRM/
    CMS/
    ...
```

---

## 16. Cross-Cutting Concerns

### 16.1 ID Generation

All IDs are auto-increment integers returned as strings. HubSpot uses string IDs that look like large integers. We match this pattern naturally with SQLite's ROWID.

### 16.2 Timestamp Format

All timestamps use ISO 8601 format: `2024-11-20T20:12:09.236Z`. Use Go's `time.Now().UTC().Format("2006-01-02T15:04:05.000Z")`.

### 16.3 Pagination

Implement two pagination helpers:

1. **Cursor-based** (most endpoints): Cursor is the string representation of the last ID seen. Response includes `paging.next.after` if more results exist.
2. **Offset-based** (list search): Classic offset+count with `hasMore` flag.

### 16.4 Object Type Resolution

Every handler that takes `{objectType}` in the path must resolve it to an internal type ID. Accept both:
- String names: `contacts`, `companies`, `deals`, etc.
- Numeric IDs: `0-1`, `0-2`, `2-12345`, etc.

### 16.5 Property Filtering

When `properties` query param is provided, only include those properties in the response. Always include `hs_object_id`, `hs_createdate`, `hs_lastmodifieddate` as defaults.

### 16.6 Testing Strategy

- **Unit tests**: For each store function, test with in-memory SQLite
- **Integration tests**: Start the HTTP server, make real HTTP requests
- **Test fixtures**: Provide a `testhelpers.SeedTestData()` function that creates standard objects, properties, pipelines for testing
- **Conformance tests**: Eventually, run the same test suite against both notspot and the real HubSpot API to verify behavioral compatibility

### 16.7 Admin API (notspot-specific)

Add endpoints under `/_notspot/` for test control:

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/_notspot/reset` | Drop all data, re-run seeds |
| `GET` | `/_notspot/requests` | View request log |
| `POST` | `/_notspot/seed` | Load test fixture data |

---

## 17. Agent Task Breakdown

When using a team of AI agents to build this, split work along these boundaries:

### Team Structure

| Agent | Responsibility | Phases |
|-------|---------------|--------|
| **Foundation** | Server skeleton, middleware, errors, config, migrations, seed framework | Phase 1 |
| **Objects** | Object CRUD, batch operations, merge, object type registry | Phase 2 |
| **Properties** | Property CRUD, property groups, batch, default seeding | Phase 3a |
| **Pipelines** | Pipeline CRUD, stage CRUD, default seeding | Phase 3b |
| **Associations** | Association record CRUD, label management, batch, seeding | Phase 4 |
| **Search** | Search query builder, filter operators, full-text search | Phase 5 |
| **Lists** | List CRUD, membership management, list search | Phase 6 |
| **Schemas** | Schema CRUD, dynamic type registration | Phase 7 |
| **Imports/Exports** | Import/export async workflows | Phase 8 |

### Dependencies Between Agents

```
Foundation ──→ Objects ──→ Search
    │              │
    │              ├──→ Associations
    │              │
    ├──→ Properties ──→ Schemas
    │
    ├──→ Pipelines
    │
    └──→ Lists (depends on Objects for membership)
```

**Foundation must complete first.** Then Objects + Properties + Pipelines can run in parallel. Associations and Search depend on Objects. Lists depends on Objects. Schemas depends on Properties.

### Per-Agent Deliverables

Each agent must deliver:
1. Store interface + SQLite implementation with tests
2. HTTP handler with route registration
3. Domain types
4. Integration test (HTTP-level)
5. Seed data where applicable

### Shared Conventions (all agents must follow)

- Use the error helpers from `internal/api/errors.go`
- Use the response writers from `internal/api/response.go`
- Register routes via a `RegisterRoutes(mux *http.ServeMux, store *store.Store)` function
- All store functions take `context.Context` as first parameter
- Return domain types from store, convert to API response types in handlers
- Use `slog` for logging
- Follow the existing linter config (`.golangci.yml`)

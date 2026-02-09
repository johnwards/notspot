# HubSpot Public API Complete Inventory and Reimplementation Specification

**The HubSpot platform exposes approximately 60 distinct API areas organized into 15 categories, all documented via OpenAPI 3.0 specs and accessible through a unified REST interface at `api.hubapi.com`.** The CRM is the core: every object type — standard or custom — shares the same CRUD, batch, and search endpoint patterns at `/crm/v3/objects/{objectType}`. Associations form a labeled many-to-many graph connecting all objects. Authentication uses OAuth 2.0 or private app Bearer tokens, with rate limits of **100–250 requests per 10 seconds** depending on tier. This document catalogs every public API, its endpoints, data models, and architectural patterns needed for a clean-room Go + SQLite reimplementation.

---

## 1. OpenAPI spec catalog and primary sources

### The spec catalog endpoint

HubSpot deprecated `GET https://api.hubspot.com/api-catalog-public/v1/apis` on November 11, 2024. The replacement is:

```
GET https://api.hubspot.com/public/api/spec/v1/specs
```

The response is JSON with a `results` array. Each entry contains `name`, `group`, and a `versions` array with `version` (integer), `stage` (`LATEST` or `STABLE`), `requirements` (tier info per hub), `introduction`, `useCase`, and `openApi` (URL to the individual OpenAPI 3.0 JSON spec). The `openApi` URLs follow the pattern `https://api.hubspot.com/public/api/spec/v1/specs/{Group}/{Name}/versions/{rolloutId}`. **Known issue**: some URLs contain unencoded spaces and ampersands that must be URL-encoded before fetching.

### The GitHub spec repository

The repository at `https://github.com/HubSpot/HubSpot-public-api-spec-collection` contains OpenAPI 3.0 JSON files under:

```
PublicApiSpecs/{Category}/{ApiName}/Rollouts/{RolloutId}/v{Version}/{filename}.json
```

The repository is integrated with [HubSpot's Postman workspace](https://www.postman.com/hubspot/hubspot-public-api-workspace/overview) and is auto-synced. HubSpot states these specs are "intended for internal use" (Postman integration), but they are publicly accessible and are the authoritative machine-readable source for all public API schemas. The only directly confirmed rollout ID from search results is **424** for CRM/Products (`PublicApiSpecs/CRM/Products/Rollouts/424/v3/products.json`); rollout IDs are internal versioning numbers that change with updates.

### Complete spec file inventory

The following table enumerates every confirmed API spec, cross-referenced from the spec catalog endpoint response (via changelog examples), the GitHub repository structure, the `clarkmcc/go-hubspot` auto-generated client (which lists packages generated from the catalog), and official HubSpot documentation. The go-hubspot client confirms **42 packages** for the older catalog; the current catalog has expanded to approximately **60 specs**.

| # | Group | API Name | Version(s) | GitHub Path Pattern | Description |
|---|-------|----------|-----------|-------------------|-------------|
| 1 | Account | Account Info | v3 | `PublicApiSpecs/Account/Account Info/...` | Login history, security activity, app usage data |
| 2 | Account | Audit Logs | v3 | `PublicApiSpecs/Account/Audit Logs/...` | Account activity and login history |
| 3 | Auth | OAuth | v1 | `PublicApiSpecs/Auth/Oauth/...` | OAuth token management (v1 deprecated → v3) |
| 4 | Automation | Actions V4 | v4 | `PublicApiSpecs/Automation/Actions/...` | Custom workflow actions |
| 5 | Automation | Sequences | v3 | `PublicApiSpecs/Automation/Sequences/...` | Sales sequences |
| 6 | CMS | Blog Posts (Posts) | v3 | `PublicApiSpecs/CMS/Posts/...` | Blog post CRUD, publishing, scheduling |
| 7 | CMS | Blog Authors | v3 | `PublicApiSpecs/CMS/Authors/...` | Blog author profiles |
| 8 | CMS | Blog Tags | v3 | `PublicApiSpecs/CMS/Tags/...` | Blog tag taxonomy |
| 9 | CMS | Domains | v3 | `PublicApiSpecs/CMS/Domains/...` | Domain management |
| 10 | CMS | HubDB | v3 | `PublicApiSpecs/CMS/HubDB/...` | Structured data tables for CMS |
| 11 | CMS | Pages | v3 | `PublicApiSpecs/CMS/Pages/...` | Site pages and landing pages |
| 12 | CMS | Performance | v3 | `PublicApiSpecs/CMS/Performance/...` | CMS performance metrics (deprecated) |
| 13 | CMS | Site Search | v3 | `PublicApiSpecs/CMS/Site Search/...` | Search indexed CMS content |
| 14 | CMS | Source Code | v3 | `PublicApiSpecs/CMS/Source Code/...` | Design Manager file system CRUD |
| 15 | CMS | URL Redirects | v3 | `PublicApiSpecs/CMS/URL Redirects/...` | URL mapping/redirect management |
| 16 | Commerce | Payments | v3 | `PublicApiSpecs/Commerce/Payments/...` | Payment processing data |
| 17 | Commerce | Invoices | v3 | `PublicApiSpecs/Commerce/Invoices/...` | Invoice management |
| 18 | Conversations | Inbox & Messages | v3 | `PublicApiSpecs/Conversations/Conversations Inbox & Messages/...` | Conversation threads and messages |
| 19 | Conversations | Custom Channels | v3 | `PublicApiSpecs/Conversations/Custom Channels/...` | Custom messaging channels |
| 20 | Conversations | Visitor Identification | v3 | `PublicApiSpecs/Conversations/Visitor Identification/...` | Identify visitors for chat widget |
| 21 | CRM | Associations | v3, v4 | `PublicApiSpecs/CRM/Associations/...` | Object relationship management (v4 is current) |
| 22 | CRM | Companies | v3 | `PublicApiSpecs/CRM/Companies/...` | Company records |
| 23 | CRM | Contacts | v3 | `PublicApiSpecs/CRM/Contacts/...` | Contact records |
| 24 | CRM | Deals | v3 | `PublicApiSpecs/CRM/Deals/...` | Deal records |
| 25 | CRM | Deal Splits | v3 | `PublicApiSpecs/CRM/DealSplits/...` | Deal revenue splitting |
| 26 | CRM | Exports | v3 | `PublicApiSpecs/CRM/Exports/...` | Async data export |
| 27 | CRM | Feedback Submissions | v3 | `PublicApiSpecs/CRM/Feedback Submissions/...` | Feedback survey submissions (read-only) |
| 28 | CRM | Imports | v3 | `PublicApiSpecs/CRM/Imports/...` | Bulk data import |
| 29 | CRM | Line Items | v3 | `PublicApiSpecs/CRM/Line Items/...` | Commerce line items |
| 30 | CRM | Lists | v3 | `PublicApiSpecs/CRM/Lists/...` | CRM segments (static + dynamic) |
| 31 | CRM | Objects | v3 | `PublicApiSpecs/CRM/Objects/...` | Generic CRUD for all object types |
| 32 | CRM | Owners | v3 | `PublicApiSpecs/CRM/Owners/...` | User/owner records |
| 33 | CRM | Pipelines | v3 | `PublicApiSpecs/CRM/Pipelines/...` | Deal/ticket/custom pipeline management |
| 34 | CRM | Products | v3 | `PublicApiSpecs/CRM/Products/Rollouts/424/v3/products.json` | Product catalog |
| 35 | CRM | Properties | v3 | `PublicApiSpecs/CRM/Properties/...` | Property and property group definitions |
| 36 | CRM | Quotes | v3 | `PublicApiSpecs/CRM/Quotes/...` | Sales quotes |
| 37 | CRM | Schemas | v3 | `PublicApiSpecs/CRM/Schemas/...` | Custom object schema definitions |
| 38 | CRM | Tickets | v3 | `PublicApiSpecs/CRM/Tickets/...` | Support tickets |
| 39 | CRM | Timeline | v3 | `PublicApiSpecs/CRM/Timeline/...` | Custom timeline events |
| 40 | CRM | Calls | v3 | `PublicApiSpecs/CRM/Calls/...` | Call engagement records |
| 41 | CRM | Emails | v3 | `PublicApiSpecs/CRM/Emails/...` | Email engagement records |
| 42 | CRM | Meetings | v3 | `PublicApiSpecs/CRM/Meetings/...` | Meeting engagement records |
| 43 | CRM | Notes | v3 | `PublicApiSpecs/CRM/Notes/...` | Note engagement records |
| 44 | CRM | Tasks | v3 | `PublicApiSpecs/CRM/Tasks/...` | Task engagement records |
| 45 | CRM | Communications | v3 | `PublicApiSpecs/CRM/Communications/...` | SMS, WhatsApp, LinkedIn messages |
| 46 | CRM | Postal Mail | v3 | `PublicApiSpecs/CRM/PostalMail/...` | Postal mail engagement records |
| 47 | CRM | Leads | v3 | `PublicApiSpecs/CRM/Leads/...` | Lead records |
| 48 | CRM | Goals | v3 | `PublicApiSpecs/CRM/Goals/...` | Sales goal records |
| 49 | CRM Extensions | Calling | v3 | `PublicApiSpecs/CRM Extensions/Calling/...` | Calling extensions SDK |
| 50 | CRM Extensions | Video Conferencing | v3 | `PublicApiSpecs/CRM Extensions/Videoconferencing/...` | Video conference integrations |
| 51 | CRM Extensions | Accounting | v3 | `PublicApiSpecs/CRM Extensions/Accounting/...` | Accounting/eCommerce extension |
| 52 | CRM Extensions | Cards | v3 | `PublicApiSpecs/CRM Extensions/Cards/...` | CRM extension cards (legacy) |
| 53 | Events | Custom Behavioral Events | v3 | `PublicApiSpecs/Events/Custom Behavioral Events/...` | Send custom events |
| 54 | Events | Events | v3 | `PublicApiSpecs/Events/Events/...` | Event analytics/tracking |
| 55 | Events | Manage Event Definitions | v3 | `PublicApiSpecs/Events/Manage Event Definitions/...` | Event type definitions |
| 56 | Files | Files | v3 | `PublicApiSpecs/Files/Files/...` | File Manager CRUD |
| 57 | Marketing | Transactional Email | v3 | `PublicApiSpecs/Marketing/Transactional/...` | Single-send transactional emails |
| 58 | Marketing | Marketing Events | v3 | `PublicApiSpecs/Marketing/Marketing Events Beta/...` | Marketing event management |
| 59 | Marketing | Forms | v3 | `PublicApiSpecs/Marketing/Forms/...` | Form CRUD and submissions |
| 60 | Marketing | Campaigns | v3 | `PublicApiSpecs/Marketing/Campaigns/...` | Marketing campaigns |
| 61 | Marketing | Communications Status | v3 | `PublicApiSpecs/Marketing/Communications Status/...` | Email subscription preferences |
| 62 | Marketing | Marketing Emails | v3 | `PublicApiSpecs/Marketing/Emails/...` | Marketing email management |
| 63 | Settings | Business Units | v3 | `PublicApiSpecs/Settings/Business Units/...` | Business unit management |
| 64 | Settings | Users | v3 | `PublicApiSpecs/Settings/Users/...` | User provisioning |
| 65 | Settings | Currencies | v3 | `PublicApiSpecs/Settings/Currencies/...` | Currency configuration |
| 66 | Webhooks | Webhooks | v3, v4 | `PublicApiSpecs/Webhooks/Webhooks/...` | Event subscription & journal API |
| 67 | App Mgmt | Feature Flags | v3 | — | Public app feature flag control |

> **⚠ Caveat**: Rollout IDs are dynamic and change with updates. To get the current authoritative file listing, clone the repo or query the live `public/api/spec/v1/specs` endpoint.

---

## 2. CRM Objects API — the unified object interface

### Base URL and endpoint pattern

**Base URL**: `https://api.hubapi.com`

All CRM objects use a single, uniform API surface. Substitute `{objectType}` with either the string name (e.g., `contacts`) or the numeric objectTypeId (e.g., `0-1`).

### Complete endpoint table for CRM Objects v3

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/crm/v3/objects/{objectType}` | List all records (paginated, cursor-based) |
| `POST` | `/crm/v3/objects/{objectType}` | Create a single record |
| `GET` | `/crm/v3/objects/{objectType}/{recordId}` | Read a single record by ID |
| `PATCH` | `/crm/v3/objects/{objectType}/{recordId}` | Update a single record |
| `DELETE` | `/crm/v3/objects/{objectType}/{recordId}` | Archive (soft-delete) a single record |
| `POST` | `/crm/v3/objects/{objectType}/batch/create` | Batch create up to 100 records |
| `POST` | `/crm/v3/objects/{objectType}/batch/read` | Batch read up to 100 records by ID |
| `POST` | `/crm/v3/objects/{objectType}/batch/update` | Batch update up to 100 records |
| `POST` | `/crm/v3/objects/{objectType}/batch/upsert` | Batch create-or-update up to 100 records |
| `POST` | `/crm/v3/objects/{objectType}/batch/archive` | Batch archive up to 100 records |
| `POST` | `/crm/v3/objects/{objectType}/search` | Search/filter records (max 10,000 results) |
| `PUT` | `/crm/v3/objects/{objectType}/{fromId}/associations/{toObjectType}/{toId}/{assocTypeId}` | Create association (v3 style) |
| `DELETE` | `/crm/v3/objects/{objectType}/{fromId}/associations/{toObjectType}/{toId}/{assocTypeId}` | Remove association (v3 style) |

**Authentication**: OAuth 2.0 scopes `crm.objects.{objectType}.read` / `.write`, or private app Bearer token with equivalent scopes.

### All supported object types with objectTypeId

#### Core CRM objects

| Object | objectTypeId | API path alias | Required creation properties |
|--------|-------------|---------------|------------------------------|
| **Contacts** | `0-1` | `contacts` | None (recommended: `email`) |
| **Companies** | `0-2` | `companies` | `domain` or `name` |
| **Deals** | `0-3` | `deals` | `dealname`, `dealstage`, `pipeline` |
| **Tickets** | `0-5` | `tickets` | `subject`, `hs_pipeline_stage`, `hs_pipeline` |

#### Commerce objects

| Object | objectTypeId | API path alias | Required creation properties |
|--------|-------------|---------------|------------------------------|
| **Products** | `0-7` | `products` | `name`, `price` |
| **Line Items** | `0-8` | `line_items` | (varies; typically from product) |
| **Quotes** | `0-14` | `quotes` | `hs_title`, `hs_expiration_date` |
| **Orders** | `0-123` | `orders` | `hs_order_name` |
| **Carts** | `0-142` | `carts` | (none required) |
| **Invoices** | `0-53` | `invoices` | `hs_currency` |
| **Subscriptions** | `0-69` | `subscriptions` | `hs_name` |
| **Payments** | `0-101` | `payments` | `hs_initial_amount`, `hs_initiated_date` |
| **Discounts** | `0-84` | `discounts` | — |
| **Fees** | `0-85` | `fees` | — |
| **Taxes** | `0-86` | `taxes` | — |

#### Engagement/activity objects

| Object | objectTypeId | API path alias | Required properties |
|--------|-------------|---------------|---------------------|
| **Calls** | `0-48` | `calls` | `hs_timestamp` |
| **Emails** | `0-49` | `emails` | `hs_timestamp`, `hs_email_direction` |
| **Meetings** | `0-47` | `meetings` | `hs_timestamp` |
| **Notes** | `0-46` | `notes` | `hs_timestamp` |
| **Tasks** | `0-27` | `tasks` | `hs_timestamp` |
| **Communications** | `0-18` | `communications` | `hs_timestamp`, `hs_communication_channel_type` |
| **Postal Mail** | `0-116` | `postal_mail` | `hs_timestamp` |

#### Other standard objects

| Object | objectTypeId | API path alias | Required properties |
|--------|-------------|---------------|---------------------|
| **Leads** | `0-136` | `leads` | `hs_lead_name` + contact/company fields |
| **Feedback Submissions** | `0-19` | `feedback_submissions` | Read-only; created by HubSpot |
| **Goals** | `0-74` | `goals` | — |
| **Marketing Events** | `0-54` | `marketing_events` | `hs_event_name`, `hs_event_description` |
| **Users** | `0-115` | `users` | `hs_internal_user_id`, `hs_email` |
| **Appointments** | `0-421` | `appointments` | `hs_appointment_name` |
| **Courses** | `0-410` | `courses` | `hs_course_name` |
| **Listings** | `0-420` | `listings` | `hs_name` |
| **Services** | `0-162` | `services` | `hs_name`, `hs_pipeline`, `hs_pipeline_stage` |
| **Projects** | `0-970` | `projects` | `hs_name`, `hs_pipeline`, `hs_pipeline_stage` |
| **Custom Objects** | `2-{schemaId}` | `p{hubId}_{name}` | Defined per schema |

**Standard response shape** for all object reads:

```json
{
  "id": "12345",
  "properties": {
    "hs_object_id": "12345",
    "hs_createdate": "2024-11-20T20:12:09.236Z",
    "hs_lastmodifieddate": "2024-11-20T20:12:10.610Z"
  },
  "createdAt": "2024-11-20T20:12:09.236Z",
  "updatedAt": "2024-11-20T20:12:10.610Z",
  "archived": false
}
```

Properties not explicitly requested via the `properties` query parameter are **not returned**. Only `hs_object_id`, `hs_createdate`, and `hs_lastmodifieddate` are returned by default.

---

## 3. Properties API and property groups

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/crm/v3/properties/{objectType}` | List all properties for an object type |
| `GET` | `/crm/v3/properties/{objectType}/{propertyName}` | Get a single property definition |
| `POST` | `/crm/v3/properties/{objectType}` | Create a new property |
| `PATCH` | `/crm/v3/properties/{objectType}/{propertyName}` | Update a property definition |
| `DELETE` | `/crm/v3/properties/{objectType}/{propertyName}` | Archive a property |
| `GET` | `/crm/v3/properties/{objectType}/groups` | List all property groups |
| `GET` | `/crm/v3/properties/{objectType}/groups/{groupName}` | Get a single group |
| `POST` | `/crm/v3/properties/{objectType}/groups` | Create a property group |
| `PATCH` | `/crm/v3/properties/{objectType}/groups/{groupName}` | Update a group |
| `DELETE` | `/crm/v3/properties/{objectType}/groups/{groupName}` | Archive a group |

### Property type system

| `type` | Description | Valid `fieldType` values |
|--------|-------------|------------------------|
| `bool` | Binary yes/no | `booleancheckbox`, `calculation_equation` |
| `enumeration` | Semicolon-separated option set | `booleancheckbox`, `checkbox`, `radio`, `select`, `calculation_equation` |
| `date` | ISO 8601 date (YYYY-MM-DD) | `date` |
| `datetime` | ISO 8601 date+time | `date` |
| `string` | Plain text (max 65,536 chars) | `file`, `text`, `textarea`, `calculation_equation`, `html`, `phonenumber` |
| `number` | Numeric value | `number`, `calculation_equation` |
| `object_coordinates` | Internal reference (not user-creatable) | `text` |
| `json` | JSON text (internal only) | `text` |

**Required fields for creating a property**: `groupName`, `name`, `label`, `type`, `fieldType`. Up to **10 unique identifier properties** per object (`hasUniqueValue: true`).

**Calculated properties** use `fieldType: calculation_equation` with a `calculationFormula` field supporting arithmetic, comparison, logic operators, and functions (`max`, `min`, `contains`, `concatenate`, `number_to_string`, `string_to_number`, conditionals).

---

## 4. Pipelines API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/crm/v3/pipelines/{objectType}` | List all pipelines |
| `POST` | `/crm/v3/pipelines/{objectType}` | Create a pipeline |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | Get a pipeline |
| `PATCH` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | Update a pipeline |
| `PUT` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | Replace a pipeline |
| `DELETE` | `/crm/v3/pipelines/{objectType}/{pipelineId}` | Delete a pipeline |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages` | List all stages |
| `POST` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages` | Create a stage |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | Get a stage |
| `PATCH` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | Update a stage |
| `PUT` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | Replace a stage |
| `DELETE` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}` | Delete a stage |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}/audit` | Pipeline change history |
| `GET` | `/crm/v3/pipelines/{objectType}/{pipelineId}/stages/{stageId}/audit` | Stage change history |

**Pipeline-enabled objects**: Deals, Tickets, Appointments, Courses, Listings, Orders, Services, Leads, Custom Objects. **Stage limits**: 30 stages for appointments/courses/listings/leads/orders/services; **100 stages** for deals/tickets/custom objects. Deal stages require a `probability` field (0.0–1.0) and `isClosed` boolean. Ticket stages require a `ticketState` of `OPEN` or `CLOSED`.

---

## 5. Associations v4 API — the relationship graph

### Record-level endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/crm/v4/objects/{from}/{fromId}/associations/default/{to}/{toId}` | Associate without label (default) |
| `PUT` | `/crm/v4/objects/{from}/{fromId}/associations/{to}/{toId}` | Associate with label(s) |
| `GET` | `/crm/v4/objects/{from}/{fromId}/associations/{to}` | Get associations for a record |
| `DELETE` | `/crm/v4/objects/{from}/{fromId}/associations/{to}/{toId}` | Remove all associations between two records |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/associate/default` | Bulk associate without labels |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/create` | Bulk create labeled associations (limit: **2,000**) |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/read` | Bulk read associations (limit: **1,000**) |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/archive` | Bulk remove associations |
| `POST` | `/crm/v4/associations/{from}/{to}/batch/labels/archive` | Remove specific labels only |

### Schema/label management endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/crm/v4/associations/{from}/{to}/labels` | Create an association label |
| `GET` | `/crm/v4/associations/{from}/{to}/labels` | List all labels between two types |
| `PUT` | `/crm/v4/associations/{from}/{to}/labels` | Update a label |
| `DELETE` | `/crm/v4/associations/{from}/{to}/labels/{typeId}` | Delete a label |

### Limit configuration endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/crm/v4/associations/definitions/configurations/{from}/{to}/batch/create` | Create association limits |
| `POST` | `/crm/v4/associations/definitions/configurations/{from}/{to}/batch/update` | Update limits |
| `GET` | `/crm/v4/associations/definitions/configurations/all` | Get all limits |
| `GET` | `/crm/v4/associations/definitions/configurations/{from}/{to}` | Get limits between types |
| `POST` | `/crm/v4/associations/definitions/configurations/{from}/{to}/batch/purge` | Delete limits |
| `POST` | `/crm/v4/associations/usage/high-usage-report/{userId}` | Records at ≥80% of limit |

### Association categories and types

Associations are **directional** — Contact→Company has a different `typeId` than Company→Contact.

- **`HUBSPOT_DEFINED`**: System defaults. Includes **Primary** (typeId 1 = Contact→Company Primary, only one allowed) and **Unlabeled** (typeId 279 = Contact→Company, label: `null`).
- **`USER_DEFINED`**: Custom labels created by admins (e.g., "Billing contact", "Decision maker").
- **`INTEGRATOR_DEFINED`**: Labels created by integrations.

**Labels** can be single (same label both directions) or **paired** (different labels per direction, e.g., "Manager" ↔ "Employee", created with `inverseLabel`). Up to **10 labels** per object pairing. An unlabeled association always accompanies any labeled one.

Selected standard association typeIds:

| TypeID | Direction |
|--------|-----------|
| 1 | Contact → Company (Primary) |
| 279 | Contact → Company (Unlabeled) |
| 280 | Company → Contact (Unlabeled) |
| 4 | Contact → Deal |
| 3 | Deal → Contact |
| 5 | Deal → Company (Primary) |
| 15 | Contact → Ticket |
| 13 | Parent → Child Company |
| 14 | Child → Parent Company |
| 449 | Contact → Contact |
| 450 | Company → Company |

---

## 6. CRM Search API

**Endpoint**: `POST /crm/v3/objects/{objectType}/search` — works for all CRM object types.

### Request schema

```json
{
  "query": "<full-text search, max 3000 chars>",
  "filterGroups": [{
    "filters": [{
      "propertyName": "email",
      "operator": "EQ",
      "value": "test@example.com"
    }]
  }],
  "sorts": [{"propertyName": "createdate", "direction": "DESCENDING"}],
  "properties": ["email", "firstname"],
  "limit": 100,
  "after": "0"
}
```

### Filter operators

`EQ`, `NEQ`, `LT`, `LTE`, `GT`, `GTE`, `BETWEEN` (uses `value`+`highValue`), `IN` (uses `values` array; lowercase for strings), `NOT_IN`, `HAS_PROPERTY`, `NOT_HAS_PROPERTY`, `CONTAINS_TOKEN` (supports `*` wildcards), `NOT_CONTAINS_TOKEN`.

### Constraints

- Filters within one `filters` array are **AND'd**; multiple `filterGroups` are **OR'd**
- Max **5 filter groups**, **6 filters per group**, **18 filters total**
- **One sort rule** per request
- Default page size: **10**; max: **200** per page
- **10,000 total results** hard cap (400 error beyond)
- **Rate limit: 5 requests/second per account**
- `after` cursor must be formatted as integer
- Association filtering via pseudo-property `associations.{objectType}` with `EQ` and a record ID

---

## 7. CMS APIs endpoint inventory

### Blog Posts (`/cms/v3/blogs/posts`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/cms/v3/blogs/posts` | List all blog posts (filterable, sortable) |
| `POST` | `/cms/v3/blogs/posts` | Create a blog post |
| `GET` | `/cms/v3/blogs/posts/{postId}` | Get a blog post |
| `PATCH` | `/cms/v3/blogs/posts/{postId}` | Update a blog post |
| `DELETE` | `/cms/v3/blogs/posts/{postId}` | Delete a blog post |
| `POST` | `/cms/v3/blogs/posts/{postId}/clone` | Clone a blog post |
| `POST` | `/cms/v3/blogs/posts/{postId}/draft/push-live` | Publish draft to live |
| `POST` | `/cms/v3/blogs/posts/{postId}/draft/reset` | Reset draft to live version |
| `POST` | `/cms/v3/blogs/posts/schedule` | Schedule a post for later publish |
| `POST` | `/cms/v3/blogs/posts/batch/create` | Batch create posts |
| `POST` | `/cms/v3/blogs/posts/batch/update` | Batch update posts |
| `POST` | `/cms/v3/blogs/posts/batch/archive` | Batch delete posts |
| `POST` | `/cms/v3/blogs/posts/multi-language/create-language-variation` | Create language variant |
| `POST` | `/cms/v3/blogs/posts/multi-language/attach-to-lang-group` | Attach to multi-language group |
| `POST` | `/cms/v3/blogs/posts/multi-language/detach-from-lang-group` | Detach from group |

Blog Authors and Tags follow the same pattern at `/cms/v3/blogs/authors` and `/cms/v3/blogs/tags` respectively, with CRUD, batch, and multi-language endpoints. Blog settings are at `GET /cms/v3/blog-settings/settings`.

### HubDB (`/cms/v3/hubdb`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/cms/v3/hubdb/tables` | List all tables |
| `POST` | `/cms/v3/hubdb/tables` | Create a table |
| `GET` | `/cms/v3/hubdb/tables/{tableIdOrName}` | Get published table |
| `GET` | `/cms/v3/hubdb/tables/{tableIdOrName}/draft` | Get draft version |
| `PATCH` | `/cms/v3/hubdb/tables/{tableIdOrName}/draft` | Update draft table |
| `DELETE` | `/cms/v3/hubdb/tables/{tableIdOrName}` | Delete a table |
| `POST` | `/cms/v3/hubdb/tables/{tableIdOrName}/draft/clone` | Clone a table |
| `POST` | `/cms/v3/hubdb/tables/{tableIdOrName}/draft/push-live` | Publish draft |
| `POST` | `/cms/v3/hubdb/tables/{tableIdOrName}/draft/reset` | Reset draft to live |
| `GET` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows` | Get published rows |
| `GET` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows/draft` | Get draft rows |
| `POST` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows` | Create a row |
| `GET` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows/{rowId}` | Get a row |
| `PATCH` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows/{rowId}/draft` | Update a draft row |
| `DELETE` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows/{rowId}/draft` | Delete a draft row |
| `POST` | `/cms/v3/hubdb/tables/{tableIdOrName}/rows/batch/create` | Batch create rows |
| `POST` | `/cms/v3/hubdb/tables/{tableIdOrName}/import` | Import rows from CSV |

HubDB supports **draft/published versioning**, dynamic page generation (via `useForPage: true`), public unauthenticated access (via `portalId` query param), foreign key joins between tables, and offset-based pagination for rows.

### Source Code (`/cms/v3/source-code`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/cms/v3/source-code/{env}/content/{path}` | Download file (binary, `Accept: application/octet-stream`) |
| `PUT` | `/cms/v3/source-code/{env}/content/{path}` | Upload/update file (multipart/form-data) |
| `DELETE` | `/cms/v3/source-code/{env}/content/{path}` | Delete file |
| `GET` | `/cms/v3/source-code/{env}/metadata/{path}` | Get file/folder metadata |
| `POST` | `/cms/v3/source-code/{env}/validate/{path}` | Validate file (HubL syntax) |
| `POST` | `/cms/v3/source-code/extract/{path}` | Extract uploaded zip file |

`{env}` is either `draft` (unpublished) or `published` (live).

### Other CMS endpoints

- **Pages**: `/cms/v3/pages/site-pages` and `/cms/v3/pages/landing-pages` — CRUD, batch, clone, publish, schedule, multi-language
- **URL Redirects**: `/cms/v3/url-redirects` — CRUD for URL mappings
- **Domains**: `/cms/v3/domains` — List and read domains
- **Site Search**: `/cms/v3/site-search/search` — Search indexed CMS content

---

## 8. Marketing, events, and transactional APIs

### Transactional Email

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/marketing/v3/transactional/single-email/send` | Send a single transactional email |

Requires `emailId` (template), `message.to`, optional `contactProperties`, `customProperties`.

### Marketing Events

Standard CRUD at `/marketing/v3/marketing-events-beta/events` with attendance tracking, participant management, and `POST .../email-upsert` for upserting by email.

### Forms

CRUD at `/marketing/v3/forms` with form definition management. Form submissions via `POST /marketing/v3/forms/{formId}/submissions`.

### Communication Preferences

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/communication-preferences/v3/definitions` | Get subscription types |
| `GET` | `/communication-preferences/v3/status/email/{email}` | Get subscription status for email |
| `POST` | `/communication-preferences/v3/subscribe` | Subscribe a contact |
| `POST` | `/communication-preferences/v3/unsubscribe` | Unsubscribe a contact |

### Custom Behavioral Events

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/events/v3/send` | Send a custom event completion (rate limit: **1,250/sec**) |
| `GET` | `/events/v3/event-definitions` | List event definitions |
| `POST` | `/events/v3/event-definitions` | Create event definition |

---

## 9. Webhooks API — v3 (push) and v4 (journal/pull)

### v3 Webhooks (push-based, per-app)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/webhooks/v3/{appId}/settings` | Get webhook settings |
| `PUT` | `/webhooks/v3/{appId}/settings` | Update webhook URL and throttling |
| `GET` | `/webhooks/v3/{appId}/subscriptions` | List all subscriptions |
| `POST` | `/webhooks/v3/{appId}/subscriptions` | Create a subscription |
| `GET` | `/webhooks/v3/{appId}/subscriptions/{subscriptionId}` | Get a subscription |
| `PATCH` | `/webhooks/v3/{appId}/subscriptions/{subscriptionId}` | Update a subscription |
| `DELETE` | `/webhooks/v3/{appId}/subscriptions/{subscriptionId}` | Delete a subscription |

**Event types** (legacy format): `contact.creation`, `contact.deletion`, `contact.propertyChange`, `company.creation`, `company.deletion`, `company.propertyChange`, `deal.creation`, `deal.deletion`, `deal.propertyChange`, `ticket.creation`, `ticket.deletion`, `ticket.propertyChange`, `contact.privacyDeletion`, plus merge, restore, and association change events.

**Generic webhook subscriptions** (newer format) support all CRM object types including custom objects, with subscription types: `object.creation`, `object.deletion`, `object.propertyChange`, `object.merge`, `object.restore`, `object.associationChange`.

Up to **1,000 subscriptions per app**. HubSpot retries failed deliveries up to **10 times over 24 hours**.

### v4 Webhooks (journal/pull-based, per-portal)

Managed via `/webhooks/v4/subscriptions`. Instead of push, your app **polls a journal** to fetch changes on your schedule. Pages through "journal files" using offsets. Historical changes available for up to **3 days**. Includes snapshot endpoints for current object state. Best for high-scale, enterprise, or analytics-driven integrations.

---

## 10. Authentication — OAuth 2.0 and private apps

### OAuth 2.0 authorization code flow

**Authorization URL**: `https://app.hubspot.com/oauth/authorize?client_id={ID}&scope={SCOPES}&redirect_uri={URI}&optional_scope={OPT}&state={STATE}`

**Token exchange** (current): `POST https://api.hubapi.com/oauth/v3/token` with `grant_type=authorization_code`, `client_id`, `client_secret`, `redirect_uri`, `code`. Returns `access_token` (expires in **30 minutes**), `refresh_token`.

**Token refresh**: Same endpoint with `grant_type=refresh_token` and `refresh_token`.

**Token introspection**: `POST https://api.hubapi.com/oauth/v3/introspect`

### Private app tokens

Created in HubSpot account settings. **Do not expire**. Used via `Authorization: Bearer {TOKEN}` header. Up to **20 private apps** per account. Scopes selected at creation time.

### Key OAuth scopes

- **CRM objects**: `crm.objects.{type}.read/.write` for contacts, companies, deals, tickets, line_items, quotes, custom, leads, orders, carts, invoices, subscriptions, feedback_submission, goals, owners, users, marketing_events, appointments, courses, listings, services, projects, commercepayments
- **Sensitive data**: `crm.objects.{type}.sensitive.read/.write` and `.highly_sensitive.read/.write` (Enterprise)
- **Schemas**: `crm.schemas.{type}.read/.write`
- **CRM features**: `crm.lists.read/.write`, `crm.import`, `crm.export`, `crm.dealsplits.read_write`
- **CMS**: `content`, `cms.domains.read/.write`, `cms.functions.read/.write`, `hubdb`
- **Marketing**: `marketing-email`, `marketing.campaigns.read/.write`, `forms`, `forms-uploaded-files`
- **Automation**: `automation`, `automation.sequences.read`, `automation.sequences.enrollments.write`
- **Other**: `files`, `conversations.read/.write`, `settings.users.read/.write`, `account-info.security.read`, `analytics.behavioral_events.send`, `communication_preferences.read/.write`, `e-commerce`, `social`

---

## 11. Rate limiting rules

### Per-app burst limits

| Tier | Requests / 10 seconds | Daily limit |
|------|----------------------|-------------|
| Free / Starter | 100 | 250,000 |
| Professional | 190 | 625,000 |
| Enterprise | 190 | 1,000,000 |
| + API Limit Increase add-on | 250 | +1,000,000 (max 2 purchases) |
| Publicly distributed OAuth app | 110/10s per account | — |

### Per-endpoint overrides

- **CRM Search**: **5 requests/second** per account
- **Associations**: 500K daily (Pro/Enterprise); burst same as tier limits
- **Form submissions** (unauthenticated): 50/10s
- **Custom event completions**: 1,250/second

### Response headers

`X-HubSpot-RateLimit-Daily`, `X-HubSpot-RateLimit-Daily-Remaining`, `X-HubSpot-RateLimit-Interval-Milliseconds`, `X-HubSpot-RateLimit-Max`, `X-HubSpot-RateLimit-Remaining`. Search API responses do **not** include rate limit headers.

**429 response**: includes `policyName` (`DAILY` or `TEN_SECONDLY_ROLLING`), `correlationId`. Daily limit resets at midnight in account timezone.

---

## 12. Error response format and pagination

### Standard error response

```json
{
  "status": "error",
  "message": "Human readable message",
  "errors": [{
    "message": "Specific error detail",
    "code": "INVALID_INTEGER",
    "in": "fieldname",
    "context": {"propertyName": ["discount"]},
    "subCategory": "<string>"
  }],
  "category": "VALIDATION_ERROR",
  "correlationId": "a43683b0-..."
}
```

**Key HTTP codes**: 200 (success), 207 (multi-status batch), 400 (bad request), 401 (unauthorized), 403 (forbidden), 414 (too many identities), **423** (locked — retry after 2s for concurrent upserts), 429 (rate limit), **477** (migration in progress — respect Retry-After), 502/504 (timeout).

**Error categories**: `VALIDATION_ERROR`, `RATE_LIMITS`, `CONFLICT`, `NOT_FOUND`, `OBJECT_NOT_FOUND`.

### Pagination

**Cursor-based** (primary pattern): response includes `paging.next.after`; pass as `after` parameter. If absent, no more results. List endpoints default to 10 results, max 100. Search endpoints max 200 per page.

**Offset-based** (Lists API, some legacy endpoints): response includes `offset`, `count`, `hasMore`.

---

## 13. Imports and exports API

### Imports (`/crm/v3/imports`)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/crm/v3/imports` | Start import (multipart/form-data) |
| `GET` | `/crm/v3/imports/` | List all imports |
| `GET` | `/crm/v3/imports/{importId}` | Get import status |
| `POST` | `/crm/v3/imports/{importId}/cancel` | Cancel an import |
| `GET` | `/crm/v3/imports/{importId}/errors` | Get import errors |

Supports CSV and Excel. Import operations: `CREATE`, `UPDATE`, `UPSERT`. **Limits: 80M rows/day**, 1,048,576 rows or 512 MB per file. Import states: `STARTED`, `PROCESSING`, `DONE`, `FAILED`, `CANCELED`, `DEFERRED`.

### Exports (`/crm/v3/exports`)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/crm/v3/exports/export/async` | Start async export |
| `GET` | `/crm/v3/exports/export/async/tasks/{exportId}/status` | Get export status |

Supports XLSX, CSV, XLS. Max **30 exports per 24h**. Auto-splits at 1M rows. Download URL expires after **5 minutes**. Requires `crm.export` scope + Super Admin for OAuth.

---

## 14. Lists API (segments)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/crm/v3/lists` | Create a list |
| `GET` | `/crm/v3/lists/{listId}` | Get list by ID |
| `PUT` | `/crm/v3/lists/{listId}/update-list-name` | Rename a list |
| `PUT` | `/crm/v3/lists/{listId}/update-list-filters` | Update list filter criteria |
| `DELETE` | `/crm/v3/lists/{listId}` | Delete a list |
| `POST` | `/crm/v3/lists/search` | Search lists |
| `GET` | `/crm/v3/lists/{listId}/memberships` | Get list members |
| `PUT` | `/crm/v3/lists/{listId}/memberships/add` | Add records (static only) |
| `PUT` | `/crm/v3/lists/{listId}/memberships/remove` | Remove records (static only) |

**Processing types**: `MANUAL` (static — manual add/remove only), `DYNAMIC` (active — auto-processed by filter criteria), `SNAPSHOT` (point-in-time with branching). Supports contacts, companies, deals, tickets, and custom objects. v1 Contact Lists API sunsetting **April 30, 2026**.

---

## 15. Other API areas

### Files API (`/files/v3`)

Standard CRUD for file management: upload, download, delete, search, get by ID, manage folders.

### Owners API (`/crm/v3/owners`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/crm/v3/owners` | List all owners |
| `GET` | `/crm/v3/owners/{ownerId}` | Get owner by ID |

### Schemas API (Custom Objects) (`/crm/v3/schemas`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/crm/v3/schemas` | List all custom object schemas |
| `POST` | `/crm/v3/schemas` | Create a custom object schema |
| `GET` | `/crm/v3/schemas/{objectTypeId}` | Get a specific schema |
| `PATCH` | `/crm/v3/schemas/{objectTypeId}` | Update schema |
| `DELETE` | `/crm/v3/schemas/{objectTypeId}` | Delete schema (all records must be deleted first) |

Schema definition includes: `name`, `labels` (singular/plural), `primaryDisplayProperty`, `secondaryDisplayProperties`, `requiredProperties`, `searchableProperties`, `associatedObjects`, `properties[]`. Max **10 custom objects** per Enterprise account. Custom objects use the same standard CRUD API.

### Object Library API

`GET /crm/v3/object-library` — Check which objects are activated in an account. Newer objects (appointments, courses, listings, services) must be activated before use.

### Settings APIs

- **Business Units**: `/crm/v3/business-units/` — Manage business units
- **Users**: `/settings/v3/users` — User provisioning CRUD
- **Currencies**: `/settings/v3/currencies` — Currency management
- **Account Info**: `/account-info/v3/details` — Account details, activity

### CRM Extensions

- **Calling**: `/crm/v3/extensions/calling/{appId}/settings` — Register calling provider
- **Video Conferencing**: `/crm/v3/extensions/videoconferencing/settings/{appId}` — Register conferencing provider
- **Accounting**: `/crm/v3/extensions/accounting` — Accounting sync extension

---

## 16. Architectural patterns for reimplementation

### The unified object model

Every HubSpot CRM entity — contacts, deals, custom objects, even engagements like calls and emails — is a **generic CRM object** accessed via the same API surface at `/crm/v3/objects/{objectType}`. The `{objectType}` is either a string name or a numeric ID (`0-{n}` for standard, `2-{n}` for custom). This means a reimplementation needs a single, generic object storage layer. Key points:

- **Properties are schemaless per object type**: each object type has its own property definitions, managed via `/crm/v3/properties/{objectType}`. There is no fixed schema — properties are created/modified at runtime.
- **All objects share the same response shape**: `id`, `properties` (key-value map), `createdAt`, `updatedAt`, `archived`.
- **Unique identifiers**: Up to 10 unique-value properties per object. Contacts default-identify by `email`, companies by `domain`.

### The association graph

Associations are a **directed, labeled, many-to-many graph**:

- Every pair of object types can have multiple association type definitions (each with a `typeId`, `category`, and optional `label`).
- Associations are directional: A→B has a different typeId than B→A.
- A single record pair can have multiple concurrent labels plus an unlabeled default.
- The "primary company" concept is implemented as a specific labeled association type.
- Same-object associations are supported (Company→Company for parent/child, Contact→Contact for relationships).
- For SQLite reimplementation: model as an `associations` table with columns `from_object_type`, `from_id`, `to_object_type`, `to_id`, `association_type_id`, `category`, `label`.

### Lifecycle stages and pipeline mechanics

**Lifecycle stages** on contacts/companies are an ordered enumeration property (`lifecyclestage`) with default values: `subscriber` → `lead` → `marketingqualifiedlead` → `salesqualifiedlead` → `opportunity` → `customer` → `evangelist` → `other`. Stages are **progressive** — going backward requires explicit override. Automatic progression: `opportunity` set when contact associated with deal; `customer` set when deal closed-won.

**Deal pipelines** must contain Won and Lost stages. Stage values use **internal IDs** (not labels). `closedate` auto-updates on close. `hs_deal_stage_probability` is auto-calculated.

### Record merge rules

Merged records store previous IDs in `hs_merged_object_ids`. Old IDs can be used to update the surviving record via the single-record endpoint, but **not** in batch endpoints.

### Engagements as first-class objects

Calls, emails, meetings, notes, tasks, communications, and postal mail are CRM objects with their own objectTypeIds. They're created via the standard Objects API, then associated to contacts/companies/deals via the Associations API. They appear on record timelines ordered by `hs_timestamp`.

### Workflow/automation structure

Workflows are composed of:

1. **Triggers**: filter-based (property criteria), event-based (form submit, page view, custom event), or manual enrollment
2. **Actions**: set property, send email, create task, delay, if/then branch, webhook call, enroll in workflow, custom code (Node.js/Python), custom app actions (external `actionUrl` callback)
3. **Custom actions** (app extensions): defined with `actionUrl`, `objectTypes`, `inputFields`, `outputFields`, `functions` (Lambda transforms), `labels`

The v4 Automation API provides full CRUD for workflows including batch operations, updated branching, and support for all CRM object types.

### CMS content model

The CMS follows a **Theme → Template → Module** hierarchy:

- **Themes**: directory containing templates, modules, CSS, JS, images. Configured via `theme.json` and `fields.json`. Max 50 templates, 50 modules, 50 sections per theme.
- **Templates**: define page layout using HTML + HubL (HubSpot's templating language). Types: page, blog listing, blog post, email, system (error, search, subscription prefs).
- **Modules**: reusable components with configurable fields (boolean, text, richtext, image, link, color, choice, number, date, HubDB table, etc.) and HTML+HubL+CSS+JS. Accessed via `{{ module.field_name }}`.
- **HubDB**: relational tables powering dynamic pages. Tables have draft/published states. Rows map to dynamic URLs via `hs_path` column. Column types include text, number, date, select, multi-select, URL, image, video, location (geo), foreign ID (joins).
- **Content staging**: all CMS content supports draft/published workflow with push-live and reset-to-live operations.

### Timeline events (partner app integration)

The Timeline API (v3) lets public app partners display custom events on CRM record timelines. Event templates define structure (tokens/fields, header/detail templates). Up to **750 event types per app**, **500 properties per type**. Events are immutable after creation. Custom objects are **not supported** by the Timeline API.

---

## 17. Items not fully documented in public sources

The following areas have **incomplete or absent public documentation** and should be flagged for the reimplementation:

- **Exact rollout IDs** for all spec files in the GitHub repo — only 424 (Products) confirmed; the rest require cloning the repo or querying the live spec endpoint
- **Analytics API** — referenced in CMS docs and overview pages but no dedicated spec file confirmed in the repository
- **GraphQL API** — mentioned as having its own complexity limits but no OpenAPI spec exists (it uses GraphQL schema, not REST)
- **Marketing Campaigns API** — listed in docs overview but sparse endpoint documentation
- **Conversations v3 Inbox & Messages** — confirmed spec exists but detailed endpoint listing requires fetching the individual OpenAPI spec file
- **Custom Channels API** — newer addition with limited public documentation
- **v4 Webhooks Journal API** — endpoint paths confirmed (`/webhooks/v4/subscriptions`) but the journal polling mechanics and snapshot endpoints have limited public documentation
- **Property Validations API** — endpoints exist (`/crm/v3/property-validations/{objectTypeId}`) but are not prominently documented
- **Object Library API** — `GET /crm/v3/object-library` mentioned but limited docs
- **Sensitive/Highly Sensitive scopes** — documented at scope level but data handling rules not fully specified

---

## 18. Recommended reimplementation data model (Go + SQLite)

Based on the architectural patterns above, a clean-room reimplementation should center on these core SQLite tables:

```sql
-- Object type definitions (standard + custom)
CREATE TABLE object_types (
  id TEXT PRIMARY KEY,              -- "0-1", "2-12345"
  name TEXT UNIQUE NOT NULL,         -- "contacts", "p12345_my_object"
  label_singular TEXT,
  label_plural TEXT,
  primary_display_property TEXT,
  is_custom BOOLEAN DEFAULT FALSE
);

-- Property definitions per object type
CREATE TABLE property_definitions (
  object_type_id TEXT NOT NULL,
  name TEXT NOT NULL,
  label TEXT,
  type TEXT NOT NULL,                -- bool, enumeration, date, datetime, string, number
  field_type TEXT NOT NULL,          -- text, select, number, date, etc.
  group_name TEXT,
  has_unique_value BOOLEAN DEFAULT FALSE,
  calculation_formula TEXT,
  options TEXT,                       -- JSON array for enumeration options
  PRIMARY KEY (object_type_id, name)
);

-- Generic object records
CREATE TABLE objects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  object_type_id TEXT NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  archived BOOLEAN DEFAULT FALSE,
  archived_at DATETIME
);

-- Property values (EAV pattern)
CREATE TABLE property_values (
  object_id INTEGER NOT NULL,
  property_name TEXT NOT NULL,
  value TEXT,
  updated_at DATETIME,
  PRIMARY KEY (object_id, property_name)
);

-- Association graph
CREATE TABLE associations (
  from_object_type TEXT NOT NULL,
  from_object_id INTEGER NOT NULL,
  to_object_type TEXT NOT NULL,
  to_object_id INTEGER NOT NULL,
  association_type_id INTEGER NOT NULL,
  category TEXT NOT NULL,            -- HUBSPOT_DEFINED, USER_DEFINED, INTEGRATOR_DEFINED
  label TEXT
);

-- Pipeline definitions
CREATE TABLE pipelines (
  id TEXT PRIMARY KEY,
  object_type_id TEXT NOT NULL,
  label TEXT NOT NULL,
  display_order INTEGER
);

CREATE TABLE pipeline_stages (
  id TEXT PRIMARY KEY,
  pipeline_id TEXT NOT NULL,
  label TEXT NOT NULL,
  display_order INTEGER,
  probability REAL,                  -- for deal stages
  is_closed BOOLEAN,
  ticket_state TEXT                   -- OPEN or CLOSED for ticket stages
);

-- Lists/segments
CREATE TABLE lists (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  object_type_id TEXT NOT NULL,
  processing_type TEXT NOT NULL,      -- MANUAL, DYNAMIC, SNAPSHOT
  filter_branch TEXT                  -- JSON filter definition
);

CREATE TABLE list_memberships (
  list_id INTEGER NOT NULL,
  object_id INTEGER NOT NULL,
  PRIMARY KEY (list_id, object_id)
);
```

This schema mirrors HubSpot's unified object model: every object type shares the same storage pattern, properties are defined dynamically per type, and associations form a labeled directed graph. The EAV (Entity-Attribute-Value) pattern for properties matches HubSpot's behavior where properties are schemaless and queried by name.

---

## Conclusion

HubSpot's public API surface is large but architecturally consistent. **The single most important pattern to replicate is the unified CRM object model** — every object type, from contacts to custom objects to engagement activities, shares the same CRUD/batch/search interface at `/crm/v3/objects/{objectType}`. The association graph provides the relationship layer. Properties are dynamic and type-defined. The entire CRM can be modeled as: objects + properties + associations + pipelines.

Beyond the CRM core, the platform layers on CMS (blog, pages, HubDB, source code), Marketing (transactional email, forms, events, subscriptions), Automation (workflow CRUD, custom actions, sequences), Commerce (orders, invoices, payments), and Conversations (inbox, custom channels). Each has its own API surface but follows consistent patterns: REST + JSON, Bearer auth, cursor-based pagination, standard error format, rate limits per the documented tiers.

For a complete machine-readable specification, **clone the GitHub repo** (`git clone https://github.com/HubSpot/HubSpot-public-api-spec-collection.git`) and fetch the live catalog (`GET https://api.hubspot.com/public/api/spec/v1/specs`). Cross-reference with the reference docs at `developers.hubspot.com/docs/api-reference/` for each API area. The go-hubspot project at `github.com/clarkmcc/go-hubspot` provides a working example of generating Go clients from these specs.
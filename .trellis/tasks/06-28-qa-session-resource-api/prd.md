# PRD: QA Session Resource API

## Goal

Implement task DLQA-169 / 1.1 on top of the administrator-provided QA service
framework in `upstream/develop`.

The feature owns the current-user QA session resource API:

- `POST /api/v1/qa-sessions`
- `GET /api/v1/qa-sessions`
- `GET /api/v1/qa-sessions/{sessionId}`
- `PATCH /api/v1/qa-sessions/{sessionId}`
- `DELETE /api/v1/qa-sessions/{sessionId}`

The concrete service implementation uses the QA service internal routes under
`/internal/v1/**`; gateway remains responsible for the public `/api/v1/**`
entrypoint and for injecting trusted identity headers.

## Source Requirements

- Latest uploaded interface document: `D:/download/README.md`.
- Gateway contract: `docs/services/gateway/api/openapi.yaml`.
- QA behavior document: `docs/services/qa/README.md`.
- QA service framework: `services/qa`.
- Backend rules: `.trellis/spec/backend/`.

## Contract Requirements

- Require gateway-injected `X-User-Id` for business routes.
- Require `X-Service-Token` for QA internal `/internal/v1/**` routes.
- Echo or generate `X-Request-Id`.
- Use success envelope `{data, requestId}`.
- Use paginated envelope `{data, page, requestId}`.
- Use error envelope `{error:{code,message,requestId,fields?}}`.
- Return camelCase `QASession` fields: `id`, `title`, `status`,
  `messageCount`, `lastMessagePreview`, `createdAt`, and `updatedAt`.

## Session List Parameters

`GET /api/v1/qa-sessions` supports only the parameters defined by the latest
uploaded README:

- `page`, default `1`;
- `pageSize`, default `20`;
- `status`, default `active`, allowed `active` or `archived`;
- `sort`, default `-updatedAt`.

The latest document does not define a `q` parameter, so the implementation and
Gateway OpenAPI must not expose session text filtering in this task.

## Data Requirements

- Store sessions in `conversations`.
- Isolate users by `conversations.external_user_id`.
- Soft-deleted sessions have `deleted_at` set and are hidden from list/detail,
  update, delete, and child resources.
- Aggregate `messageCount` from `messages`.
- Build `lastMessagePreview` from the latest displayable
  `message_content_blocks` entry.

## Behavior Requirements

1. Create session
   - Accept optional `title`.
   - Create an `active` session for the current user.
   - Return `201`.

2. List sessions
   - Return only the current user's non-deleted sessions.
   - Filter by `status`.
   - Sort by documented stable sort values.
   - Return pagination metadata.

3. Get session
   - Return the current user's non-deleted session.
   - Return `403 forbidden` for another user's active session without exposing
     resource data.
   - Return `404 not_found` for missing or deleted sessions.

4. Update session
   - Update `title` and/or `status`.
   - Allow status values `active` and `archived`.
   - Apply the same ownership and deletion rules as detail.

5. Delete session
   - Soft-delete the session.
   - Return `204` with no response body.
   - Apply the same ownership and deletion rules as detail.

## Acceptance Criteria

- A logged-in user can create and fetch their own session.
- The list endpoint returns only the current user's sessions.
- The list endpoint supports `page`, `pageSize`, `status`, and `sort` exactly as
  documented.
- The list endpoint includes `messageCount`, `lastMessagePreview`, and
  pagination metadata.
- Renaming and archiving a session returns the updated camelCase DTO.
- Deleting a session hides it from list and detail requests.
- Cross-user detail, update, and delete return `403` without session data.
- Missing or deleted sessions return `404`.
- `go test ./...` passes under `services/qa`.
- `go build -buildvcs=false ./cmd/server` passes under `services/qa`.

## Out Of Scope

- Adding or exposing a `q` text filter.
- Replacing the administrator-provided QA service framework.
- Gateway routing implementation.
- Auth user/role CRUD.
- Message creation, SSE, citations, config, metrics, and retrieval APIs beyond
  what already exists in the upstream QA framework.

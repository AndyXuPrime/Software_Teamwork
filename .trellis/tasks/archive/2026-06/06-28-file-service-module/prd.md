# Implement File Service Module

## Goal

Create the first runnable `services/file` Go service module so gateway and future backend services have a concrete file-service target to integrate with. The MVP should follow the existing file-owned contract for upload, metadata read/update, deletion, content lookup, health checks, request IDs, and stable error envelopes.

## What I Already Know

* Backend services live under `services/<service>/` as independent Go modules.
* `docs/file.md` is the file service contract source for this task.
* Active file-owned public gateway routes are documented in `docs/api/gateway.openapi.yaml`.
* File service owns raw upload, file-owned metadata, object storage coordination, content lookup, and deletion.
* Knowledge ingestion, public document list/detail, chunks, parsing, embedding, and processing state transitions are outside this task until `knowledge` contracts are finalized.
* User explicitly wants the file module built first.
* User explicitly does not want a local MinIO setup now; the service should leave an object-storage port for MinIO integration later.
* User explicitly wants missing related service interfaces added in a RESTful shape when they are needed for later integration.

## Requirements

* Add `services/file` as an independent Go module with service-local `go.mod`.
* Add service-local structure matching backend specs:
  * `cmd/server`
  * `internal/config`
  * `internal/http`
  * `internal/service`
  * `internal/repository`
  * `internal/platform/storage`
  * `api`
  * `migrations`
* Implement `GET /healthz` and `GET /readyz`.
* Implement internal file-service routes shaped for gateway forwarding and future internal integration:
  * `POST /internal/v1/knowledge-bases/{knowledgeBaseId}/documents`
  * `GET /internal/v1/documents/{documentId}`
  * `PATCH /internal/v1/documents/{documentId}`
  * `DELETE /internal/v1/documents/{documentId}`
  * `GET /internal/v1/documents/{documentId}/content`
* Keep new internal routes RESTful and resource-oriented.
* Do not add active public gateway OpenAPI operations for knowledge-owned missing contracts such as public document detail, document list, chunks, or processing state.
* Return stable JSON success and error envelopes for JSON responses.
* Preserve binary success response behavior for content lookup.
* Accept gateway context headers where present:
  * `X-Request-Id`
  * `X-User-Id`
  * `X-User-Roles`
  * `X-User-Permissions`
  * `X-Forwarded-For`
  * `X-Forwarded-Proto`
* Validate request shape and basic file attributes at service boundary.
* Implement file-owned document metadata model with public string IDs, RFC3339 timestamps, tags, content type, and size.
* Implement repository and storage as interfaces so PostgreSQL and MinIO can be added later without changing handlers.
* Do not implement local MinIO or require a local MinIO process.
* Use a non-production storage adapter only as a test/dev placeholder if needed, clearly marked as not a MinIO replacement.
* Add focused unit and handler tests for validation, envelopes, upload metadata, metadata read, tags update, delete, and content retrieval behavior.
* Add service README documenting local run, environment variables, storage-port limitation, and follow-up integration points.

## Acceptance Criteria

* [ ] `services/file` builds independently with `go build ./cmd/server`.
* [ ] `services/file` tests pass with `go test ./...`.
* [ ] Health endpoints return standard JSON envelopes with request IDs.
* [ ] Upload endpoint accepts multipart `file` and repeated `tags` fields.
* [ ] Upload response matches `DocumentResponse` shape from gateway OpenAPI.
* [ ] Internal metadata read endpoint returns `DocumentResponse` for file-owned metadata.
* [ ] Update tags endpoint replaces file-owned tags and returns `DocumentResponse`.
* [ ] Delete endpoint makes the document unavailable to later metadata and content reads.
* [ ] Content endpoint returns raw bytes and safe content headers when storage can provide content.
* [ ] JSON errors use the standard `{ "error": { "code", "message", "requestId" } }` shape.
* [ ] Object storage is behind an interface; no MinIO client or local MinIO dependency is required for this MVP.
* [ ] README and `api/openapi.yaml` describe the internal file-service routes and the storage-port limitation.

## Definition of Done

* Service-local tests added and passing.
* `go build ./cmd/server` passing from `services/file`.
* `go test ./...` passing from `services/file`.
* Code follows `.trellis/spec/backend/` guidelines.
* Docs updated where service behavior or local startup expectations changed.
* Work committed with a useful Conventional Commit after user confirmation.

## Technical Approach

Build a small standard-library Go HTTP service first. Keep dependencies minimal while the service boundary is still young. Define ports in the service layer for metadata repository and object storage. Provide in-memory repository and in-process test storage for tests/dev, but keep the storage interface explicit so a future MinIO adapter can be added under `internal/platform/storage/minio` without changing HTTP handlers or service use cases.

Add the missing file-owned internal metadata read endpoint now because gateway and later services need a way to fetch file-owned metadata after upload. Keep it internal only: the public `GET /api/v1/documents/{documentId}` placeholder remains knowledge-owned until the knowledge contract is finalized.

## Decision (ADR-lite)

**Context**: The project needs a real file service module, but local MinIO should not be set up in this task.

**Decision**: Implement the service boundary, HTTP routes, business workflow, validation, metadata repository port, and object storage port now. Do not introduce a MinIO dependency or local MinIO setup. Keep any dev/test storage adapter clearly scoped as a placeholder.

**Consequences**:

* Gateway and backend teams get concrete route behavior and tests to integrate against.
* Production object persistence is not complete until a MinIO adapter and PostgreSQL repository are added.
* The service shape should remain stable because handlers depend on interfaces rather than MinIO/PostgreSQL details.

## Out of Scope

* Local MinIO setup or Docker Compose wiring for MinIO.
* Production MinIO adapter implementation.
* PostgreSQL repository implementation and real migrations beyond documented schema notes.
* Knowledge ingestion handoff.
* Public knowledge-owned document list/detail/chunks endpoints.
* Parsing, chunking, embedding, indexing, retry, and processing jobs.
* Gateway implementation or public gateway proxy handlers.
* Frontend upload UI.
* Report file storage.

## Technical Notes

* Relevant docs:
  * `docs/file.md`
  * `docs/api/gateway.openapi.yaml`
  * `docs/frontend-backend-contract.md`
  * `docs/service-boundaries.md`
  * `docs/gateway.md`
* Relevant specs:
  * `.trellis/spec/backend/index.md`
  * `.trellis/spec/backend/directory-structure.md`
  * `.trellis/spec/backend/database-guidelines.md`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/error-handling.md`
  * `.trellis/spec/backend/logging-guidelines.md`
  * `.trellis/spec/backend/quality-guidelines.md`
* Multipart tags encoding for this MVP: repeated `tags` fields.
* Current service directories under `services/` only contain `.gitkeep`; no existing Go service pattern exists yet.

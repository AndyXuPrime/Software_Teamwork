# Knowledge Service

Knowledge owns knowledge-base metadata, knowledge document metadata/status,
processing trace state, and future chunk/vector lifecycle coordination.

This implementation includes the A-09 foundation slice plus the A-10 document
upload handoff: Knowledge accepts the document upload, stores raw bytes through
File Service, creates durable document/job state, and enqueues ingestion work.

## Runtime

- Go module: `go 1.25.0`
- HTTP: standard `net/http` `ServeMux`
- Logging: `log/slog`
- PostgreSQL access: `pgx` + `sqlc`-shaped query package
- Migrations: `goose`

All landed Go services use the repository Go 1.25 baseline. Knowledge keeps the
standard `net/http` / `http.ServeMux` service shape while leaving room for later
RAG MCP server work.

## Configuration

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `DATABASE_URL` | yes | - | PostgreSQL connection string. |
| `FILE_SERVICE_BASE_URL` | yes | - | Internal File Service base URL for `/internal/v1/files`. |
| `KNOWLEDGE_REDIS_ADDR` | yes | - | Redis/asynq endpoint for ingestion task handoff. |
| `KNOWLEDGE_HTTP_ADDR` | no | `:8083` | HTTP listen address. |
| `KNOWLEDGE_SERVICE_VERSION` | no | `dev` | Version returned by readiness checks. |
| `KNOWLEDGE_ENV` | no | `local` | Runtime environment label. |
| `KNOWLEDGE_MAX_UPLOAD_BYTES` | no | `33554432` | Multipart upload limit in bytes. |
| `KNOWLEDGE_SERVICE_TOKEN` | no | - | Optional internal service token forwarded to File Service. |
| `KNOWLEDGE_SHUTDOWN_TIMEOUT` | no | `10s` | Graceful shutdown timeout. |

## Implemented Routes

Operational routes:

- `GET /healthz`
- `GET /readyz`

Internal service routes:

- `GET /internal/v1/knowledge-bases`
- `POST /internal/v1/knowledge-bases`
- `GET /internal/v1/knowledge-bases/{knowledgeBaseId}`
- `PATCH /internal/v1/knowledge-bases/{knowledgeBaseId}`
- `DELETE /internal/v1/knowledge-bases/{knowledgeBaseId}`
- `GET /internal/v1/knowledge-bases/{knowledgeBaseId}/documents`
- `POST /internal/v1/knowledge-bases/{knowledgeBaseId}/documents`
- `GET /internal/v1/documents/{documentId}`

Public gateway equivalents are documented in
`docs/services/gateway/api/openapi.yaml`.

## Access Context

Business routes require gateway-injected `X-User-Id`.

Supported permission strings follow the current auth docs:

- `knowledge:read`
- `knowledge:write`

Rules:

- Callers can read resources they created.
- `knowledge:read`, `knowledge:write`, `admin`, or `super_admin` can read
  broader resources.
- Create, update, and delete require `knowledge:write`, `admin`, or
  `super_admin`.
- Hidden or deleted resources return `404 not_found`.
- Authenticated callers without mutation rights receive `403 forbidden`.

## Data Model

The first migration creates:

- `knowledge_bases`
- `knowledge_documents`
- `processing_jobs`
- `document_chunks`

Document upload stores the File Service object ID only in
`knowledge_documents.file_ref`. Public document responses expose `jobId` and
document status, but never `fileRef`, File Service internal IDs, object keys, or
internal URLs.

`document_chunks` is included now as a provenance and cleanup anchor. This task
does not implement parser/chunker writes, embedding generation, Qdrant indexing,
or retrieval execution.

Knowledge base deletion is soft-delete-first:

- mark `knowledge_bases.deleted_at`;
- mark owned `knowledge_documents.deleted_at` in the same transaction for the
  PostgreSQL runtime repository;
- leave chunk/index cleanup for a future lifecycle job instead of hard-deleting
  chunks or vectors in this metadata route.


## Migrations

Apply the service-owned migration with the project-pinned `goose@v3.27.1` command:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$DATABASE_URL" up
```
## Development

```bash
go test ./...
go build ./cmd/server
```

If `sqlc` is available, regenerate the query package from `sqlc.yaml` after
changing SQL files.

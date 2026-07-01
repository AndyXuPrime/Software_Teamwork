# Knowledge Service

Knowledge exposes Gateway `/internal/v1/*` contract routes via the **contract
adapter** (`cmd/adapter`). KB metadata, documents, chunks, queries, and upload
flow through the vendored RAGFlow runtime at `VENDOR_RUNTIME_URL` (deepdoc +
Elasticsearch/Infinity + MinIO).

Parser-config admin routes (`/internal/v1/parser-configs`) optionally use legacy
goose PostgreSQL tables when `DATABASE_URL` is set.

## Runtime

- Go module: `go 1.25.0`
- Binary: `cmd/adapter` only (legacy `cmd/server` removed in Phase 5)
- HTTP: standard `net/http` `ServeMux`
- Logging: `log/slog`
- Parser-config storage: `pgx` + `sqlc` (optional)

See `runtime/README.md` for vendor runtime wiring and compose profiles.

## Configuration

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `VENDOR_RUNTIME_URL` | yes | `http://127.0.0.1:9380` | RAGFlow vendor HTTP base URL. |
| `DATABASE_URL` | no | - | PostgreSQL for parser-config admin; omit to return `502` on those routes. |
| `KNOWLEDGE_HTTP_ADDR` | no | `:8083` | HTTP listen address. |
| `KNOWLEDGE_SERVICE_VERSION` | no | `dev` | Version returned by readiness checks. |
| `KNOWLEDGE_ENV` | no | `local` | Runtime environment label. |
| `KNOWLEDGE_AUTO_START_INGESTION` | no | `true` | Call vendor `/documents/parse` after upload. |
| `KNOWLEDGE_SHUTDOWN_TIMEOUT` | no | `10s` | Graceful shutdown timeout. |

Upload storage and vector retrieval are configured in the vendor runtime
(`runtime/service_conf.compose.yaml`): MinIO bucket `software-teamwork-knowledge`,
doc engine `elasticsearch` or `infinity`. Knowledge does not call File Service,
Qdrant, Redis, or `services/parser`.

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
- `GET /internal/v1/documents/{documentId}/chunks`
- `GET /internal/v1/documents/{documentId}/content`
- `PATCH /internal/v1/documents/{documentId}`
- `DELETE /internal/v1/documents/{documentId}`
- `POST /internal/v1/knowledge-queries`
- `GET|POST|PATCH|DELETE /internal/v1/parser-configs[/**]` (requires `DATABASE_URL`)

Public gateway equivalents are documented in
`docs/services/gateway/api/public.openapi.yaml`.

## Access Context

Business routes require gateway-injected `X-User-Id` (from Auth service).
The adapter forwards this as vendor tenant context; vendor login/JWT is disabled.

Supported permission strings:

- `knowledge:read`
- `knowledge:write`
- `knowledge:admin` / `admin:parser-config:write` for parser-config admin

Rules:

- Read routes require `knowledge:read` or `knowledge:write` (or admin roles).
- Mutations require `knowledge:write` (or admin roles).
- Vendor errors map to standard `{error}` envelopes.

## Data Model

Goose migrations under `migrations/` retain legacy tables (`knowledge_bases`,
`parser_configs`, etc.) for parser-config admin. Vendor metadata uses separate
RAGFlow tables in the same PostgreSQL database when vendor PG is enabled.

## Local Integration Notes

Default compose runs the adapter against `VENDOR_RUNTIME_URL`. Start the vendor
Python API (:9380) and task executor locally, plus `knowledge-v2` profile services
(Elasticsearch, MinIO bucket init) as documented in `runtime/README.md`.

`services/parser` is retired; document parsing uses vendor deepdoc.

## Migrations

Apply the service-owned migration with the project-pinned `goose@v3.27.1` command:

```bash
go run github.com/pressly/goose/v3/cmd/goose@v3.27.1 -dir migrations postgres "$DATABASE_URL" up
```
## Development

Set `GOFLAGS=-mod=mod` when building or testing: `vendor/ragflow-runtime` is upstream
source, not Go module vendoring.

```bash
GOFLAGS=-mod=mod go test ./internal/adapter/... ./internal/adapterconfig/... ./internal/service/...
GOFLAGS=-mod=mod go build ./cmd/adapter
```

The Knowledge service runs the contract adapter (`cmd/adapter`) which proxies
Gateway `/internal/v1/*` routes to the vendored RAGFlow runtime at
`VENDOR_RUNTIME_URL`. Document upload, deepdoc parsing, embedding, and retrieval
use vendor MinIO + Elasticsearch/Infinity — not `services/parser`, Qdrant, or
the legacy Go ingestion worker.

Contract tests under `internal/adapter` use a fake vendor HTTP server. Live vendor
tests require `-tags=integration` and `KNOWLEDGE_VENDOR_INTEGRATION_URL`.

Regenerate the query package from `sqlc.yaml` after changing SQL files:

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate
```

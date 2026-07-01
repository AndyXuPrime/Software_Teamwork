# Knowledge RAGFlow Runtime (Phase 1)

This directory holds deployment scaffolding for replacing the legacy Go Knowledge
service with the trimmed vendor runtime under `vendor/ragflow-runtime/`.

## Storage decisions

| Layer | Target |
| --- | --- |
| Metadata | PostgreSQL `knowledge_system` (`DB_TYPE=postgres`, `postgres:` in service_conf) |
| Retrieval | Elasticsearch (default) or Infinity (compose profile) |
| Object storage | MinIO bucket `software-teamwork-knowledge` |

Phase 2 enables vendor metadata on PostgreSQL. Go ingestor and Python API both read
`postgres:` from [`service_conf.compose.yaml`](service_conf.compose.yaml). Legacy
goose tables (`knowledge_bases`, etc.) coexist in the same database but use a
different table namespace from vendor tables (`knowledgebase`, `user`, `document`).

## Processes

| Process | Port | Role |
| --- | --- | --- |
| `knowledge-adapter` | `:8083` | Gateway-facing contract adapter (sole Knowledge binary) |
| RAGFlow Python API | `:9380` | deepdoc / rag / dataset APIs (vendor) |
| RAGFlow task executor | n/a | ingestion workers (vendor) |

## Local compose profiles

- Default stack runs the Knowledge contract adapter (vendor proxy).
- `knowledge-v2` profile adds Elasticsearch and the `software-teamwork-knowledge`
  MinIO bucket init for the vendor doc engine.

```bash
cd deploy
VENDOR_RUNTIME_URL=http://host.docker.internal:9380 \
  docker compose --profile knowledge-v2 up -d elasticsearch knowledge-minio-init knowledge
```

The vendor Python API and task executor must be running separately until a vendor
container is added to compose. Adapter upload/parse/retrieve do not call
`services/parser`, File Service, Qdrant, or Redis.

## Environment

| Variable | Default | Meaning |
| --- | --- | --- |
| `VENDOR_RUNTIME_URL` | `http://127.0.0.1:9380` | Vendor HTTP base URL |
| `KNOWLEDGE_AUTO_START_INGESTION` | `true` | After upload, call vendor `/documents/parse` (deepdoc pipeline) |
| `DATABASE_URL` | optional | Legacy goose PostgreSQL for adapter parser-config admin routes |
| `DOC_ENGINE` | `elasticsearch` | Vendor doc engine selector (Elasticsearch or Infinity) |

## Ingestion (Phase 4)

Adapter upload flow:

1. `POST /internal/v1/knowledge-bases/{id}/documents` → vendor
   `POST /api/v1/datasets/{id}/documents?type=local`
2. When `KNOWLEDGE_AUTO_START_INGESTION=true` (default), adapter immediately calls
   vendor `POST /api/v1/datasets/{id}/documents/parse` with the new document id.
3. Vendor task executor runs deepdoc chunking/embedding; document `run` progresses
   `UNSTART` → `RUNNING` → `DONE`.

Contract tests (`internal/adapter/contract_test.go`) use a fake vendor HTTP server.
Live vendor tests use `-tags=integration` with `KNOWLEDGE_VENDOR_INTEGRATION_URL` and
`KNOWLEDGE_INTEGRATION_USER_ID`.

## Phase 5 (complete)

Legacy Go Knowledge server (`cmd/server`), ingestion worker, Qdrant client, Parser
HTTP client, and File Service upload path removed. Knowledge container runs adapter
only. Default compose no longer starts `services/parser` (available under `legacy`
profile only). Object storage and retrieval are vendor-only (MinIO + ES/Infinity).
Identity flows through Gateway/Auth (`X-User-Id`); vendor auth surfaces remain disabled.

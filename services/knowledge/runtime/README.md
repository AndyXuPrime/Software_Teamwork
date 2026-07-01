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
| `knowledge-adapter` | `:8083` | Gateway-facing contract adapter (this repo) |
| RAGFlow Python API | `:9380` | deepdoc / rag / dataset APIs (vendor) |
| RAGFlow task executor | n/a | ingestion workers (vendor) |

## Local compose profiles

- Default stack keeps the legacy Knowledge Go server (`KNOWLEDGE_RUNTIME_MODE=legacy`).
- `knowledge-v2` profile switches the Knowledge container to adapter mode and
  starts Elasticsearch for the vendor doc engine.

```bash
cd deploy
KNOWLEDGE_RUNTIME_MODE=adapter VENDOR_RUNTIME_URL=http://host.docker.internal:9380 \
  docker compose --profile knowledge-v2 up -d elasticsearch knowledge-minio-init
```

Default compose keeps the legacy Knowledge server. Adapter mode is opt-in via
`KNOWLEDGE_RUNTIME_MODE=adapter` once the vendor runtime is reachable at
`VENDOR_RUNTIME_URL`.

## Environment

| Variable | Default | Meaning |
| --- | --- | --- |
| `KNOWLEDGE_RUNTIME_MODE` | `legacy` | `adapter` runs `cmd/adapter` |
| `VENDOR_RUNTIME_URL` | `http://127.0.0.1:9380` | Vendor HTTP base URL |
| `DOC_ENGINE` | `elasticsearch` | Vendor doc engine selector |
| `DB_TYPE` | `mysql` (legacy vendor default) / `postgres` (Knowledge replacement) | Metadata backend selector |
| `DATABASE_URL` | optional | Overrides Go `DatabaseConfig` when set (`postgres://...`) |

## Next phases

1. Phase 2 — PostgreSQL metadata port + schema migration
2. Phase 3 — Implement `/internal/v1/*` contract routes in adapter
3. Phase 4 — deepdoc ingestion pipeline, drop parser dependency
4. Phase 5 — Remove legacy Go implementation

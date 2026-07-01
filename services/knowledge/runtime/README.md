# Knowledge deployment notes

The contract adapter lives in `services/knowledge/`. The RAG/deepdoc runtime is a
separate first-class service at `services/knowledge-runtime/`.

## Processes

| Compose service | Port | Role |
| --- | --- | --- |
| `knowledge` | `:8083` | Go contract adapter (Gateway-facing) |
| `knowledge-runtime-api` | `:9380` | RAGFlow Python API (profile `knowledge-v2`) |
| `knowledge-runtime-worker` | n/a | deepdoc task executors (profile `knowledge-v2`) |

## Storage

| Layer | Target |
| --- | --- |
| Adapter parser-config admin | PostgreSQL `knowledge_system` (goose migrations) |
| Runtime metadata | PostgreSQL `knowledge_system` (vendor Peewee tables) |
| Object storage | MinIO bucket `software-teamwork-knowledge` |
| Retrieval | Elasticsearch (default) |

## Compose profiles

- Default stack runs the Knowledge adapter only.
- `knowledge-v2` adds Elasticsearch, knowledge MinIO bucket init, and optional
  in-compose runtime containers (`knowledge-runtime-api`, `knowledge-runtime-worker`).

Full in-compose vendor stack:

```bash
cd deploy
VENDOR_RUNTIME_URL=http://knowledge-runtime-api:9380 \
  docker compose --profile knowledge-v2 up -d \
  elasticsearch knowledge-minio-init \
  knowledge-runtime-api knowledge-runtime-worker knowledge
```

External vendor (local Python dev):

```bash
cd deploy
docker compose --profile knowledge-v2 up -d elasticsearch knowledge-minio-init knowledge
# In another terminal: services/knowledge-runtime/deploy/api/run-local.sh
# and deploy/worker/run-local.sh
```

## Environment

| Variable | Default | Meaning |
| --- | --- | --- |
| `VENDOR_RUNTIME_URL` | `http://host.docker.internal:9380` | Vendor HTTP base URL for adapter |
| `KNOWLEDGE_AUTO_START_INGESTION` | `true` | After upload, call vendor parse API |
| `KNOWLEDGE_RUNTIME_WORKERS` | `2` | Worker count for `knowledge-runtime-worker` |
| `DOC_ENGINE` | `elasticsearch` | Vendor doc engine selector |

See also `services/knowledge-runtime/README.md`.

# Knowledge Runtime

First-class RAG/deepdoc runtime (formerly `services/knowledge/vendor/ragflow-runtime`).
The Knowledge **contract adapter** lives separately in `services/knowledge/cmd/adapter`.

## Processes (Tier 2 split)

| Service | Port | Entry | Role |
| --- | --- | --- | --- |
| `knowledge-runtime-api` | `:9380` | `api/ragflow_server.py` | Dataset/document/search HTTP API |
| `knowledge-runtime-worker` | n/a | `rag/svr/task_executor.py` | deepdoc parse, chunk, embed (Redis queue) |

Both share PostgreSQL (`knowledge_system`), MinIO (`software-teamwork-knowledge`), Elasticsearch, and Redis.

## Docker (production)

Build once from this directory, run two compose services with different commands:

```bash
cd services/knowledge-runtime
docker build -t knowledge-runtime:local .

# API only
docker run --rm knowledge-runtime:local ./entrypoint.sh --disable-taskexecutor

# Worker only
docker run --rm knowledge-runtime:local ./entrypoint.sh --disable-webserver --workers=2
```

Full stack via root compose (profile `knowledge-v2`):

```bash
cd deploy
docker compose --profile knowledge-v2 up -d \
  elasticsearch knowledge-minio-init \
  knowledge-runtime-api knowledge-runtime-worker knowledge
```

## Local development (no Docker image build)

Requires Python 3.13 + [uv](https://github.com/astral-sh/uv):

```bash
cd services/knowledge-runtime
uv sync --python 3.13 --frozen
export PYTHONPATH=.
cp conf/service_conf.compose.yaml conf/service_conf.yaml
# Edit conf/service_conf.yaml hosts for localhost (postgres, redis, minio, es)

# Terminal 1 — API
./deploy/api/run-local.sh

# Terminal 2 — worker
./deploy/worker/run-local.sh
```

Adapter (separate module):

```bash
cd services/knowledge
VENDOR_RUNTIME_URL=http://127.0.0.1:9380 go run ./cmd/adapter
```

## Configuration

- Project overlay: `conf/service_conf.compose.yaml` (used by compose via env substitution)
- Upstream template: `docker/service_conf.yaml.template` (rendered by `entrypoint.sh` in containers)
- Go ingestor (`cmd/ingestor.go`, NATS) is **not** the default worker path; Python `task_executor.py` + Redis is.

## Upstream

See `UPSTREAM.md` for import provenance and refresh instructions.

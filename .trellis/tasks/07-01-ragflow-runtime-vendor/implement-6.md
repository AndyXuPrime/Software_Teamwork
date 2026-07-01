# Phase 6 — Promote runtime to first-class service (Tier 1 + Tier 2)

## Done

### Tier 1: First-class service
- `git mv services/knowledge/vendor/ragflow-runtime → services/knowledge-runtime/`
- Added `services/knowledge-runtime/README.md`, `UPSTREAM.md` path updated
- Project compose template: `conf/service_conf.yaml.template` (PostgreSQL + MinIO + ES)
- Dockerfile uses project template instead of upstream MySQL default

### Tier 2: API / worker split
- `deploy/api/` — local `run-local.sh`, container CMD doc (`--disable-taskexecutor`)
- `deploy/worker/` — local `run-local.sh`, container CMD doc (`--disable-webserver --workers=N`)
- Compose services (profile `knowledge-v2`):
  - `knowledge-runtime-api` (:9380)
  - `knowledge-runtime-worker` (Redis queue)
- `knowledge` optional `depends_on: knowledge-runtime-api` (`required: false`)

### Docs / cleanup
- `services/knowledge/runtime/README.md` — points to knowledge-runtime
- `services/knowledge/README.md` — removed `GOFLAGS=-mod=mod` note
- `services/knowledge/Dockerfile` — removed `GOFLAGS`
- `deploy/README.md` — in-compose runtime instructions

## Verify

```bash
docker compose config --quiet   # deploy/
go test ./internal/adapter/...  # services/knowledge/
```

Full stack (requires RAGFlow image build — heavy):

```bash
cd deploy
VENDOR_RUNTIME_URL=http://knowledge-runtime-api:9380 \
  docker compose --profile knowledge-v2 up -d \
  elasticsearch knowledge-minio-init \
  knowledge-runtime-api knowledge-runtime-worker knowledge
```

## Not in scope (later)

- Dedicated CI workflow for `knowledge-runtime` Docker build
- Remove duplicate `services/knowledge/runtime/service_conf.compose.yaml`
- Task `info.md` / `prd.md` path updates (historical audit log)

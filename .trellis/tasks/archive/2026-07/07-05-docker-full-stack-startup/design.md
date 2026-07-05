# Docker Full-Stack Startup Design

## Architecture

Keep the existing infra-only baseline intact:

```text
deploy/docker-compose.yml
  postgres
  redis
  minio
  minio-init
  elasticsearch
```

Add a separate cloud-first Docker path:

```text
scripts/docker/start.sh
  -> docker compose -f deploy/docker-compose.cloud.yml up --build

deploy/docker-compose.cloud.yml
  migrate
  seed
  auth / file / knowledge / ai-gateway / qa / document / gateway
  frontend
```

The short command is the user-facing API:

```bash
./scripts/docker/start.sh
```

The script hides Compose file names, env-file handling, project name, and common
flags. Existing host-run users continue to use:

```bash
./scripts/local/start.sh
```

## Files To Add

- `scripts/docker/start.sh`
- `scripts/docker/stop.sh`
- `scripts/docker/clean.sh`
- `deploy/docker-compose.cloud.yml`
- `deploy/docker/full/go-service.Dockerfile`
- `deploy/docker/full/go-tools.Dockerfile`
- `deploy/docker/full/frontend.Dockerfile`
- `deploy/docker/full/nginx.conf`
- `deploy/docker/full/migrate.sh`
- `deploy/docker/full/seed.sh`
- `deploy/docker/cloud.env.example`

## Image Strategy

Use a shared Go service Dockerfile with build args:

```text
SERVICE_DIR=services/auth
TARGET=./cmd/server
BINARY=auth-server
```

This avoids seven near-identical Dockerfiles while still producing distinct
service images from Compose build definitions.

Use multi-stage builds:

- Go builder image: `golang:1.25-alpine` or equivalent pinned Go 1.25 image.
- Runtime image: Alpine or distroless-like base with CA certificates.
- Frontend builder: pinned Bun image, followed by nginx static serving.
The default cloud Docker path intentionally does not build Knowledge runtime.
It expects `VENDOR_RUNTIME_URL` to point to an external/cloud Knowledge runtime
API. That runtime can use cloud OCR and cloud AI providers, and the local Docker
stack only runs the product Go services plus frontend.

## Container Configuration

The Docker path should not source host-run `.local/config/dev.env` directly
because many defaults use `localhost` and host-run infrastructure. Instead it
uses an explicit Docker cloud env file copied from
`deploy/docker/cloud.env.example`.

Examples:

```text
AUTH_DATABASE_URL=postgres://auth_app:<password>@cloud-postgres.example:5432/auth_system?sslmode=require
GATEWAY_AUTH_BASE_URL=http://auth:8001
GATEWAY_KNOWLEDGE_BASE_URL=http://knowledge:8083
GATEWAY_QA_BASE_URL=http://qa:8084
GATEWAY_DOCUMENT_BASE_URL=http://document:8085
GATEWAY_AI_GATEWAY_BASE_URL=http://ai-gateway:8086
FILE_MINIO_ENDPOINT=<cloud-minio-or-s3-compatible-endpoint>
VENDOR_RUNTIME_URL=https://knowledge-runtime.example
PADDLEOCR_ACCESS_TOKEN=<cloud-ocr-token>
```

Cloud secrets come from `.env.docker.cloud`, which is ignored by git. Non-secret
container-specific service URLs live in the cloud Compose file so they are
reviewable and do not mutate the existing host-run profile.

## Cloud OCR And Model Providers

`seed.sh` mirrors the current host-run seed behavior:

- If `PADDLEOCR_ACCESS_TOKEN` is set, it writes a default
  `paddleocr_cloud` parser config into the Knowledge database.
- If `AI_GATEWAY_LOCAL_SEED_ENABLED=true`, it runs the AI Gateway local seed
  helper so default chat, embedding, and rerank profiles point to the configured
  cloud provider.

This keeps heavy OCR/model inference off the local machine. The local Docker
containers still orchestrate requests, but OCR, LLM, embedding, and rerank work
are handled by cloud services.

## Startup Ordering

Use Compose health checks and one-shot jobs:

1. `migrate` runs goose migrations for all services against cloud PostgreSQL.
2. `seed` applies local demo seed SQL and optional AI Gateway seed overlay if
   supported in the Docker path.
3. App services start.
4. `frontend` serves static assets and proxies `/api/v1` to `gateway`.

Compose `depends_on.condition` is acceptable for local/demo startup ordering of
jobs and app containers. Cloud dependencies must already be provisioned and
reachable before running the Docker command.

## Frontend Entry

The Docker path should expose one browser entrypoint, preferably:

```text
http://localhost:18080
```

Using a non-conflicting port avoids colliding with host-run Gateway `8080` and
Vite `5173`. The frontend container can serve static files with nginx and proxy
`/api/v1`, `/healthz`, and `/readyz` to `gateway`.

## Compatibility

This is an additive path. It must not:

- change `deploy/docker-compose.yml`
- change host-run service defaults
- change `.env.example` in a way that breaks host-run startup
- require `docker compose --build` on the infra-only baseline
- start local OCR, local Knowledge runtime worker, or local Elasticsearch in the
  default Docker path

## Trade-Offs

- A wrapper script is slightly less "pure Docker" than a raw Compose command,
  but it gives the user the short one-line command they asked for and isolates
  Compose complexity.
- Cloud Docker startup requires users to provide cloud dependency URLs and
  secrets before the one-line start command can succeed.
- Real model provider calls still require local user credentials; placeholder
  profiles can validate process startup but not real LLM behavior.

## Rollback

Because the path is additive, rollback is removing the new `scripts/docker/*`,
`deploy/docker-compose.cloud.yml`, and `deploy/docker/full/*` files. Existing
host-run behavior should remain untouched.

# Production/Staging Compose Baseline

This is the production or staging deployment baseline for a single-machine
Docker Compose target. It is separate from `deploy/docker-compose.yml`, which
remains a local/demo integration stack with placeholder credentials and seed
data.

## Files

| File | Purpose |
| --- | --- |
| `deploy/docker-compose.production.yml` | Production/staging Compose skeleton. Uses published service images, persistent volumes, migrations, and internal service networking. |
| `deploy/.env.production.example` | Required variable names and safe placeholders. Copy it outside git before filling real values. |
| `deploy/nginx/production.conf` | Public ingress contract. Routes `/api/v1`, `/healthz`, and `/readyz` to gateway and all other browser paths to frontend. |
| `deploy/postgres/init-production/001-create-service-databases.sh` | Empty-volume PostgreSQL initializer that creates service databases and roles from env-injected passwords. |

Do not use `deploy/.env.example`, `deploy/postgres/init/`, or `deploy/seeds/`
for production/staging. Those files are local/demo only.

## Scope

This baseline covers:

- frontend, gateway, auth, file, parser, knowledge, qa, document, ai-gateway;
- a single public ingress that keeps frontend and gateway on one browser origin;
- PostgreSQL, Redis, Qdrant, and MinIO;
- service migrations before service startup;
- persistent Docker volumes;
- health/readiness checks;
- operator runbook requirements for startup, backup, upgrade, and rollback.

It does not deploy to a real cloud provider, create DNS/TLS certificates, add a
GitHub Actions deploy workflow, or replace the future #125 full cross-service
smoke.

Long-running services use `restart: unless-stopped` so they recover after a
process crash or Docker daemon restart. One-shot initialization and migration
jobs keep `restart: "no"` so failed setup remains visible to operators.

## Image Strategy

Production/staging should run immutable, prebuilt image tags:

```bash
FRONTEND_IMAGE=registry.example.com/software-teamwork/frontend:<git-sha-or-release>
NGINX_IMAGE=nginx:<pinned-alpine-tag>
GATEWAY_IMAGE=registry.example.com/software-teamwork/gateway:<git-sha-or-release>
AUTH_IMAGE=registry.example.com/software-teamwork/auth:<git-sha-or-release>
FILE_IMAGE=registry.example.com/software-teamwork/file:<git-sha-or-release>
PARSER_IMAGE=registry.example.com/software-teamwork/parser:<git-sha-or-release>
KNOWLEDGE_IMAGE=registry.example.com/software-teamwork/knowledge:<git-sha-or-release>
QA_IMAGE=registry.example.com/software-teamwork/qa:<git-sha-or-release>
DOCUMENT_IMAGE=registry.example.com/software-teamwork/document:<git-sha-or-release>
AI_GATEWAY_IMAGE=registry.example.com/software-teamwork/ai-gateway:<git-sha-or-release>
MIGRATE_IMAGE=registry.example.com/software-teamwork/goose-migrate:<git-sha-or-release>
```

Do not use `latest`. Rollback depends on being able to identify the previous
image tags.

The current repository has backend/parser Dockerfiles and no frontend Dockerfile
yet. Before enabling the `frontend` service in a shared environment, publish a
frontend static-server image that listens on port 80 and serves the built SPA.
The Compose `ingress` service is the only public HTTP entrypoint: it proxies
`/api/v1`, `/healthz`, and `/readyz` to `gateway:8080`, and proxies all other
browser paths to `frontend:80`. This preserves the frontend client contract that
browser API calls use same-origin `/api/v1`.

If frontend hosting is terminated outside this Compose stack, keep the same
ingress contract at the external edge: browser `/api/v1/**` traffic must still
route to gateway, and browsers must not call service containers directly.

## First-Time Build Or Pull

First-time image build/pull requires Docker Hub or an enterprise registry mirror
to be reachable for base image metadata and packages. The parser image is the
most sensitive path because it depends on `python:3.12-slim`, Debian packages,
`uv`, Python indexes, and PaddleOCR dependencies.

If `docker compose up --no-build` fails because the parser image is missing,
recover explicitly instead of relying on a developer machine's image cache:

```bash
docker pull "$PARSER_IMAGE"
```

If the parser image must be built locally:

```bash
DOCKER_BUILDKIT=1 docker build \
  -f services/parser/Dockerfile \
  -t "$PARSER_IMAGE" \
  services/parser
```

For mainland China networks, prefer explicit registry/package rewrites as
documented in `deploy/.env.china.example` and
`docs/runbooks/docker-build-environment.md`. Keep Go checksum verification on;
do not make `GOSUMDB=off` a normal deployment path.

## Environment And Secrets

Prepare a real env file outside version control:

```bash
cp deploy/.env.production.example deploy/.env.production
```

Then replace every `REPLACE_WITH_...` value. Do not commit the filled file.

Required secret classes:

- PostgreSQL admin and per-service database passwords.
- `INTERNAL_SERVICE_TOKEN`.
- `TOKEN_HASH_SECRET` and `TOKEN_HASH_KEY_VERSION`.
- `AI_GATEWAY_SERVICE_TOKEN_HASHES`, derived from allowed internal tokens.
- MinIO root user/password or equivalent S3-compatible credentials.
- `QA_CONFIG_ENCRYPTION_KEY`, a 64-hex-character key.
- `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF` and
  `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY`.

Current service database configuration uses PostgreSQL URL strings. Generate
per-service database passwords with URL-safe characters such as letters,
numbers, dot, underscore, tilde, or hyphen. If an operator must use characters
such as `@`, `/`, `#`, or `:`, update both the DSN value and the database role
password deliberately; do not leave an accidentally broken URL in the env file.

Provider API keys are not stored in `.env.production`. Configure them through AI
Gateway model profile/credential APIs after AI Gateway is running. Production
operators should prefer secret references or deployment-environment injection;
the encrypted-column mode is the first-stage repository-supported fallback.

## Static Validation

Before starting containers:

```bash
python3 scripts/check_docker_policy.py
docker compose -f deploy/docker-compose.production.yml \
  --env-file deploy/.env.production.example \
  config --quiet
```

Validate the filled env file before deployment:

```bash
docker compose -f deploy/docker-compose.production.yml \
  --env-file deploy/.env.production \
  config --quiet
```

These checks prove Compose interpolation and policy shape. They do not prove
that placeholder secrets or unpublished images can run.

## Startup Order

Use a release checkout that contains `services/*/migrations`, because the
migration services mount those directories read-only.

The PostgreSQL initializer in `deploy/postgres/init-production/` runs only when
the `postgres_data` volume is empty, following the official PostgreSQL image
contract. Later database password rotation must be performed explicitly with
`ALTER ROLE` or an approved operator procedure; changing `.env.production` alone
does not re-run the initializer on an existing volume.

1. Start infrastructure:

   ```bash
   docker compose -f deploy/docker-compose.production.yml \
     --env-file deploy/.env.production \
     up -d postgres redis qdrant minio minio-init
   ```

2. Apply migrations:

   ```bash
   docker compose -f deploy/docker-compose.production.yml \
     --env-file deploy/.env.production \
     up migrate-auth migrate-file migrate-knowledge migrate-qa migrate-document migrate-ai-gateway
   ```

3. Start services:

   ```bash
   docker compose -f deploy/docker-compose.production.yml \
     --env-file deploy/.env.production \
     up -d auth file parser ai-gateway knowledge qa document gateway frontend ingress
   ```

4. Configure AI Gateway provider credentials and model profiles through the
   admin/profile APIs. QA and Document model-dependent flows require matching
   profile/model IDs in `AI_GATEWAY_PROFILE_ID`, `MODEL_ID`, and
   `DOCUMENT_AI_GATEWAY_PROFILE_ID`.

The production Compose healthcheck for `ai-gateway` intentionally accepts
`/readyz` status `degraded` during startup. `degraded` means the AI Gateway
process, database connection, and profile store query path are available, but
one or more chat/embedding/rerank provider profiles or credentials still need
operator configuration. This prevents a first-deploy deadlock where the
admin/profile APIs cannot be reached until the profiles they configure already
exist.

This baseline intentionally does not create a demo admin user. Bootstrap the
first production administrator through the approved Auth/operator process before
using admin model-profile or settings APIs.

For a compact staging rollout after env and image tags are ready, `up -d` on the
full file is acceptable, but keep the ordered sequence above for first deploys
and incident recovery.

## Readiness Checks

Only `ingress` exposes a host port by default. Frontend, gateway, and internal
service ports stay on the Compose network.

Host checks:

```bash
curl -fsS http://localhost:${INGRESS_HTTP_PORT:-80}/healthz
curl -fsS http://localhost:${INGRESS_HTTP_PORT:-80}/readyz
```

Container-network checks:

```bash
docker compose -f deploy/docker-compose.production.yml \
  --env-file deploy/.env.production ps

docker compose -f deploy/docker-compose.production.yml \
  --env-file deploy/.env.production logs ingress gateway auth file parser knowledge qa document ai-gateway
```

Service readiness endpoints inside containers:

```text
auth       http://auth:8001/readyz
file       http://file:8082/readyz
parser     http://parser:8087/readyz
knowledge  http://knowledge:8083/readyz
qa         http://qa:8084/readyz
document   http://document:8085/readyz
ai-gateway http://ai-gateway:8086/readyz
gateway    http://gateway:8080/readyz
ingress    http://ingress/readyz
```

`ai-gateway /readyz` returns HTTP 503 with top-level status `degraded` until
required provider profiles and credentials are configured. Compose treats that
specific state as healthy only for startup ordering; operators should still
inspect the endpoint body and complete model-profile bootstrap before accepting
AI-dependent traffic.

`ingress /readyz` delegates to `gateway /readyz`. Gateway readiness checks
Redis, Auth `/readyz`, and owner service base URL configuration; it does not
call Knowledge, QA, Document, or AI Gateway readiness endpoints and does not
prove business workflows or external provider credentials are accepted. Use the
AI Gateway provider smoke and the Gateway -> Knowledge -> QA RAG smoke from
`docs/runbooks/local-integration.md` when provider/profile or full
cross-service validation is required.

## Persistence And Backup

| Volume | Data | Backup concern |
| --- | --- | --- |
| `postgres_data` | Auth, File metadata, Knowledge metadata, QA messages/settings, Document jobs, AI Gateway profiles/credentials. | Use `pg_dumpall` or per-database `pg_dump` before upgrades and before irreversible migrations. |
| `redis_data` | Redis AOF for gateway sessions and async queues. | Redis is not the business source of truth, but queue/session loss affects in-flight work. Snapshot before maintenance when feasible. |
| `qdrant_data` | Knowledge vector index. | Use Qdrant snapshot/export procedures or rebuild from Knowledge chunks if an index rebuild is acceptable. |
| `minio_data` | File object bytes and generated document artifacts. | Use MinIO/S3 backup tooling, bucket replication, or `mc mirror`; keep bucket/object data aligned with File metadata backups. |
| `qa_workspace` | QA agent workspace files. | Treat as operational state; back up when active sessions or tool artifacts must survive rollback. |

PostgreSQL is the authority for business state. Redis/asynq is queue/session
runtime state, Qdrant is retrieval index state, and MinIO stores bytes behind
the File Service boundary.

## Upgrade

1. Publish new immutable image tags.
2. Back up PostgreSQL and MinIO before schema or object-format changes.
3. Review migrations. Do not run irreversible migrations automatically without
   an explicit release decision.
4. Update `deploy/.env.production` image tags.
5. Run Compose config validation.
6. Apply migrations.
7. Restart affected services.
8. Run readiness checks and targeted smoke tests.

## Rollback

Minimum rollback path:

1. Keep the previous `.env.production` or record the previous image tags.
2. Repoint image variables to the last known-good tags.
3. Re-run Compose config validation.
4. Start the previous images:

   ```bash
   docker compose -f deploy/docker-compose.production.yml \
     --env-file deploy/.env.production \
     up -d
   ```

5. If a migration is not backward-compatible, stop and restore the matching
   PostgreSQL/MinIO backup instead of assuming image rollback is enough.

## Local/Demo Boundary

The local/demo stack still lives in `deploy/docker-compose.yml` and
`deploy/.env.example`. It intentionally includes local placeholders, local admin
seed data, local hashing defaults, and optional fake AI profiles for smoke
testing. Do not promote those values into production/staging.

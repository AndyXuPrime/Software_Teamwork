# Local Integration Environment

This directory is the S-05 local/demo integration baseline. It starts shared
infrastructure plus the backend service loop through gateway. It is not a
production deployment baseline.

Production or staging uses a separate baseline:
[`production-baseline.md`](./production-baseline.md),
`docker-compose.production.yml`, `.env.production.example`, and
`nginx/production.conf`. Do not promote the local `.env.example`, local seed
data, or local demo credentials into shared or long-lived environments. The
production baseline exposes only the `ingress` service publicly; it routes
browser `/api/v1/**` calls to gateway and keeps frontend/gateway service ports
inside the Compose network.

## Entry Points

- Browser/frontend entrypoint: `http://localhost:8080` through gateway only.
- Do not point frontend code at `auth`, `file`, `parser`, `knowledge`, `qa`,
  `document`, `ai-gateway`, PostgreSQL, Redis, Qdrant, or MinIO directly.
- Internal service ports are exposed for local debugging only.

## Start

```powershell
cd deploy
Copy-Item .env.example .env
docker compose up -d --build
```

Mainland China recommended overlay:

```powershell
cd deploy
Copy-Item .env.example .env
Get-Content .env.china.example | Add-Content .env
$env:DOCKER_BUILDKIT = "1"
docker compose up -d --build
```

For Bash:

```bash
cd deploy
cp .env.example .env
cat .env.china.example >> .env
DOCKER_BUILDKIT=1 docker compose up -d --build
```

This overlay uses explicit registry rewrites and package mirrors. It is the
preferred path for users with no Docker mirror/proxy configured, and it avoids
depending on daemon-level mirror behavior.

AI/模型功能必需：AI Gateway profile

The default stack starts the core services. Enable the `ai` profile before using
admin model profiles, QA real model calls, Document AI generation, real
embedding/rerank paths, or AI Gateway provider smoke.

```powershell
cd deploy
docker compose --profile ai up -d --build
```

To start only the AI Gateway profile service on top of an already running core
stack:

```powershell
docker compose --env-file .env --profile ai up -d --build ai-gateway
Invoke-RestMethod http://localhost:8086/readyz
```

`gateway /readyz` does not prove AI Gateway or provider readiness. Seeded local
AI profiles are placeholders until replaced with real provider credentials.

Default seeded login:

```text
username: admin
password: LocalDemoAdmin#12345
```

These credentials and all secrets in `.env.example` are local placeholders.
Replace them for any shared or long-lived environment.

## Docker Images Required

If Docker has no local images, install them with:

```powershell
docker pull postgres:16-alpine
docker pull redis:7-alpine
docker pull qdrant/qdrant:v1.18.2
docker pull minio/minio:RELEASE.2025-09-07T16-13-09Z
docker pull minio/mc:RELEASE.2025-08-13T08-35-41Z
docker pull golang:1.25-alpine
docker pull alpine:3.22
docker pull python:3.12-slim
```

Then build service images:

```powershell
cd deploy
docker compose build
docker compose --profile ai build
```

Docker build priority for this repository is: builds must run first, then be
fast, then produce small images, then reduce memory use, then reduce storage
use. For that reason the default Go Docker build args keep checksum verification
on and use the Go toolchain's official proxy/sumdb defaults:

```text
GO_DOCKER_GOPROXY=https://proxy.golang.org,direct
GO_DOCKER_GOSUMDB=sum.golang.org
```

For mainland China networks, use an explicit override instead of changing the
repository default:

```powershell
$env:GO_DOCKER_GOPROXY = "https://goproxy.cn,direct"
$env:GO_DOCKER_GOSUMDB = "sum.golang.google.cn"
docker compose build migrate-file
```

Do not set `GOSUMDB=off` for normal builds. `goproxy.cn` may proxy
`sum.golang.org` checksum requests; if that mirror path returns bad 404s, goose
or service module verification can fail during migration image builds. Pairing a
module proxy with `sum.golang.google.cn` keeps checksum verification enabled
while avoiding that third-party sumdb proxy path.

Optional build args for local acceleration:

| Variable | Use |
| --- | --- |
| `DOCKER_IMAGE_REGISTRY_PREFIX` | Prefix `FROM` images with a local/enterprise registry mirror. Include the trailing slash. |
| `POSTGRES_IMAGE` / `REDIS_IMAGE` / `QDRANT_IMAGE` / `MINIO_IMAGE` / `MINIO_MC_IMAGE` | Override Compose infrastructure images while keeping pinned defaults. |
| `GO_DOCKER_GOPROXY` / `GO_DOCKER_GOSUMDB` | Override Go module proxy and checksum database for Go service and migration builds. |
| `ALPINE_MIRROR` | Override Alpine apk repositories, for example a university mirror ending in `/alpine`. |
| `DEBIAN_APT_MIRROR` / `DEBIAN_SECURITY_APT_MIRROR` | Override Parser Debian apt repositories. |
| `PIP_INDEX_URL` / `UV_DEFAULT_INDEX` / `UV_INDEX` | Override Parser Python package indexes. |

Detailed setup, mirror diagnostics, and storage cleanup are documented in
[`docs/runbooks/docker-build-environment.md`](../docs/runbooks/docker-build-environment.md).

Before changing daemon mirrors or proxies, run:

```powershell
python3 ../scripts/check_docker_environment.py --profile all --clean-env
```

Use the results to choose: explicit registry rewrite first, working daemon
mirror second, Docker daemon proxy last.

The local Qdrant, MinIO server, MinIO `mc`, Redis, PostgreSQL, and Alpine
runtime images are pinned to explicit tags in this repository. MinIO uses one
server image plus one `mc` initialization image; `minio-init` is not a second
MinIO server. Update this document and
`docs/architecture/technology-decisions.md` in the same PR when changing them.

## Ports

| Component | Host port | Container port | Purpose |
| --- | ---: | ---: | --- |
| gateway | 8080 | 8080 | Browser/backend entrypoint |
| auth | 8001 | 8001 | Internal auth service |
| file | 8082 | 8082 | Internal file service |
| knowledge | 8083 | 8083 | Internal knowledge service |
| qa | 8084 | 8084 | Internal QA service |
| document | 8085 | 8085 | Internal document service |
| ai-gateway | 8086 | 8086 | Model/profile service for AI features |
| parser | 8087 | 8087 | Internal parser service |
| postgres | 5432 | 5432 | Local relational databases |
| redis | 6379 | 6379 | Sessions, queues, coordination |
| qdrant | 6333/6334 | 6333/6334 | Vector database |
| minio | 9000/9001 | 9000/9001 | Object storage and console |

Override host ports in `deploy/.env`.

## Environment Variables

| Variable | Service | Required | Description |
| --- | --- | --- | --- |
| `INTERNAL_SERVICE_TOKEN` | gateway/auth/file/parser/knowledge/qa/document/ai-gateway | yes | Local service-to-service token placeholder. |
| `TOKEN_HASH_SECRET` | gateway/auth | yes | Local HMAC secret for opaque token hashes. |
| `GATEWAY_AUTH_BASE_URL` | gateway | set in Compose | Internal auth base URL. |
| `GATEWAY_KNOWLEDGE_BASE_URL` | gateway | set in Compose | Internal knowledge base URL. |
| `GATEWAY_QA_BASE_URL` | gateway | set in Compose | Internal QA base URL. |
| `GATEWAY_DOCUMENT_BASE_URL` | gateway | set in Compose | Internal document base URL. |
| `GATEWAY_AI_GATEWAY_BASE_URL` | gateway | set in Compose | Internal AI Gateway base URL; AI/model routes require the `ai` profile to run. |
| `AUTH_DATABASE_URL` | auth | yes | Auth PostgreSQL DSN. |
| `FILE_DATABASE_URL` | file | yes | File metadata PostgreSQL DSN. |
| `FILE_STORAGE_BACKEND` | file | no | `local` in Compose for durable local smoke tests. |
| `DATABASE_URL` | knowledge | yes | Knowledge PostgreSQL DSN. |
| `FILE_SERVICE_BASE_URL` | knowledge | yes | Internal File Service URL. |
| `PARSER_SERVICE_BASE_URL` | knowledge | yes | Internal Parser Service URL. |
| `PARSER_SERVICE_TOKEN` | knowledge/parser | yes | Local service token for Parser Service calls. |
| `KNOWLEDGE_REDIS_ADDR` | knowledge | yes | Redis/asynq endpoint. |
| `EMBEDDING_PROVIDER` / `EMBEDDING_MODEL` / `EMBEDDING_DIMENSION` | knowledge | no | Defaults to local hashing embeddings for deterministic local retrieval tests. |
| `KNOWLEDGE_QDRANT_URL` / `QDRANT_COLLECTION` | knowledge | no | Optional Qdrant REST URL and collection; leave URL empty to use Knowledge's in-memory vector index. |
| `KNOWLEDGE_AI_GATEWAY_BASE_URL` / `AI_GATEWAY_EMBEDDING_PROFILE_ID` | knowledge | no | AI Gateway embedding profile wiring. Requires `--profile ai` and real provider credentials when `EMBEDDING_PROVIDER=ai_gateway`. |
| `RERANK_MODEL` / `RERANK_PROFILE_ID` | knowledge | no | AI Gateway rerank wiring. Empty `RERANK_MODEL` keeps rerank requests on the local no-op fallback. |
| `PARSER_BACKEND` | parser | no | Defaults to `ppstructurev3` for structured PDF/image parsing; set `document` only for local text/Office parsing without OCR dependencies. |
| `QA_DATABASE_URL` | qa | yes | QA PostgreSQL DSN. |
| `KNOWLEDGE_SERVICE_URL` | qa | yes | Internal Knowledge Service URL. |
| `AI_GATEWAY_URL` | qa | yes | Internal chat completions URL; QA real model calls require `--profile ai`. |
| `AI_GATEWAY_PROFILE_ID` / `MODEL_ID` | qa | no | Optional default QA AI Gateway chat profile/model. QA settings versions can override these; model must exactly match the selected AI Gateway profile. |
| `QA_SETTINGS_OPEN` / `QA_ADMIN_USER_IDS` | qa | no | Local QA settings-write allowance. Keep closed by default; enable only for trusted local smoke or configure explicit admin user ids. |
| `MCP_TRANSPORT` / `MCP_SERVER_ALIAS` / `MCP_SERVER_URL` | qa | no | Local Compose defaults to `streamable_http` / `document` / `http://document:8085/mcp` so QA can discover Document report tools. |
| `MCP_SERVER_TOKEN` / `MCP_SERVER_TOKEN_HEADER` | qa | no | Document MCP credential. Defaults to the same `INTERNAL_SERVICE_TOKEN` placeholder and `Authorization` header used by Document MCP. |
| `MCP_TOOL_TIMEOUT` | qa | no | Per-tool timeout for remote MCP calls; defaults to `30s`. |
| `DOCUMENT_DATABASE_URL` | document | yes | Document PostgreSQL DSN. |
| `DOCUMENT_REDIS_ADDR` | document | yes | Redis/asynq endpoint. |
| `DOCUMENT_FILE_SERVICE_URL` | document | yes | Internal File Service URL. |
| `DOCUMENT_FILE_SERVICE_TOKEN` | document | yes | Local service token for File Service calls without gateway request context. |
| `DOCUMENT_AI_GATEWAY_URL` | document | yes | Internal AI Gateway base URL. |
| `DOCUMENT_AI_GATEWAY_PROFILE_ID` | document | yes | Seeded placeholder profile id, `default-chat`. |
| `DOCUMENT_AI_GATEWAY_SERVICE_TOKEN` | document | yes | Local service token for AI Gateway internal profile APIs. |
| `DOCUMENT_MCP_SERVICE_TOKEN` / `DOCUMENT_MCP_TOKEN_HEADER` | document | yes | Streamable HTTP MCP credential; defaults to `INTERNAL_SERVICE_TOKEN` and `Authorization` in local Compose. |
| `AI_GATEWAY_DATABASE_URL` | ai-gateway | yes | AI Gateway PostgreSQL DSN. |
| `AI_GATEWAY_SERVICE_TOKEN_HASHES` | ai-gateway | yes | SHA-256 hashes for allowed service tokens. |
| `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF` | ai-gateway | yes | Local encryption key reference placeholder. |
| `AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY` | ai-gateway | yes | Local encryption key placeholder. |

## Health And Readiness

If the current shell has `HTTP_PROXY`/`HTTPS_PROXY`/`http_proxy`/`https_proxy`,
set `NO_PROXY=localhost,127.0.0.1,::1` before local checks, or use
`curl --noproxy '*'`. Otherwise the request can go through the proxy and return
the proxy's status instead of the local container's status.

Use gateway for the top-level signal:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
Invoke-RestMethod http://localhost:8080/readyz
```

Service-level readiness endpoints:

```powershell
Invoke-RestMethod http://localhost:8001/readyz
Invoke-RestMethod http://localhost:8082/readyz
Invoke-RestMethod http://localhost:8083/readyz
Invoke-RestMethod http://localhost:8084/readyz
Invoke-RestMethod http://localhost:8085/readyz
Invoke-RestMethod http://localhost:8086/readyz
Invoke-RestMethod http://localhost:8087/readyz
```

`gateway /readyz` checks Redis and auth, and verifies owner service URLs are
configured. It does not call Knowledge, QA, Document, or AI Gateway readiness
endpoints and does not prove business workflows such as upload, retrieval, QA
answers, report generation, model profile bootstrap, or real provider calls.
Auth, document, and ai-gateway readiness identify PostgreSQL problems. Compose
health checks identify container-level dependency failures.

Use the targeted smoke checks in
[`docs/runbooks/local-integration.md`](../docs/runbooks/local-integration.md)
for complete cross-service availability. #352 owns the repeatable
Auth/Gateway/Redis smoke, and #125 owns the broader Gateway -> owner service
and MCP/cross-service smoke coverage.

## Seed Data

`seed-local` applies `deploy/seeds/001-local-demo-seed.sql` after Auth,
Knowledge, Document, and QA migrations. `seed-local-ai` applies
`deploy/seeds/002-ai-gateway-model-profiles.sql` after the AI Gateway migration.
Both scripts are idempotent and use deterministic IDs with `ON CONFLICT`.

CI-safe checks validate static seed contracts without starting containers:

```powershell
python scripts/verify_local_seed_contract.py
docker compose --env-file deploy/.env.example -f deploy/docker-compose.yml config --quiet
docker compose --env-file deploy/.env.example -f deploy/docker-compose.yml --profile ai config --quiet
```

The local/manual seed path is the Compose run itself:

```powershell
cd deploy
docker compose --env-file .env.example up -d --build gateway
docker compose --env-file .env.example --profile ai up -d --build ai-gateway
```

Seeded local resources:

| Area | Deterministic resource |
| --- | --- |
| Auth | user `usr_local_admin`, username `admin`, password `LocalDemoAdmin#12345`, role `admin` |
| Auth permissions | `admin:model-profile:write`, `admin:parser-config:write`, `qa:settings:read`, and `qa:settings:write`; `system:admin` is not required for this local admin |
| Knowledge | knowledge base `kb_local_demo`, document `doc_local_demo_seed`, chunk `chunk_local_demo_seed_001` |
| Document | material `22222222-2222-4222-8222-222222222201`, report `22222222-2222-4222-8222-222222222301`, outline `22222222-2222-4222-8222-222222222401` |
| QA | conversation `33333333-3333-4333-8333-333333333301`, user message `33333333-3333-4333-8333-333333333401`, assistant message `33333333-3333-4333-8333-333333333402` |
| AI Gateway | optional placeholder profiles `default-chat`, `default-embedding`, and `default-rerank` |

The local admin password hash in `001-local-demo-seed.sql` is an `argon2id`
PHC string for `LOCAL_ADMIN_PASSWORD=LocalDemoAdmin#12345` using the documented
`argon2id-v1` parameters: `m=65536`, `t=3`, `p=2`, 16-byte salt, and 32-byte
key. For rotation, generate a new local-only `argon2id` hash, update
`deploy/.env.example`, `001-local-demo-seed.sql`, and this README together,
then rerun `seed-local`. Never reuse the demo password or hash in a shared or
long-lived environment.

After the stack is up, verify the seeded admin through Gateway:

```powershell
$body = @{ username = "admin"; password = "LocalDemoAdmin#12345" } | ConvertTo-Json
$session = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/sessions -ContentType "application/json" -Body $body
$session.data.user.roles
$session.data.user.permissions
$token = $session.data.session.accessToken
$headers = @{ Authorization = "Bearer $token" }
Invoke-RestMethod -Uri http://localhost:8080/api/v1/admin/parser-configs -Headers $headers
```

The response should include role `admin` and admin runtime config permissions
such as `admin:model-profile:write`, `admin:parser-config:write`, or
`qa:settings:read`. The
`GET /api/v1/admin/parser-configs` call proves the seeded admin token passes a
Gateway admin route preflight; use `/api/v1/admin/model-profiles` when the
optional AI profile is running.

To remove only the deterministic local demo rows after migrations are present:

```powershell
cd deploy
docker compose --env-file .env.example run --rm seed-local sh -c "psql -v ON_ERROR_STOP=1 -h postgres -U postgres -d postgres -f /seeds/099-local-demo-cleanup.sql"
```

For a full reset, remove volumes and rerun the stack:

```powershell
cd deploy
docker compose --env-file .env.example --profile ai down -v
docker compose --env-file .env.example up -d --build gateway
```

The AI profiles are enabled local placeholders for readiness checks and include
fake encrypted provider credentials. They are not real API keys, so model
invocation still requires operators to configure a real provider key.
Their default provider URL is `http://host.docker.internal:11434/v1`; Compose
maps that hostname to the Docker host for Linux engines with
`host.docker.internal:host-gateway`.

`ai-gateway /readyz` distinguishes these states per purpose:

- `missing`: the chat, embedding, or rerank profile/active credential is absent.
- `placeholder`: the seeded local fake credential is still configured.
- `ok`: a non-placeholder credential is configured for the profile.

`ok` only means the profile configuration no longer matches the known local
placeholder. It does not prove the external provider accepted the key; run the
env-gated real-provider smoke in
`docs/services/ai-gateway/docs/seed-runbook.md` before recording cross-service
AI validation.

## Request Id Troubleshooting

Every service returns or propagates `X-Request-Id`.

```powershell
$rid = "req_local_debug_001"
Invoke-RestMethod http://localhost:8080/readyz -Headers @{ "X-Request-Id" = $rid }
docker compose logs gateway auth knowledge qa document | Select-String $rid
```

For frontend issues, capture the response `requestId` or `X-Request-Id`, then
search gateway logs first. If gateway reports a dependency error, search the
same id in the owner service logs.

## Knowledge Integration Notes

Knowledge active operations are exposed through gateway:

```powershell
# after logging in and setting $token to the returned access token
$headers = @{ Authorization = "Bearer $token"; "X-Request-Id" = "req_knowledge_local_001" }
Invoke-RestMethod "http://localhost:8080/api/v1/knowledge-bases" -Headers $headers
Invoke-RestMethod "http://localhost:8080/api/v1/knowledge-bases/kb_local_demo/documents" -Headers $headers
Invoke-RestMethod "http://localhost:8080/api/v1/documents/<documentId>/chunks" -Headers $headers
Invoke-WebRequest "http://localhost:8080/api/v1/documents/<documentId>/content" -Headers $headers -OutFile .\knowledge-content.bin
Invoke-RestMethod "http://localhost:8080/api/v1/knowledge-queries" -Method Post -Headers $headers -ContentType "application/json" -Body '{"query":"local demo","topK":3}'
```

The default Compose profile validates File Service upload/content handoff,
Parser Service parsing, Redis/asynq enqueue/worker execution, PostgreSQL state,
and gateway request-id propagation. It uses local hashing embeddings and an
in-memory vector index unless `KNOWLEDGE_QDRANT_URL` is set.

For real Qdrant smoke, create or verify the `knowledge_chunks` collection before
setting `KNOWLEDGE_QDRANT_URL=http://qdrant:6333`; otherwise ingestion and query
calls return `502 dependency_error` from the Qdrant adapter. For real AI Gateway
embedding or rerank smoke, start `docker compose --profile ai up -d --build`,
replace the seeded fake provider credential with a usable one, then set
`EMBEDDING_PROVIDER=ai_gateway`, `KNOWLEDGE_AI_GATEWAY_BASE_URL=http://ai-gateway:8086`,
and the relevant profile/model variables.

Knowledge's default local path uses deterministic local hashing embeddings and
empty rerank configuration. Do not count that path as real AI Gateway
embedding/rerank validation.

## Common Dependency Failures

| Symptom | Likely cause | Check |
| --- | --- | --- |
| `gateway /readyz` returns `503 dependency_error` | Redis, auth, or required owner service base URL configuration is not ready | `docker compose ps`, `docker compose logs redis auth gateway` |
| `auth /readyz` returns `postgres unavailable` | Auth migration or PostgreSQL failed | `docker compose logs postgres migrate-auth auth` |
| Knowledge upload returns `502 dependency_error` | File Service, Parser Service, or Redis queue unavailable | `docker compose logs file parser knowledge redis` |
| Knowledge query returns `502 dependency_error` | Qdrant collection missing, AI Gateway embedding/rerank unavailable, or fake provider credential still configured | `docker compose logs knowledge qdrant ai-gateway` |
| Document readyz returns dependency error | Document DB migration failed or DB is unreachable | `docker compose logs migrate-document document postgres` |
| QA message call fails on model invocation | AI Gateway profile is not running, fake local credential is still in use, or host provider is not listening on `host.docker.internal:11434` | `docker compose --profile ai ps`, `docker compose logs ai-gateway qa` |
| MinIO bucket missing | `minio-init` did not complete | `docker compose logs minio minio-init` |
| Host port conflict | Another local process uses a default port | Change the matching `*_PORT` in `deploy/.env` |

## Reset

```powershell
cd deploy
docker compose down -v
docker compose --profile ai down -v
```

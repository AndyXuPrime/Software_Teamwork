# Add independent Docker full-stack startup

## Goal

Add an independent Docker build/start path that can build and run the full local
application stack with a short one-line command, while preserving the existing
host-run local workflow unchanged.

The user wants the Docker path to be convenient for cloud-backed local/demo use,
without forcing contributors to remember a long `docker compose -f ... -f ...`
command or changing the current `./scripts/local/start.sh` behavior. The Docker
path should be cloud-first so local machines do not carry OCR/model/runtime
worker pressure.

## Confirmed Facts

- The current standard local workflow is host-run: Docker starts only shared
  infrastructure, while Go services, Knowledge runtime, and the frontend run on
  the host.
- `deploy/docker-compose.yml` is intentionally infra-only and currently contains
  `postgres`, `redis`, `minio`, `minio-init`, and `elasticsearch`.
- Existing project specs and runbooks say the root Compose baseline must not add
  business services or `build:` entries.
- The repository currently has no `Dockerfile*` files.
- Go service entrypoints are:
  - `services/auth/cmd/server`
  - `services/file/cmd/server`
  - `services/knowledge/cmd/adapter`
  - `services/ai-gateway/cmd/server`
  - `services/qa/cmd/server`
  - `services/document/cmd/server`
  - `services/gateway/cmd/server`
- Services with migrations are auth, file, knowledge, qa, document, and
  ai-gateway.
- Frontend code lives under `apps/web`; production static serving can use the
  existing `deploy/nginx/production.conf` pattern, with `/api/v1` proxied to
  Gateway.
- The existing host-run seed path already supports a PaddleOCR cloud parser
  overlay when `PADDLEOCR_ACCESS_TOKEN` is set.
- AI Gateway local seed overlay can point default chat/embedding/rerank profiles
  at user-provided cloud providers through `AI_GATEWAY_LOCAL_*` variables.

## Requirements

- Preserve existing host-run startup:
  - Do not change `./scripts/local/start.sh` semantics.
  - Do not add business services or `build:` entries to
    `deploy/docker-compose.yml`.
  - Do not require current host-run users to change `.env.local`.
- Add a separate cloud-first Docker path:
  - It must be as independent as practical from the existing host-run path.
  - It must support a short one-line command, recommended as
    `./scripts/docker/start.sh`.
  - It must build business service images and start the application containers.
  - It must default to cloud dependencies rather than local OCR/runtime worker
    execution.
  - It must include migration and seed execution in the container startup path.
  - It must expose a browser/API entrypoint on a stable local port.
- Support cloud OCR / model operation:
  - Docker seed should make PaddleOCR cloud the default parser when
    `PADDLEOCR_ACCESS_TOKEN` is provided.
  - Docker seed should keep the existing AI Gateway cloud-provider bootstrap
    path available through `AI_GATEWAY_LOCAL_SEED_ENABLED=true`.
  - The Docker path should not require local Knowledge runtime worker,
    local OCR models, or local Elasticsearch in its default mode.
- Reuse existing configuration and seed assets where practical, but map
  container networking correctly:
  - service-to-service URLs should use Compose service names, not `localhost`.
  - PostgreSQL, Redis, object storage, and Knowledge runtime URLs must be
    cloud/container-appropriate in the Docker path.
- Keep secrets local:
  - Do not commit real provider credentials.
  - Reuse `.env.local` or an explicit Docker env file template with local demo
    placeholders only.
- Document usage:
  - one-line start
  - stop
  - rebuild/reset
  - relationship to existing host-run startup
- Validation must include config/syntax checks and the existing Docker policy
  check for the unchanged infra baseline.

## Acceptance Criteria

- [ ] `./scripts/local/start.sh` and `deploy/docker-compose.yml` remain valid for
      the current host-run path.
- [ ] A new short command exists for full Docker startup, preferably
      `./scripts/docker/start.sh`.
- [ ] The Docker startup path is separate from the infra-only Compose baseline,
      using new files rather than changing the baseline contract.
- [ ] The new Docker path builds Go services, frontend static assets, and
      startup tooling without requiring manual long Compose arguments.
- [ ] The default Docker path is cloud-first and does not start local OCR,
      local Knowledge runtime worker, or local Elasticsearch.
- [ ] Providing `PADDLEOCR_ACCESS_TOKEN` configures the seeded default parser as
      PaddleOCR cloud for the Docker path.
- [ ] Migration and seed jobs run before app services are expected to serve
      traffic.
- [ ] Gateway and frontend are reachable through the documented local entrypoint
      after startup.
- [ ] Docker/Compose validation commands pass or any unrun runtime validation is
      clearly reported with the reason.

## Out Of Scope

- Replacing the existing host-run workflow.
- Production-grade deployment hardening, Kubernetes manifests, TLS, external
  secret managers, or autoscaling.
- Committing real model provider credentials.
- Making real LLM/embedding/rerank calls pass without user-provided provider
  credentials.
- Provisioning managed cloud PostgreSQL, Redis, object storage, Elasticsearch,
  or external Knowledge runtime instances. The Docker path documents required
  connection variables and consumes them.

## Open Questions

No blocking product questions remain. The user requested a short command with
minimal parameters and approved starting the task.

# Fix cloud docker managed dependency gaps

## Goal

Close the remaining cloud Docker review gaps that prevent the cloud app stack
from working with common managed dependencies. Gateway must be able to connect
to managed Redis endpoints that require ACL username and TLS, and
`DOCKER_SEED_ENABLED=false` must be a real path for pre-provisioned cloud
databases without forcing seed-only model/OCR values.

## Requirements

- Gateway Redis configuration supports `GATEWAY_REDIS_USERNAME` and
  `GATEWAY_REDIS_TLS_ENABLED` in addition to existing addr/password/db.
- Gateway Redis client passes username and TLS configuration into go-redis and
  keeps local defaults unchanged.
- Cloud compose, cloud env template, Gateway README, deploy docs, and runbooks
  document the Gateway Redis username/TLS variables.
- `DOCKER_SEED_ENABLED=false` skips seed-only validation in
  `scripts/docker/start.sh` and does not require seed-only values during
  `docker compose ... config`.
- Seed-only values include `POSTGRES_ADMIN_URL`, cloud OCR token, and cloud
  model provider seed values. When seed remains enabled, the startup preflight
  still fails early for missing required seed inputs.
- QA and Document cloud compose model env fallback must not force
  `AI_GATEWAY_LOCAL_CHAT_MODEL` when seed is disabled and the cloud database is
  pre-provisioned.
- Add focused tests for Gateway config/client Redis options and cloud Docker
  start preflight behavior.
- Preserve existing host-run local startup behavior and root infra-only Compose
  policy.

## Acceptance Criteria

- [x] Gateway config parses `GATEWAY_REDIS_USERNAME` and
      `GATEWAY_REDIS_TLS_ENABLED`, rejects invalid TLS booleans, and keeps
      defaults empty/false.
- [x] Gateway Redis client options include username, password, db, and TLS
      minimum version TLS 1.2 when enabled.
- [x] `deploy/docker-compose.cloud.yml` passes Gateway Redis username/TLS to the
      Gateway container.
- [x] `deploy/docker/cloud.env.example` exposes Gateway Redis username/TLS and
      documents that seed can be disabled for pre-provisioned cloud databases.
- [x] `DOCKER_SEED_ENABLED=false` allows `./scripts/docker/start.sh` preflight
      to proceed without `POSTGRES_ADMIN_URL`, `PADDLEOCR_ACCESS_TOKEN`, or
      `AI_GATEWAY_LOCAL_PROVIDER_*` values when all non-seed required cloud
      values are set.
- [x] `DOCKER_SEED_ENABLED=true` still fails preflight for missing static seed,
      cloud OCR, and enabled AI Gateway provider seed values.
- [x] Cloud compose config validates with `deploy/docker/cloud.env.example`.
- [x] Relevant Go tests, Docker policy checks, shell syntax checks, compose
      config checks, and whitespace checks pass.

## Out Of Scope

- Real cloud Redis/OCR/provider smoke tests, because this environment does not
  have the user's managed service credentials.
- Changing production deployment semantics or adding local heavy dependency
  containers to the cloud stack.

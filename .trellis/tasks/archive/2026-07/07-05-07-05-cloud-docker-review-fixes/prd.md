# Fix cloud Docker review findings

## Goal

Address two PR review findings for the cloud Docker app stack:

- Document service must support common authenticated cloud Redis deployments.
- Docker policy checks must actively constrain the new cloud Compose exception so
  it cannot grow into a local all-in-one infra/runtime stack.

## Requirements

- Keep the existing host-run local startup path unchanged.
- Add Document Redis configuration for password and database selection at
  minimum. Add TLS only if the current codebase already has a clear local
  pattern or asynq support can be wired without broad refactor.
- Surface the new Document Redis cloud variables in:
  - `deploy/docker-compose.cloud.yml`
  - `deploy/docker/cloud.env.example`
  - user-facing Docker/cloud startup docs
- Update `scripts/check_docker_policy.py` so `deploy/docker-compose.cloud.yml`
  is not just path-allowed, but also policy-validated.
- Add policy tests proving cloud Compose rejects local infra/heavy services such
  as PostgreSQL, Redis, MinIO, Elasticsearch, Knowledge runtime worker, and OCR
  services.
- Preserve the approved cloud Compose services:
  `migrate`, `seed`, `auth`, `file`, `knowledge`, `ai-gateway`, `document`,
  `qa`, `gateway`, `web`.

## Acceptance Criteria

- Document service can receive Redis password and DB from environment in the
  cloud Docker path.
- Cloud env template no longer implies only unauthenticated Redis is supported.
- Docker policy checker fails if the cloud Compose file adds local infra or
  heavy runtime/OCR containers.
- Existing root Compose infra-only policy still passes.
- Relevant unit tests and Compose config checks pass.

## Out Of Scope

- Provisioning or testing real managed Redis.
- Replacing the host-run local configuration model.
- Turning the cloud Docker stack into production deployment infrastructure.

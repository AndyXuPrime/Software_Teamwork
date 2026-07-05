# Design

## Document Redis

The review correctly identifies an asymmetry: Gateway already accepts Redis
password/database env, while Document only receives `DOCUMENT_REDIS_ADDR`.

Add Document config fields:

- `DOCUMENT_REDIS_PASSWORD`
- `DOCUMENT_REDIS_DB`

Wire them into the asynq Redis client options used by Document. Keep defaults
compatible with current local startup:

- password defaults to empty
- DB defaults to `0`

Cloud Docker Compose will pass these env vars from `.env.docker.cloud`.

## Cloud Compose Policy

Add a dedicated `validate_cloud_compose` path in
`scripts/check_docker_policy.py` for `deploy/docker-compose.cloud.yml`.

Contracts:

- The cloud Compose service set may contain only approved app/job/web services.
- It may use `build:`.
- It must not contain local infra/heavy services or images for PostgreSQL,
  Redis, MinIO, Elasticsearch, Knowledge runtime worker/API, or OCR containers.
- Root `deploy/docker-compose.yml` remains validated by the existing
  infra-only checker.

Tests should include both allowed cloud Compose and explicit negative examples.

## Compatibility

Existing `.env.local` and host-run defaults keep working because new Document
Redis vars are optional. Cloud users can now configure authenticated managed
Redis without changing service code.

# Design

## Boundary Contract

The repository has two Docker-related startup contracts:

1. Local host-run development remains rooted at `deploy/docker-compose.yml`. That compose file is pull-only infrastructure and may only define PostgreSQL, Redis, MinIO, minio-init, and Elasticsearch.
2. Cloud Docker app stack is the explicit second path rooted at `deploy/docker-compose.cloud.yml` and `scripts/docker/start.sh`. It may build and run app/web containers, but must not start local heavy dependencies. PostgreSQL, Redis, object storage, Elasticsearch/search runtime, Knowledge runtime, OCR, and model provider must remain external/cloud dependencies.

This is a policy change, not a checker loophole. The checker should keep separate validators and error messages for local compose and cloud compose so future changes cannot blur the two.

## Seed Safety Contract

Cloud Docker should be safe after a user copies `deploy/docker/cloud.env.example` and fills cloud endpoints.

- The template defaults `DOCKER_SEED_ENABLED=false`.
- `scripts/docker/start.sh` treats seed-disabled as the normal cloud path and skips seed-only variables.
- If seed is enabled, `scripts/docker/start.sh` validates required seed values and rejects known local/demo placeholder values before `docker compose up`.
- `deploy/docker/full/seed.sh` repeats the placeholder guard inside the seed container so direct compose use cannot bypass the script preflight.

The existing local demo SQL remains unchanged for local host-run startup. Cloud usage of that seed becomes explicit opt-in and guarded.

## Documentation Shape

Docs should use consistent names:

- "local host-run path" for `./scripts/local/start.sh`.
- "cloud Docker app stack" or "second Docker startup path" for `./scripts/docker/start.sh`.

Docs must state that the cloud app stack is not the production/near-production deployment baseline. It is a developer/operator convenience for building and running app containers while externalizing heavyweight dependencies.

## Compatibility

Existing local startup behavior is unchanged. Existing cloud users who intentionally rely on demo seed can still set `DOCKER_SEED_ENABLED=true`, but they must replace all local demo placeholders with real rotated values.

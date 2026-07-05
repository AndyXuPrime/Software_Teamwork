# Design

## Boundary

This task stays inside the existing cloud Docker app stack and Gateway service
configuration. It does not change the root infra-only compose baseline or add
new runtime containers.

## Gateway Redis

Gateway will mirror the managed Redis capability already added to Document:

- `GATEWAY_REDIS_ADDR`
- `GATEWAY_REDIS_USERNAME`
- `GATEWAY_REDIS_PASSWORD`
- `GATEWAY_REDIS_DB`
- `GATEWAY_REDIS_TLS_ENABLED`

`services/gateway/internal/config` owns env parsing and validation. The Redis
platform package owns translating those values to `redis.Options`. TLS uses
`&tls.Config{MinVersion: tls.VersionTLS12}` when enabled. Empty username and
TLS disabled remain the local default.

## Seed Disabled Path

`DOCKER_SEED_ENABLED=false` means the seed container exits successfully without
touching seed inputs. Therefore seed-only variables must not be required by
compose interpolation or the start script preflight when this flag is false.

The start script will split validation into:

- always-required cloud values for the running app stack,
- seed-required values when `DOCKER_SEED_ENABLED` is not false,
- provider-seed-required values when seed is enabled and
  `AI_GATEWAY_LOCAL_SEED_ENABLED` is not false.

`POSTGRES_ADMIN_URL`, `PADDLEOCR_ACCESS_TOKEN`, and
`AI_GATEWAY_LOCAL_PROVIDER_BASE_URL` / `AI_GATEWAY_LOCAL_PROVIDER_API_KEY` /
`AI_GATEWAY_LOCAL_CHAT_MODEL` are treated as seed-only preflight values.

Compose will stop using `${VAR:?set ...}` for seed-only values so
`docker compose ... config` can render the disabled-seed path. The seed script
will keep runtime protection by validating seed-only values when it actually
runs.

QA and Document model env fallbacks will use empty defaults instead of forcing
`AI_GATEWAY_LOCAL_CHAT_MODEL`, allowing pre-provisioned AI Gateway profiles to
own the model when seed is disabled.

## Compatibility

Existing `.env.docker.cloud` files with all provider/OCR values continue to
work. New optional values default to empty or false. Users with pre-provisioned
cloud databases can set `DOCKER_SEED_ENABLED=false` and omit seed-only values,
but must ensure required data, model profiles, parser configs, and demo users
already exist.

## Rollback

The changes are isolated to Gateway env parsing, Redis client options, cloud
compose/env/docs, and Docker start/seed scripts. Reverting the task commit
restores the previous strict seed preflight and Gateway password-only Redis
behavior.

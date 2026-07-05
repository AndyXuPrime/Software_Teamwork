# Fix Docker cloud stack boundary and seed safety

## Goal

Resolve review findings that the cloud Docker app stack must be a deliberate second startup path rather than an infra-only policy bypass, and make cloud seed behavior safe by default.

## Confirmed Facts

- Review finding: `deploy/docker-compose.cloud.yml` adds business service containers and `build:` entries for auth, file, knowledge, ai-gateway, document, qa, gateway, and web. This is acceptable only if the repository explicitly changes the Docker boundary and release/startup policy for a second path.
- Existing local contract remains valuable: root `deploy/docker-compose.yml` must stay infra-only for local host-run development.
- The user explicitly wants a second startup path, not removal of the cloud Docker app stack.
- Review finding: `deploy/docker/cloud.env.example` currently defaults `DOCKER_SEED_ENABLED=true` and contains `local-dev-*` / `change-me` token-style values. `deploy/docker/full/seed.sh` applies `001-local-demo-seed.sql` and related local demo seeds when seed is enabled, which can write demo users/tokens/config into cloud PostgreSQL.
- Existing tests already cover root compose policy, cloud compose heavy-dependency rejection, cloud compose config rendering, and `DOCKER_SEED_ENABLED=false` relaxing seed-only preflight.

## Requirements

- R1: Keep root `deploy/docker-compose.yml` as the only default local Compose path and keep it infra-only.
- R2: Keep `deploy/docker-compose.cloud.yml` as an explicit second Docker startup path that may build app/web containers, but document and enforce that it externalizes PostgreSQL, Redis, object storage, Elasticsearch/search runtime, Knowledge runtime, OCR, and model provider dependencies.
- R3: Update policy wording, docs, and agent-facing guidance so reviewers and future agents do not interpret the cloud app stack as an accidental bypass of the local infra-only rule.
- R4: Make cloud Docker seed safe by default: copying `deploy/docker/cloud.env.example` and filling cloud endpoints must not run local demo seed automatically.
- R5: When cloud Docker seed is explicitly enabled, reject local/demo placeholder secrets such as `local-dev-*`, `local-demo-*`, `change-me`, and angle-bracket placeholders before compose starts.
- R6: Preserve the one-line startup path after `.env.docker.cloud` is prepared: `./scripts/docker/start.sh`.

## Acceptance Criteria

- [x] `deploy/docker/cloud.env.example` defaults to `DOCKER_SEED_ENABLED=false`.
- [x] `scripts/docker/start.sh` succeeds for a valid seed-disabled cloud env without requiring seed-only values.
- [x] `scripts/docker/start.sh` fails before compose when cloud seed is enabled and required secret/token values still use local demo placeholders.
- [x] `deploy/docker/full/seed.sh` refuses to apply local demo seed to cloud DB unless seed is explicitly enabled and placeholder secret checks pass.
- [x] Docker policy tests assert both sides of the boundary: root compose cannot build app services, while only `deploy/docker-compose.cloud.yml` may build the allowed app/web stack and may not add local heavy dependencies.
- [x] README, deploy runbook, local integration runbook, Docker image pull runbook, architecture docs, Trellis spec, and `AGENTS.md` describe the two supported paths consistently.
- [x] Required Docker checks pass locally: policy checker, related unit tests, root compose config, cloud compose config, disabled-seed cloud compose config, shell syntax, and diff whitespace check.

## Notes

- Out of scope: designing a production deployment pipeline, changing cloud image publishing strategy, or removing the cloud Docker app stack requested by the user.

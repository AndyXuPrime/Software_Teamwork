# Fix cloud Docker CI and seed-disabled model fallback

## Goal

Resolve review findings that cloud Dockerfile/support-file changes must trigger Docker policy checks and seed-disabled cloud startup must not inject seed-only model placeholders into Document/QA runtime configuration.

## Confirmed Facts

- `.github/workflows/docker-deploy-checks.yml` is triggered by `deploy/**`, but its detect script only sets `has-policy-check` for compose files, selected scripts, explicit policy paths, and non-approved Docker artifacts. Approved cloud build files under `deploy/docker/full/` such as `go-service.Dockerfile`, `go-tools.Dockerfile`, `frontend.Dockerfile`, `nginx.conf`, and `.dockerignore` can change without running policy or compose config checks.
- `deploy/docker-compose.cloud.yml` currently maps `DOCUMENT_AI_GATEWAY_MODEL` and QA `MODEL_ID` through `${AI_GATEWAY_LOCAL_CHAT_MODEL:-}` fallback.
- `deploy/docker/cloud.env.example` defaults `DOCKER_SEED_ENABLED=false`, but still sets `AI_GATEWAY_LOCAL_CHAT_MODEL=<cloud-chat-model>`. Because seed-only provider variables are intentionally not validated when seed is disabled, this fallback can inject a placeholder runtime model into Document/QA and override AI Gateway profile behavior.
- User intent remains to keep the second Docker startup path, but make it bounded, tested, and safe by default.

## Requirements

- R1: Any change under `deploy/docker/full/**` must trigger Docker policy checks in `docker-deploy-checks.yml`.
- R2: The detect logic must be covered by a regression test or executable local harness so a future cloud Dockerfile/support-file change cannot silently skip policy checks.
- R3: In the seed-disabled cloud path, Document and QA must not derive runtime model overrides from seed-only `AI_GATEWAY_LOCAL_CHAT_MODEL`.
- R4: Cloud env/docs should make runtime model overrides explicit through `MODEL_ID` and `DOCUMENT_AI_GATEWAY_MODEL`; provider seed variables should be required only for explicit seed.
- R5: Preserve local host-run behavior where `scripts/local/start.sh` may align local `MODEL_ID` / `DOCUMENT_AI_GATEWAY_MODEL` to `AI_GATEWAY_LOCAL_CHAT_MODEL`.

## Acceptance Criteria

- [x] Changes to `deploy/docker/full/go-service.Dockerfile`, `frontend.Dockerfile`, `go-tools.Dockerfile`, `nginx.conf`, `.dockerignore`, or scripts under `deploy/docker/full/` trigger `has-policy-check=true` in the workflow detect logic.
- [x] `deploy/docker-compose.cloud.yml` renders `MODEL_ID` and `DOCUMENT_AI_GATEWAY_MODEL` as empty strings by default when only `deploy/docker/cloud.env.example` is used.
- [x] `deploy/docker/cloud.env.example` no longer provides `<cloud-chat-model>` through a live `AI_GATEWAY_LOCAL_CHAT_MODEL` assignment on the seed-disabled default path.
- [x] Unit or harness tests cover both the workflow detect gap and seed-disabled model fallback.
- [x] Docker policy, compose config, shell syntax, workflow syntax/actionlint, and diff whitespace checks pass.

## Notes

- Out of scope: changing the local host-run AI Gateway seed overlay behavior, replacing AI Gateway profile semantics, or removing the cloud Docker app stack.

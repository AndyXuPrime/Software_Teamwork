# Implementation Plan

## Step 1: Baseline and Specs

- Confirm current branch/worktree and PR remote state.
- Read backend, shared guide, Docker, and local integration specs.
- Inventory current `services/knowledge-runtime`, `services/knowledge`, and deploy Compose surfaces.

## Step 2: Protect Runtime Surface

- Introduce an explicit runtime route allowlist.
- Register only route modules needed by Knowledge core behavior.
- Add unit tests proving excluded RAGFlow route modules are not registered.
- Remove route modules only after import/reference checks pass.

## Step 3: Remove Unused Runtime Subsystems

- Remove upstream RAGFlow MCP runtime server/client surface after confirming the Go adapter owns Knowledge MCP.
- Remove entrypoint, Dockerfile, dependency, docs, DB service, and tests tied only to deleted subsystems.
- Re-run import/reference searches after each deletion batch.

## Step 4: Container Review

- Keep root Compose as local/demo integration, not infra-only.
- Verify `knowledge-v2` profile includes adapter, runtime API, runtime worker, and infrastructure.
- Keep image tags pinned and avoid `GOSUMDB=off`.
- Validate Docker policy and Compose config.

## Step 5: Tests and E2E

- Run Go tests/build for `services/knowledge`.
- Run targeted Python runtime route/parse/retrieval tests.
- Run Docker policy and related script tests.
- Run Compose config checks for default, `knowledge-v2`, and affected profiles.
- Run `DL_T_673-1999.pdf` E2E when the file is available.

## Step 6: PR

- Commit in conventional format.
- Push to `origin/L1nggTeam/feat/ragflow-runtime-vendor`.
- Confirm PR targets `develop`.
- Check GitHub Actions and Codex Review for Severe/Critical findings.

## Validation Commands

- `cd services/knowledge && go test ./...`
- `cd services/knowledge && go build ./cmd/adapter`
- `cd services/knowledge-runtime && PYTHONPATH=. uv run --no-project --with pytest --with pytest-asyncio python -m pytest <targeted tests> -q`
- `python3 scripts/check_docker_policy.py`
- `python3 -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment scripts.tests.test_local_seed_contract`
- `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet`
- `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example --profile knowledge-v2 config --quiet`
- `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example --profile ai config --quiet`

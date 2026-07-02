# Implementation Plan

## Setup

1. Checkout PR #514.
2. Rebase the PR branch onto latest `upstream/develop`.
3. Resolve conflicts by preserving PR #514 functionality and latest `develop` runtime policy.
4. Record whether force-with-lease will be required.

## Code And Docs Changes

1. Inspect `docs/runbooks/local-integration.md` around robot-commented sections.
2. Remove or rewrite:
   - root Compose business-service startup examples,
   - `--profile ai`,
   - `--build` startup paths,
   - service-level Compose local environments,
   - business-service `docker compose logs` guidance.
3. Replace with host-run script and `.local/logs/` guidance.
4. Inspect `apps/web/src/features/reports/report-generation.queries.ts`.
5. Determine available job/event timestamp or retry metadata from generated types and existing code.
6. Implement bounded failed-state polling.
7. Keep `apps/web/src/pages/reports/generate/page.tsx` outline fixes intact.
8. Check Document service files for conflicts after rebase; avoid unnecessary changes.

## Validation

Run these checks from repo root unless noted:

1. `git diff --check`
2. `python scripts/verify_local_seed_contract.py`
3. `python scripts/check_docker_policy.py`
4. `python -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment scripts.tests.test_local_seed_contract`
5. `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet`
6. `bun run --cwd apps/web check`
7. `bun run --cwd apps/web build`
8. `cd services/document && go test ./...`
9. `cd services/document && go build ./cmd/server`

If `go` is unavailable locally, report that explicitly and rely on Go Services CI after push.

## PR Updates

1. Confirm commit subjects satisfy `.github/workflows/commitlint.yml`.
2. Push to the contributor fork branch. Use `--force-with-lease` if rebase rewrites history.
3. Update PR body verification notes if local command results changed.
4. Check PR #514 status after push and inspect any new failing logs.

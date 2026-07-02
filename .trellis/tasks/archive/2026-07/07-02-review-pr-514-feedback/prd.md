# Review and fix PR 514 feedback

## Goal

Bring PR #514 onto the latest `upstream/develop`, address the Codex PR Review findings, and preserve the PR's intended AI report generation fixes:

- Document service DOCX formatting improvements.
- Document AI prompt quality and timeout/retry behavior improvements.
- Frontend outline rendering and report job polling behavior.
- Local integration runbook updates that match current host-run architecture.

## Confirmed Facts

- PR: `Sakayori-Iroha-168/Software_Teamwork#514`.
- Base branch: `develop`.
- Head branch: `bingyuwu645-sudo:wubin/docs/report-ai-generation-setup`.
- `maintainerCanModify` is `true`, so this workspace can push fixes to the PR branch.
- Local `develop` has been fast-forwarded to latest `upstream/develop` at planning time.
- PR currently touches:
  - `apps/web/src/features/reports/report-generation.queries.ts`
  - `apps/web/src/pages/reports/generate/page.tsx`
  - `docs/runbooks/local-integration.md`
  - `services/document/internal/platform/aigateway/chat_client.go`
  - `services/document/internal/platform/aigateway/profile_client.go`
  - `services/document/internal/service/docx_generator.go`
  - `services/document/internal/service/report_generation_service.go`
  - `services/document/internal/worker/worker.go`
- There are no inline review threads. Review feedback is in top-level Codex PR Review comments.
- Latest CI shown on PR #514 is green, but the bot review still flags correctness/policy concerns.

## Review Feedback To Address

1. High severity: `docs/runbooks/local-integration.md` still includes root-level Compose commands that start or build business services, including `--profile ai up -d --build gateway ai-gateway`, and references Docker logs for business service containers.
2. High severity: `docs/runbooks/local-integration.md` still recommends service-level Compose for QA/Auth/Gateway and Document local environments.
3. Medium severity: `apps/web/src/features/reports/report-generation.queries.ts` keeps polling forever when report jobs or events remain in a terminal `failed` state. This conflicts with backend behavior where validation failures use `asynq.SkipRetry` and retry-exhausted jobs will not recover automatically.

## Requirements

- R1. Rebase or otherwise update PR #514 on top of the latest `upstream/develop`.
- R2. Preserve all intended functional changes from PR #514 unless a change directly conflicts with current repository policy or latest `develop`.
- R3. Update local integration documentation so current instructions only use:
  - `cp deploy/.env.example deploy/.env`
  - `./scripts/local/dev-up.sh`
  - `./scripts/local/run-backend.sh`
  - `cd apps/web && bun install && bun run dev`
  - host-run service logs under `.local/logs/`
- R4. Remove or rewrite current-path documentation that suggests root Compose business services, service-level Compose, migration containers, seed containers, `--profile ai`, or `--build`.
- R5. Adjust frontend report polling so terminal failed jobs/events do not cause unbounded background polling, while preserving the intended ability to observe automatic retry transitions when there is credible evidence that a retry may still occur.
- R6. Keep PR branch commit history compatible with repository commitlint.
- R7. Update PR description if verification commands or risk notes change materially.

## Acceptance Criteria

- [ ] PR branch is based on latest `upstream/develop`.
- [ ] `rg` confirms current startup docs do not contain forbidden active-path tokens such as `docker compose --profile ai`, `docker compose up --build`, service-level Compose startup guidance, or `export AI_GATEWAY_DATABASE_URL`.
- [ ] `python scripts/verify_local_seed_contract.py` passes.
- [ ] `python scripts/check_docker_policy.py` passes.
- [ ] `python -m unittest scripts.tests.test_check_docker_policy scripts.tests.test_check_docker_environment scripts.tests.test_local_seed_contract` passes.
- [ ] `docker compose -f deploy/docker-compose.yml --env-file deploy/.env.example config --quiet` passes.
- [ ] `bun run --cwd apps/web check` passes.
- [ ] `bun run --cwd apps/web build` passes or any pre-existing warning is documented.
- [ ] `cd services/document && go test ./...` and `go build ./cmd/server` pass when Go is available; if unavailable locally, CI result and residual risk are reported.
- [ ] `git diff --check` passes.
- [ ] PR #514 is pushed back to `bingyuwu645-sudo:wubin/docs/report-ai-generation-setup` with `--force-with-lease` only if history is rewritten.

## Out Of Scope

- Adding new report generation features beyond PR #514's existing scope.
- Reintroducing any business-service Docker baseline.
- Building a new cross-service E2E smoke framework.
- Resolving unrelated historical test report wording.

## Open Questions

None. Review feedback is concrete enough to proceed.

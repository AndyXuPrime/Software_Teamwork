# CI path filters and service check matrix

## Goal

Complete issue #123 by making GitHub Actions run checks for the changed area only: frontend changes run Bun check/build, Go service changes run the matching service test/build commands, and Docker/deploy changes run lightweight packaging validation without weakening collaboration guardrails.

## Requirements

- Keep PR Guard, Commitlint, and Auto Label workflows separate and unchanged in behavior.
- Reuse the landed frontend CI for `apps/web/**` and frontend dependency/config changes.
- Change Go service CI so `services/<service>/**` changes run only the matching landed Go service matrix entry.
- Preserve the minimum Go commands for `ai-gateway`, `auth`, `document`, `file`, `gateway`, `knowledge`, and `qa`: `go test ./...` and `go build ./cmd/server`.
- Preserve the QA extra command: `go build ./cmd/agent`.
- Keep Knowledge repository lifecycle integration test, but run it only when Knowledge or the Go Services workflow changes.
- Add Docker/deploy path checks that validate changed Dockerfiles / Compose files without pushing images or requiring secrets.
- Update CI documentation to explain which paths trigger which checks and to keep PR target rules tied to `CONTRIBUTING.md`.

## Acceptance Criteria

- [ ] A PR changing only `apps/web/**` runs frontend CI and does not run all backend service tests.
- [ ] A PR changing `services/auth/**` runs at least `cd services/auth && go test ./...` and `go build ./cmd/server`.
- [ ] A PR changing `services/qa/**` also runs `go build ./cmd/agent`.
- [ ] Workflow changes can force the relevant full matrix where needed.
- [ ] Docker/Compose/deploy checks are path-scoped and do not push images from pull requests.
- [ ] PR base remains `develop`; collaboration rules are not rewritten.
- [ ] CI docs describe path-to-check behavior.

## Definition of Done

- Workflow YAML parses.
- Documentation reflects current workflow behavior, not future target behavior.
- `git diff --check` passes.
- Local syntax checks for embedded scripts pass where practical.

## Technical Approach

- Use a lightweight `detect` job in `.github/workflows/go-services.yml` to inspect changed files and emit a JSON service matrix.
- Use GitHub Actions `fromJSON()` for the service matrix and skip the service job when no service is selected.
- Reuse `.github/workflows/frontend.yml` for Bun install, `check`, `build`, unit tests, and Playwright smoke.
- Add `.github/workflows/docker-deploy-checks.yml` for path-scoped Dockerfile build validation and Compose config validation.
- Update `README.md`, `docs/testing/strategy.md`, `docs/architecture/technology-decisions.md`, `docs/architecture/current-capability-matrix.md`, and `.trellis/spec/cicd.md`.

## Decision (ADR-lite)

**Context**: The repository has independent Go modules per service and no root Go module. The previous all-service matrix made unrelated service changes expensive and caused frontend-only changes to rely only on local checks.

**Decision**: Keep separate product workflows, add path-aware detection inside Go Services, and add frontend/Docker-deploy workflows with scoped triggers.

**Consequences**: The workflows stay simple and readable. A changed shared workflow can still run broader checks. Future shared Go code or root Compose work must update path detection rules.

## Out of Scope

- No branch protection setting changes.
- No image push or deployment automation.
- No new frontend test framework dependencies.
- No cross-service E2E smoke implementation.

## Technical Notes

- Issue: https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/123
- Base branch: refreshed to latest `origin/develop` at `306e415` during implementation.
- Relevant specs/docs read:
  - `CONTRIBUTING.md`
  - `docs/collaboration/frontend-workflow.md`
  - `docs/collaboration/repository-settings.md`
  - `docs/architecture/technology-decisions.md`
  - `.trellis/spec/cicd.md`
  - `.trellis/spec/backend/quality-guidelines.md`
  - `.trellis/spec/frontend/quality-guidelines.md`
  - `.trellis/spec/frontend/directory-structure.md`
  - `docs/testing/strategy.md`

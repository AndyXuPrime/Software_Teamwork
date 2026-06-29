# Complete Issue 147 Go 1.25 Baseline

## Goal

Unify all landed Go services on the documented Go 1.25 baseline so current
service modules, Docker builds, and CI documentation no longer conflict with
the repository technology baseline or `goose@v3.27.1` requirements.

## Requirements

- Update every landed service `go.mod` under `services/*/` to use `go 1.25.0`.
- Update every existing service Dockerfile to use `golang:1.25-alpine` or an
  equivalent fixed Go 1.25 image.
- Update service README text that still describes Go 1.22 or Go 1.23 as the
  service baseline.
- Add or update GitHub Actions CI so changed Go services can be verified with a
  Go 1.25 toolchain using service-local `go test ./...` and
  `go build ./cmd/server`.
- For QA, also verify `go build ./cmd/agent` if the agent command exists.
- Keep the change scoped to version baseline, validation wiring, and directly
  related documentation.

## Acceptance Criteria

- [x] `services/*/go.mod` no longer declares `go 1.22` or `go 1.23.0`.
- [x] Existing Dockerfiles no longer use `golang:1.22-*` or `golang:1.23-*`.
- [x] Each changed service runs `go test ./...`.
- [x] Each changed service runs `go build ./cmd/server`.
- [x] QA runs `go build ./cmd/agent` when `services/qa/cmd/agent` exists.
- [x] PR body lists validation results and links `Closes #147`.

## Definition of Done

- Backend checks for all affected services have been run or any toolchain
  limitation is clearly reported.
- Trellis task is archived after work commits.
- Branch is pushed to the fork and a PR targets upstream `develop`.

## Technical Approach

Use the existing repository conventions: service-local Go modules under
`services/<service>/`, existing Dockerfiles only, and GitHub Actions with
`actions/setup-go` only if a Go service CI workflow is missing. Do not create a
root Go module or introduce shared backend packages.

## Out of Scope

- Upgrading third-party Go dependencies except where `go mod tidy` requires
  metadata changes.
- Fixing unrelated tests or service behavior beyond what is needed to validate
  the Go baseline.
- Changing local Compose `latest` images unless they are Go build images.

## Technical Notes

- Issue: <https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/147>
- Authority: `docs/architecture/technology-decisions.md`.
- Referenced issue authority `private/doc-update-tasks-20260629.md` is not
  present in this checkout.
- Current local toolchain observed before implementation: `go1.26.4
  windows/amd64`; it can validate `go 1.25.0` modules but is not the exact Go
  1.25 toolchain.
- Existing Go modules at task start: `auth`, `document`, `file`, `gateway`,
  `knowledge`, `qa`.
- Existing service Dockerfiles at task start: `document`, `qa`.

## Validation

- `rg -n "go 1\.(22|23)|golang:1\.(22|23)|Go 1\.(22|23)|httpRouter: Go 1\.(22|23)|older Go module|Knowledge-specific|后续迁移 PR|从 Go 1\.22 迁移" services docs .github -g "!apps/web/**"` returned no matches.
- `go test ./...` passed in `services/auth`, `services/document`, `services/file`, `services/gateway`, `services/knowledge`, and `services/qa`.
- `go build ./cmd/server` passed in all six landed Go services.
- `go build ./cmd/agent` passed in `services/qa`.
- Local verification used `go1.26.4 windows/amd64`; CI is configured with `go-version: '1.25.x'`.
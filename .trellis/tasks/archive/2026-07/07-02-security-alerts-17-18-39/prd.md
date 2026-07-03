# Resolve GitHub security alerts 17 18 39

## Goal

Resolve the three open GitHub security alerts reported against
`Sakayori-Iroha-168/Software_Teamwork` develop at commit
`736acde080ea27e58cecd9784953a4fc7bb36be6`, without creating duplicate GitHub
issues for alerts that can be handled directly in this task.

Source alerts:

- Dependabot alert 39:
  `https://github.com/Sakayori-Iroha-168/Software_Teamwork/security/dependabot/39`
- Code scanning alert 18:
  `https://github.com/Sakayori-Iroha-168/Software_Teamwork/security/code-scanning/18`
- Code scanning alert 17:
  `https://github.com/Sakayori-Iroha-168/Software_Teamwork/security/code-scanning/17`
- Local startup issue:
  `.local/logs/gateway.log` and `.local/logs/auth.log` show host-run Go
  dependency downloads timing out against `https://proxy.golang.org/...`.

The repository was synchronized with latest `upstream/develop` before planning.
All three security alerts remain open on the latest baseline, so this task owns
fixing them rather than only recording them as already covered. The local Go
proxy timeout is a repository startup-default gap and is included as a fourth
issue in this task instead of creating a separate issue.

## Confirmed Facts

- Alert 39 is a medium severity Dependabot alert for transitive
  `golang.org/x/net` in `services/file/go.mod`; vulnerable range is `< 0.55.0`,
  first patched version is `0.55.0`, advisory is `GHSA-5cv4-jp36-h3mw` /
  `CVE-2026-25680`.
- Alert 18 is CodeQL `go/allocation-size-overflow` in
  `services/qa/internal/repository/postgres.go:609`, where
  `make(map[string]any, len(item.Metadata)+1)` uses a size derived from a
  potentially large metadata map. The traced input source is
  `services/qa/internal/service/agent/loop.go:248`.
- Alert 17 is CodeQL `js/insecure-randomness` in
  `apps/web/src/pages/qa/chat/page.tsx`; `nextId()` uses `Math.random()` and
  its output is used for client-side QA message/session IDs.
- `scripts/local/dev-up.sh` and `scripts/local/run-backend.sh` both source
  `deploy/.env`. `dev-up.sh` uses host-run `go run` for goose migrations, and
  `run-backend.sh` uses host-run `go run ./cmd/server` for Go services.
- `deploy/.env.example` currently provides a default `UV_DEFAULT_INDEX` for
  Parser/uv and Docker registry rewrites for infrastructure images, but no
  `GOPROXY` / `GOSUMDB` defaults for host-run Go module downloads.
- The observed `.local/logs/gateway.log` failure includes
  `github.com/prometheus/client_golang@v1.23.2` and
  `github.com/redis/go-redis/v9@v9.21.0` timing out at `proxy.golang.org`.
  The observed `.local/logs/auth.log` failure includes `golang.org/x/crypto`
  and `github.com/jackc/pgx/v5` timing out at `proxy.golang.org`.
- Current branch for the fix is `fix/security-alerts-17-18-39`, targeting
  `develop`.

## Requirements

- R1: Update the File service dependency graph so `golang.org/x/net` resolves
  to `v0.55.0` or newer in `services/file/go.mod` / `go.sum`.
- R2: Remove the CodeQL allocation-size-overflow finding by avoiding allocation
  capacity arithmetic derived directly from untrusted citation metadata length.
- R3: Preserve citation metadata behavior: copy user metadata, remove
  `attachmentId` / `attachment_id`, and re-add the canonical trimmed
  `attachmentId` when present.
- R4: Remove the insecure `Math.random()` ID generator from the QA chat page and
  replace it with a browser-supported cryptographically secure random source.
- R5: Keep the frontend change local to the QA chat flow and avoid new
  dependencies.
- R6: Add focused regression coverage where the repository already has a
  suitable unit-test surface.
- R7: Provide repository-owned host-run Go module proxy defaults through
  `deploy/.env.example`, using the existing single-default-config model.
- R8: Document that `GOPROXY` / `GOSUMDB` affect Go module downloads for
  migrations, Go service startup, and Go checks; keep this separate from Docker
  registry rewrite and `UV_DEFAULT_INDEX`.
- R9: Extend the local seed/startup contract checker so future changes cannot
  remove the Go proxy defaults silently.

## Acceptance Criteria

- [ ] `gh api` or equivalent confirms alerts 17, 18, and 39 were investigated
      against current GitHub alert data, not guessed from stale context.
- [ ] `services/file` no longer resolves `golang.org/x/net` below `v0.55.0`.
- [ ] CodeQL pattern in `services/qa/internal/repository/postgres.go:609` is
      removed without changing persisted citation metadata semantics.
- [ ] Frontend QA chat IDs are generated without `Math.random()`.
- [ ] Focused tests cover the QA citation metadata helper.
- [ ] Host-run Go startup and migration commands inherit `GOPROXY` / `GOSUMDB`
      from `deploy/.env` copied from `deploy/.env.example`.
- [ ] README/runbook text explains Go module proxy timeout triage separately
      from Parser uv and Docker image pull triage.
- [ ] Relevant checks pass or any skipped check is reported with a concrete
      reason.
- [ ] Trellis task is archived after verification and commit.

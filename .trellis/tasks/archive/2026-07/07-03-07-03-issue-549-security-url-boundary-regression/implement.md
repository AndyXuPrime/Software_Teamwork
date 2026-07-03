# Implementation Plan

## Checklist

- [ ] Confirm branch and base commit after fetching latest `upstream/develop`.
- [ ] Inspect #533/#546 code surfaces and existing tests.
- [ ] Run or add focused checks for File, QA, Gateway, and frontend alert
      boundaries.
- [ ] Run issue-required validation commands:
      - `cd services/file && go test ./... && go build ./cmd/server`
      - `cd services/qa && go test ./...`
      - `cd services/gateway && go test ./internal/platform/authclient ./internal/config ./internal/http`
      - `cd services/gateway && go build ./cmd/server`
      - `bun run --cwd apps/web check`
      - `bun run --cwd apps/web test:unit`
      - `git diff --check`
- [ ] Query available GitHub alert status via authenticated CLI/API when
      possible; record inaccessible items as limitations.
- [ ] Create
      `docs/testing/reports/2026-07-03/security-url-boundary-regression-test-report.md`.
- [ ] Review report text for mojibake and sensitive data.
- [ ] Commit with a Conventional Commits message.

## Risk Points

- Issue #549 still contains a stale `Blocked` status even though #533/#546 are
  closed. Record this in the report and PR body.
- GitHub code scanning/dependabot alert details may not be visible through the
  current token. Do not invent alert states.
- Frontend checks may require installed Bun dependencies. Run `bun install` only
  if needed and keep lockfile changes only when they are real.

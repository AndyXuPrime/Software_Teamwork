# Implementation Plan

## Phase 1: Planning

- [x] Create Trellis task.
- [x] Switch to `Test/test/qa-sse-behavior-verification` from `upstream/develop`.
- [x] Read issue #498, QA docs, testing strategy, report template, local integration runbook, and relevant QA SSE code.
- [x] Write PRD, design, implementation plan, and real context manifests.

## Phase 2: Environment Readiness

- [x] Record baseline:
  - `git status --short --branch`
  - `git rev-parse HEAD`
  - `git rev-parse upstream/develop`
  - `go version`
  - `docker version`
  - `bash --version`
- [x] Check whether `deploy/.env` exists. If absent, create it from `deploy/.env.example`.
- [x] Start or verify infra:
  - `bash scripts/local/dev-up.sh`
- [x] Start or verify host backend:
  - `bash scripts/local/run-backend.sh`
- [x] Check readiness:
  - `curl --noproxy '*' -fsS http://localhost:8080/healthz`
  - `curl --noproxy '*' -fsS http://localhost:8080/readyz`
  - `curl --noproxy '*' -fsS http://localhost:8084/readyz`
  - `curl --noproxy '*' -fsS http://localhost:8086/readyz`

## Phase 3: QA SSE Verification

- [x] Login with the local demo admin through Gateway and capture a token without committing it.
- [x] Create or select a QA session.
- [x] Capture a real SSE response with `Accept: text/event-stream`.
- [x] Parse the raw SSE file for:
  - response headers;
  - event names;
  - sequence order;
  - `responseRunId`;
  - `answer.delta` count and timing;
  - `toolCallId` pairing.
- [x] Query replay events for the captured `responseRunId`.
- [x] Run validation/error checks for invalid session id and empty message.
- [x] Start a long-running stream and interrupt it if the environment supports it, then verify the run status is `cancelled`.
- [x] Scan raw SSE and relevant logs for forbidden sensitive patterns.

## Phase 4: Report And Review

- [x] Create `docs/testing/reports/2026-07-03/qa-sse-behavior-test-report.md` from the template.
- [x] Include each test item as pass/fail/blocked/skipped with command evidence.
- [x] If behavior fails, create or link follow-up issue(s) and record the classification.
- [x] Run verification:
  - `git diff --check`
  - `cd services/qa && go test ./...`
  - `cd services/qa && go build ./cmd/server`
  - `cd services/qa && go build ./cmd/agent`
- [x] Review changed files for secrets before commit.
- [ ] Commit with Conventional Commit.
- [ ] Push branch to fork and open PR to `develop` with `Closes #498`.

## Risk Notes

- The default local AI Gateway profile may be a placeholder; if no real provider is configured, the report must say that the real provider path was not run.
- Static code suggests `answer.delta` may be emitted once after final content is assembled. Real test evidence determines whether this becomes a failed/transferred test item.
- Do not paste bearer tokens, API keys, provider raw errors, prompts, or full document text into committed artifacts.

# T-015 Security Alert And Internal URL Boundary Regression Test Report

## 0. Basic Information

| Item | Record |
| --- | --- |
| Report date | 2026-07-03 |
| Test task / Issue | T-015 / #549 |
| Owner | @AndyXuPrime |
| Assistants | Codex |
| Scope | Dependabot alert 39, CodeQL alerts 17 and 18, Gateway Auth base URL boundary regression |
| Tested branch | Test/test/security-alert-url-boundary-regression |
| Tested commit | PR branch rebased on upstream/develop @ 3e3565a6b3e4a9c9691161a740da2c16af6f339a plus this report and one Gateway test-case addition |
| Base branch | upstream/develop @ 3e3565a6b3e4a9c9691161a740da2c16af6f339a |
| Environment | Local Windows PowerShell, Go 1.25 modules, Bun/Vitest frontend checks, GitHub CLI |
| Conclusion | Passed after environment retry; first-run failures were caused by Go proxy timeout and transient test-load timeouts, then passed on rerun |

## 1. Test Goals

- Verify #533 security alert fixes for Dependabot alert 39 and CodeQL alerts 17 and 18.
- Verify #546 Gateway Auth base URL validation rejects unsafe URL forms and still accepts normal internal/local http(s) base URLs.
- Run the issue-requested File, QA, Gateway, frontend, and diff checks.
- Record observable GitHub Security alert state without claiming a rescan result beyond what the GitHub API returned.

This round does not close GitHub Security alerts directly and does not expand into unrelated authorization matrix testing.

## 2. Test Basis

| Type | Link or file | Usage |
| --- | --- | --- |
| Issue | #549 | Source task and acceptance scope |
| Dependency issue | #533 | Source for alerts 17, 18, and 39 |
| Dependency issue | #546 | Source for Gateway Auth base URL validation |
| Test strategy | docs/testing/strategy.md | Required commands and report archival rule |
| Backend quality spec | .trellis/spec/backend/quality-guidelines.md | Security boundary and service-local check rules |
| Frontend quality spec | .trellis/spec/frontend/quality-guidelines.md | Frontend check/test command rules |
| Local env defaults | deploy/.env.example | Go module proxy retry used GOPROXY and GOSUMDB defaults |

## 3. Scope And Environment

### Tested Scope

- `services/file/go.mod` and File service tests/build for `golang.org/x/net`.
- QA citation snapshot and metadata sanitization behavior.
- QA frontend chat ID generation in `apps/web/src/pages/qa/chat/page.tsx`.
- Gateway Auth client URL validation in `services/gateway/internal/platform/authclient`.
- GitHub Dependabot and CodeQL alert state through `gh api`.

### Not Tested

- Manual GitHub Security alert closure or CodeQL rescanning.
- Full live local integration stack.
- Real provider, MinIO, Redis, PostgreSQL, or external service smoke paths beyond service-local tests.

### Environment Notes

- Initial File command used the default Go proxy path and failed on `proxy.golang.org` network timeout.
- Retry used project defaults from `deploy/.env.example`: `GOPROXY=https://goproxy.cn,direct` and `GOSUMDB=sum.golang.google.cn`.
- Initial QA and frontend full-suite runs saw transient load-related failures; targeted reruns, low-worker reruns, and final default reruns passed.
- After PR creation, issue #549 was updated from `Status=Blocked` / `Risk=Blocked` to `Status=Review` / `Risk=Low` because dependencies #533 and #546 are closed and this regression PR is ready for review.
- After upstream `develop` advanced to `3e3565a6`, this PR branch was rebased and the issue-required checks were rerun on 2026-07-03.

## 4. Test Case Matrix

| ID | Category | Case / Scenario | Expected Result | Actual Result | Conclusion |
| --- | --- | --- | --- | --- | --- |
| TEST-001 | Dependabot | Alert 39: File resolves patched `golang.org/x/net` | `golang.org/x/net` is patched and File tests/build pass | `go list -m` returned `golang.org/x/net v0.55.0`; File tests/build passed after Go proxy retry | pass |
| TEST-002 | CodeQL | Alert 18: QA citation metadata allocation/sanitization | Metadata handling is tested, sensitive fields are removed, safe fields and attachment behavior remain | QA service tests passed; existing tests cover sanitized metadata and attachment citation round trips | pass |
| TEST-003 | CodeQL | Alert 17: QA chat IDs avoid `Math.random()` | QA chat local IDs use Web Crypto or equivalent secure randomness | `rg` found `crypto.randomUUID` / `crypto.getRandomValues` in QA chat and no `Math.random()` hit | pass |
| TEST-004 | Gateway security | #546 rejects unsafe `GATEWAY_AUTH_BASE_URL` | Reject non-http(s), credentials, query, fragment, relative/missing scheme, missing host | Existing table tests cover unsafe forms; this PR adds explicit missing-host case `http:///internal` | pass |
| TEST-005 | Gateway compatibility | #546 accepts normal internal/local http(s) URLs | `http://auth:8001`, `http://localhost:8001`, and https internal DNS remain valid | `services/gateway/internal/platform/authclient` tests passed | pass |
| TEST-006 | GitHub alert status | Query alert state through GitHub API | Record current state or limitation | Dependabot #39 and CodeQL #17/#18 returned `fixed` | pass |

## 5. Commands And Results

| Time | ID | Command or operation | Result | Evidence / Notes |
| --- | --- | --- | --- | --- |
| 2026-07-03 14:22 +0800 | TEST-001 | `cd services/file && go list -m golang.org/x/net; go test ./...; go build ./cmd/server` | fail | Initial run resolved `golang.org/x/net v0.55.0` but failed downloading `golang.org/x/net` and `golang.org/x/sys` from `proxy.golang.org` due connection timeout |
| 2026-07-03 14:26 +0800 | TEST-001 | `cd services/file && $env:GOPROXY='https://goproxy.cn,direct'; $env:GOSUMDB='sum.golang.google.cn'; go list -m golang.org/x/net; go test ./...; go build ./cmd/server` | pass | `golang.org/x/net v0.55.0`; all File packages passed and server build returned exit code 0 |
| 2026-07-03 14:22 +0800 | TEST-002 | `cd services/qa && go test ./...` | fail | One transient failure in `internal/platform/mcpclient`: `TestStdioClientLifecycleAndToolCall` context deadline exceeded |
| 2026-07-03 14:26 +0800 | TEST-002 | `cd services/qa && go test ./internal/platform/mcpclient -run '^TestStdioClientLifecycleAndToolCall$' -count=1 -v` | pass | Focused MCP stdio test passed in 7.93s |
| 2026-07-03 14:26 +0800 | TEST-002 | `cd services/qa && go build ./cmd/server; go build ./cmd/agent` | pass | Both builds returned exit code 0 |
| 2026-07-03 14:29 +0800 | TEST-002 | `cd services/qa && go test -p 1 ./...` | pass | Low-concurrency full QA suite passed |
| 2026-07-03 14:32 +0800 | TEST-002 | `cd services/qa && go test ./...` | pass | Final default full QA suite passed |
| 2026-07-03 14:22 +0800 | TEST-004 / TEST-005 | `cd services/gateway && go test ./internal/platform/authclient ./internal/config ./internal/http; go build ./cmd/server` | pass | Auth client, config, and HTTP packages passed; server build returned exit code 0 |
| 2026-07-03 14:22 +0800 | TEST-003 | `bun run --cwd apps/web check` | pass | Typecheck, test typecheck, lint, and format check passed; lint reported 2 warnings in `components/ui/select.tsx`, no errors |
| 2026-07-03 14:22 +0800 | TEST-003 | `bun run --cwd apps/web test:unit` | fail | First run had 3 timeout failures under concurrent load |
| 2026-07-03 14:26 +0800 | TEST-003 | `bun run --cwd apps/web test:unit -- src/pages/admin/qa-retrieval-test.a11y.test.tsx src/pages/knowledge/documents/page.test.tsx src/pages/reports/generate/page.test.tsx` | pass | Targeted rerun for the first failed files passed: 15 tests |
| 2026-07-03 14:29 +0800 | TEST-003 | `bun run --cwd apps/web test:unit -- --maxWorkers=1` | pass | Low-worker full frontend suite passed: 36 files, 150 tests |
| 2026-07-03 14:32 +0800 | TEST-003 | `bun run --cwd apps/web test:unit` | pass | Final default full frontend suite passed: 36 files, 150 tests |
| 2026-07-03 14:22 +0800 | TEST-006 | `gh api repos/Sakayori-Iroha-168/Software_Teamwork/dependabot/alerts --paginate ...` | pass | Alert 39 returned `state=fixed`, dependency `golang.org/x/net`, manifest `services/file/go.mod`, GHSA `GHSA-5cv4-jp36-h3mw` |
| 2026-07-03 14:22 +0800 | TEST-006 | `gh api repos/Sakayori-Iroha-168/Software_Teamwork/code-scanning/alerts --paginate ...` | pass | Alert 17 returned `state=fixed`, rule `js/insecure-randomness`, path `apps/web/src/pages/qa/chat/page.tsx`; alert 18 returned `state=fixed`, rule `go/allocation-size-overflow`, path `services/qa/internal/repository/postgres.go` |
| 2026-07-03 14:27 +0800 | TEST-ALL | `git diff --check` | pass | No whitespace errors |
| 2026-07-03 14:53 +0800 | TEST-001 | `cd services/file && $env:GOPROXY='https://goproxy.cn,direct'; $env:GOSUMDB='sum.golang.google.cn'; go list -m golang.org/x/net; go test ./...; go build ./cmd/server` | pass | Latest-develop recheck after rebase; `golang.org/x/net v0.55.0`; File tests/build passed |
| 2026-07-03 14:53 +0800 | TEST-002 | `cd services/qa && go test ./...; go build ./cmd/server; go build ./cmd/agent` | pass | Latest-develop recheck after rebase; QA tests and both builds passed |
| 2026-07-03 14:53 +0800 | TEST-004 / TEST-005 | `cd services/gateway && go test ./internal/platform/authclient ./internal/config ./internal/http; go build ./cmd/server` | pass | Latest-develop recheck after rebase; Gateway tests/build passed |
| 2026-07-03 14:53 +0800 | TEST-003 | `bun run --cwd apps/web check` | pass | Latest-develop recheck after rebase; check passed with 2 existing lint warnings in `components/ui/select.tsx` |
| 2026-07-03 14:53 +0800 | TEST-003 | `bun run --cwd apps/web test:unit` | pass | Latest-develop recheck after rebase; 38 files and 162 tests passed |
| 2026-07-03 14:53 +0800 | TEST-006 | `gh api repos/Sakayori-Iroha-168/Software_Teamwork/dependabot/alerts --paginate ...` and `gh api repos/Sakayori-Iroha-168/Software_Teamwork/code-scanning/alerts --paginate ...` | pass | Latest-develop recheck after rebase; alert 39, 18, and 17 still returned `fixed` |
| 2026-07-03 14:53 +0800 | TEST-ALL | `git diff --check` | pass | Latest-develop recheck after rebase; no whitespace errors |

## 6. Defects And Handling

| Issue | Level | Handling | Linked issue / PR | Notes |
| --- | --- | --- | --- | --- |
| Initial File command timed out on `proxy.golang.org` | Environment issue | Retried with documented Go module mirror settings | This PR | The retry follows `deploy/.env.example`; no code change required |
| Initial QA MCP stdio test timed out during full-suite load | Small / transient test stability risk | Re-ran focused test, low-concurrency full suite, and final default full suite; all passed | This PR report | No persistent defect reproduced after load dropped |
| Initial frontend Vitest full suite had timeout/follow-up load failures | Small / transient test stability risk | Re-ran failed files, low-worker full suite, and final default full suite; all passed | This PR report | No persistent defect reproduced after load dropped |
| Issue #549 body initially said `Blocked`; earlier claim comment failed | Process state mismatch | Updated #549 to `Status=Review` and `Risk=Low` after PR creation; dependencies #533 and #546 are closed | #549 | PR labels were rechecked after the update and `blocked` was removed |

## 7. Evidence List

| Evidence Type | Location / Link | Description |
| --- | --- | --- |
| Code change | `services/gateway/internal/platform/authclient/client_test.go` | Added explicit missing-host URL rejection case |
| Report | `docs/testing/reports/2026-07-03/security-url-boundary-regression-test-report.md` | This report |
| Alert status | GitHub API | Alert 39, 18, and 17 returned `fixed` |
| Static source check | `apps/web/src/pages/qa/chat/page.tsx` | `nextId()` uses `crypto.randomUUID()` or `crypto.getRandomValues()` |
| Dependency version | `services/file/go.mod` | `golang.org/x/net v0.55.0` |
| QA metadata tests | `services/qa/internal/service/qa_test.go`, `services/qa/internal/service/resources_test.go`, `services/qa/internal/repository/resources_postgres_test.go` | Safe metadata preserved; object keys/internal URLs/vectors removed; attachment metadata behavior covered |

## 8. Risks And Remaining Gaps

- GitHub alert status was observed through the GitHub API as `fixed`; this report does not perform or force a new CodeQL/Dependabot rescan.
- The issue had a stale `Blocked` status at the start of this task; it has now been updated to `Review` with `Risk=Low`.
- The local machine showed transient test-load sensitivity. Final default commands passed, but the first failures are retained in the execution log for reviewer visibility.

## 9. Final Conclusion

Passed after environment retry and latest-develop rebase. All issue-scoped security alert and URL boundary checks have command-backed evidence on `upstream/develop @ 3e3565a6`, GitHub alert API state is `fixed` for #39/#18/#17, and no sensitive values were included in this report.

## 10. Review Checklist

- [x] Tests were actually run, not only listed.
- [x] Commands, environment, results, and first-run failures are recorded.
- [x] Small/environment issues are separated from security boundary defects.
- [x] Skipped or limited external checks are documented.
- [x] This report is linked-ready for issue #549 and the PR body.

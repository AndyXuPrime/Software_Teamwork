# T-015 security alert and internal URL boundary regression

## Goal

Deliver GitHub issue #549 / T-015 by independently verifying the security
alert fixes from #533 and the Gateway Auth base URL boundary fixes from #546 on
top of the latest `upstream/develop`.

The primary deliverable is an archived test report at
`docs/testing/reports/2026-07-03/security-url-boundary-regression-test-report.md`.

## Confirmed Facts

- `upstream/develop` was fetched on 2026-07-03 and is currently
  `59290d1d`.
- Issue #549 is open, assigned to `AndyXuPrime`, and requests branch
  `Test/test/security-alert-url-boundary-regression`.
- The issue body still says `Blocked`, and an earlier claim comment failed
  because the issue status had not been changed to `Draft` or `Ready`.
- Dependency issue #533 is closed as of 2026-07-03 11:47 +0800.
- Dependency issue #546 is closed as of 2026-07-03 11:58 +0800.
- Scope comes from issue #549, `docs/testing/strategy.md`, the test report
  template, and `.trellis/spec/backend/quality-guidelines.md`.

## Requirements

- Verify Dependabot alert 39 by confirming `services/file` resolves to a
  patched `golang.org/x/net` version and the File service tests/build pass.
- Verify Code Scanning alert 18 by confirming QA citation metadata behavior is
  covered and metadata map capacity is not derived from external input length
  without a safe cap.
- Verify Code Scanning alert 17 by confirming the QA chat frontend uses Web
  Crypto or an equivalent secure random source instead of `Math.random()` for
  message/session IDs.
- Verify #546 by confirming Gateway Auth rejects invalid
  `GATEWAY_AUTH_BASE_URL` forms: non-http(s), credentials, query, fragment, and
  missing host, while normal local/internal http(s) addresses still work.
- Run the issue-required minimum checks where local tooling permits:
  File service tests/build, QA service tests, Gateway auth/config/http tests and
  build, frontend check/unit tests, and `git diff --check`.
- Record any GitHub Security / CodeQL / Dependabot alert state that can be
  observed, and record limitations rather than inventing a rescan result.
- Do not broaden the task into unrelated security fixes or documentation
  changes unless a small directly related defect is found and can be safely
  fixed in this PR.
- Do not include tokens, internal secrets, connection strings, object keys,
  provider raw bodies, or other sensitive values in the report.

## Acceptance Criteria

- [ ] The test report exists at the required path and follows the project test
      report template structure.
- [ ] Each alert/config boundary from issue #549 has an explicit verification
      result, command/evidence, and conclusion.
- [ ] Required service/frontend commands are run and recorded, or skipped with
      a concrete environment/tooling reason and residual risk.
- [ ] The branch is based on the latest fetched `upstream/develop`.
- [ ] Any discovered defect is either fixed in this task or explicitly linked
      to a follow-up owner issue.
- [ ] The final commit uses Conventional Commits and the PR body follows the
      repository issue/PR requirements without mojibake.

## Out of Scope

- Closing GitHub Security alerts directly.
- Reworking the security fixes from #533 or #546 unless regression testing
  exposes a small issue that belongs in this test PR.
- Expanding to the full site authorization matrix.

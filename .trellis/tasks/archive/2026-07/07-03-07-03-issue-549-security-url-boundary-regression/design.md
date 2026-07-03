# Design

## Boundary

This is a regression verification task. The expected repository change is a
test report and, only if needed, narrowly scoped tests or small fixes that prove
the security boundaries requested by issue #549.

## Verification Strategy

- Prefer existing focused tests and service-local commands over broad
  integration tests.
- Use static inspection only as supporting evidence; each alert should have a
  command-backed result when the repository already contains an executable
  check.
- Treat GitHub alert status as external evidence. If CLI/API access does not
  expose alert status or rescans are unavailable, record that limitation instead
  of claiming the alert is closed.
- Keep report evidence sanitized. Record command names, pass/fail summaries,
  module versions, file paths, and line references. Do not record secret values
  or raw provider/downstream bodies.

## Compatibility

No runtime behavior should change unless an issue-required regression test
reveals a direct defect. The report path is additive under
`docs/testing/reports/2026-07-03/`.

## Rollback

If a command or report update is wrong, revert the report/test additions only.
Do not alter unrelated docs or previously merged security fixes.

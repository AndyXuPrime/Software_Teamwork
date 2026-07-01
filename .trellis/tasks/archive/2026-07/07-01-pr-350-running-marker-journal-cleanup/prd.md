# Fix PR 350 running marker and journal cleanup

## Goal

Address the latest PR #350 review findings by making report section running
state persistence failures explicit errors, and by replacing newly added Trellis
journal template placeholders with real implementation and validation notes.

## Requirements

- `ReportGenerationService` must not continue to AI generation when
  `MarkReportSectionGenerationRunning` fails.
- A running-marker persistence failure must record a `section.failed` event,
  preserve normal failure progress semantics, and return a dependency error so
  the worker can mark the job/report failed instead of reporting success.
- Add a regression test that first fails against the current swallowed-error
  behavior, then passes after `markSectionGenerationRunning` returns and
  propagates an error.
- Clean the PR-added Trellis journal sessions so `Main Changes` and `Testing`
  contain concrete content, not journal template placeholders.
- Keep task manifests free of Trellis template placeholder rows.

## Acceptance Criteria

- [x] Targeted regression test covers `MarkReportSectionGenerationRunning`
  failure and verifies no chat request is sent, a dependency error is returned,
  and a `section.failed` event is recorded.
- [x] Journal Session 19-25 no longer contain template placeholders in
  `Main Changes` or `Testing`.
- [x] Placeholder scans are clean for PR-added task manifests and journal
  entries.
- [x] `go test ./internal/service -run ... -count=1` shows RED before the fix
  and GREEN after the fix.
- [x] `go test ./... -count=1`, `go build ./cmd/server`, `go vet ./...`,
  `govulncheck ./...`, and `git diff --check` pass before push.

## Notes

- This is a lightweight review-fix task; PRD-only planning is sufficient.

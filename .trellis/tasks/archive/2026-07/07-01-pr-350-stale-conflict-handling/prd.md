# Fix PR 350 stale conflict handling

## Goal

Close the latest PR #350 review findings by ensuring stale AI section responses
do not make the worker mark a generation job/report failed, and by removing
Trellis template manifest placeholders from archived/current task context.

## Requirements

- When generated-section success persistence detects a stale response via
  `CodeConflict` (current section changed, status no longer matches, or another
  job owns it), Document generation must treat that section as skipped rather
  than returning an execution error to the worker.
- Stale-section skips must preserve the current section content, tables,
  version, source, manual edit state, `last_job_id`, and generation status; they
  must create no stale AI `report_section_versions` row and must not call
  section failed compensation.
- The worker must not mark job/report failed for this stale-section path. The
  existing non-error generation result path should be used so normal final
  status handling applies.
- Remove Trellis template placeholder rows from Trellis `implement.jsonl` /
  `check.jsonl` manifests in current and archived tasks included in this PR so
  future Trellis context and review scans do not ingest template text.

## Acceptance Criteria

- [x] Regression tests demonstrate stale successful section writes return no
  error, update progress/skip event, preserve current section state, and create
  no AI section version.
- [x] The existing worker path receives a non-error generation result for stale
  section skips, so it does not call `markFailed`.
- [x] Repository scan for template manifest placeholders under `.trellis/tasks` returns no matches
  for task manifests included in the PR.
- [x] `cd services/document && go test ./internal/service -run ... -count=1`
  shows RED before implementation and GREEN after implementation for stale
  conflict handling.
- [x] `cd services/document && go test ./... -count=1`, `go build ./cmd/server`,
  `go vet ./...`, `govulncheck ./...`, and `git diff --check` pass before push.

## Constraints

- Do not change public API status enums in this review fix.
- Do not include local machine paths in PR body or public comments.

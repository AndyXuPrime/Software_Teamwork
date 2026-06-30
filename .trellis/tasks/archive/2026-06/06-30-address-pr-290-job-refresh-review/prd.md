# Address PR 290 Job Completion Refresh Review

## Background

PR #290 received a hidden Codex PR Review finding in the failed review Action
log. The review was not posted as a PR comment because GitHub Actions could not
create an issue comment from the fork context.

The finding points out that report jobs are long-running resources. The current
frontend invalidates outlines and sections immediately after creating a job, but
does not refresh report data again when the polled job later reaches a terminal
state.

## Requirements

- When `useReportJobQuery` observes a report job entering a terminal status,
  refresh the report data produced by that long-running job.
- Refresh at least:
  - `reportKeys.outlines(reportId)`
  - `reportKeys.sections(reportId)`
  - `reportKeys.detail(reportId)`
  - `reportKeys.records()`
  - `reportKeys.events(reportId)`
- Avoid repeatedly invalidating the same terminal job on every render/refetch.
- Keep the fix scoped to PR #290 frontend report query behavior.
- Do not modify backend or Gateway OpenAPI.

## Acceptance Criteria

- After an outline/content job reaches `succeeded`, `partial_succeeded`,
  `failed`, or `canceled`, related report queries are invalidated so the UI can
  fetch generated outlines/sections and updated status without a manual refresh.
- A regression test covers terminal job polling triggering report data refresh.
- Existing report API contract fixes remain intact.

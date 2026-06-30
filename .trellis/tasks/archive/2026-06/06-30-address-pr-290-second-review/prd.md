# Address PR 290 Second Review

## Background

PR #290 received a second Codex PR Review after the first fix. The new review
points to concrete API contract mismatches and Trellis artifact hygiene issues.

## Requirements

- Use the existing void transport helper for DELETE endpoints that return
  `204 No Content`.
- Correct report job retry attempt typing to match Gateway OpenAPI:
  `POST /api/v1/report-jobs/{jobId}/attempts` returns `ReportJobAttempt`.
- After creating a retry attempt, refresh the original job and report events so
  the UI can observe the retried task state.
- Clean Trellis archived task jsonl placeholders and journal template text that
  were already added to PR #290.
- Keep the change scoped to PR #290 frontend/Trellis follow-up; do not modify
  backend or Gateway OpenAPI.

## Acceptance Criteria

- Successful DELETE 204 responses do not produce JSON parse errors.
- Retry attempt code no longer treats the attempt payload as a `ReportJob`.
- Retry attempt tests or existing tests cover user-visible behavior.
- Archived task context files contain real context instead of seeded placeholder
  records.
- Journal entries added by this PR contain concrete main changes, commit
  messages, and verification notes instead of template filler text.

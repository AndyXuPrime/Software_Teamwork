# Fix report generation failure compensation

## Goal

Resolve the PR #350 review finding that report generation failure compensation can overwrite concurrent section edits by writing an old full-row snapshot back to the database.

## Requirements

- When generated-section persistence fails, mark generation failure without overwriting user-editable section fields such as `content`, `tables`, `version`, `content_source`, or `manual_edited`.
- Keep the failure compensation scoped to status/error timestamp fields, unless the current section still matches an explicitly safe update condition.
- Preserve the existing successful-generation behavior and public API contracts.
- Add focused regression coverage for the concurrent-edit overwrite scenario.

## Acceptance Criteria

- [ ] Failure compensation after a persistence error does not revert a manual edit made while generation was running.
- [ ] Existing report generation service tests still pass.
- [ ] Document service build, vet, and vulnerability checks remain clean for reachable code.

## Notes

- Source review comment: PR #350 `github-actions` finding on `services/document/internal/service/report_generation_service.go`.
- This is a lightweight review-fix task; PRD-only planning is sufficient.

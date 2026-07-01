# Fix section version review contract gaps

## Goal

Resolve the latest PR #350 review findings for report section version creation so the write path rejects deleted reports and the public Gateway contract exposes the conflict response.

## Requirements

- `CreateSectionVersion` must reject soft-deleted reports before creating a `ReportSectionVersion` or mutating the current `ReportSection`.
- Add a regression test proving a deleted report owner cannot create a section version.
- Gateway public OpenAPI for `createReportSectionVersion` must list `409` conflict, matching the Document service behavior when section generation is running or the report is deleted.
- Keep the change scoped to the section-version write path and public contract update.

## Acceptance Criteria

- [x] Creating a section version on a deleted report returns `conflict` and creates no history row.
- [x] Gateway `docs/services/gateway/api/public.openapi.yaml` declares `409` for `createReportSectionVersion`.
- [x] Document service tests pass, including the new regression.
- [x] OpenAPI/frontend generation remains clean if the Gateway contract change affects generated clients.

## Notes

- Source review comment: PR #350 `github-actions` review on head `d6c7013162f4`.
- This is a lightweight review-fix task; PRD-only planning is sufficient.

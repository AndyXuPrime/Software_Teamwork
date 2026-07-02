# Implementation Plan

## Order

1. Add/adjust backend service tests first:
   - `coal_inventory_audit` outline generation is accepted and calls AI Gateway.
   - prompt messages contain coal-inventory wording for coal inventory jobs.
2. Implement the minimal Document service change:
   - replace single hard-coded supported type check with a two-entry metadata map.
   - build outline/section system prompts from report metadata.
3. Update seed SQL:
   - local demo labels in `deploy/seeds/001-local-demo-seed.sql`.
   - realistic placeholder template structures in `services/document/migrations/0003_seed_initial_report_defaults.sql` without secrets or internal refs.
4. Add frontend tests first:
   - report generation page does not render the stale warning.
   - report settings publish flow sends `provider: "ai-gateway"` and selected `profileId` to `/api/v1/report-settings`.
5. Implement frontend:
   - add report settings generated-type aliases.
   - add `getReportSettings` / `updateReportSettings` feature API wrappers.
   - add TanStack Query hooks and invalidation.
   - add the document generation model config panel to `pages/reports/generate/page.tsx`.
6. Align status docs that still state only `summer_peak_inspection` can generate.
7. Run quality checks and fix failures.

## Validation Commands

- `cd services/document && go test ./...`
- `cd services/document && go build ./cmd/server`
- `python scripts/verify_local_seed_contract.py`
- `bun run --cwd apps/web test:unit -- src/pages/reports/generate/page.test.tsx src/features/reports/report-generation.api.test.ts src/features/reports/report-generation.queries.test.tsx`
- `bun run --cwd apps/web check`
- `bun run --cwd apps/web build`
- `git diff --check`

## Review Gates

- No provider API keys/base URLs added to Document settings or frontend report settings payloads.
- No frontend calls to AI Gateway internal routes.
- Existing QA configuration behavior remains untouched.
- Seed changes remain deterministic, idempotent, and local/demo safe.

## Rollback Points

- Backend generation support can be reverted by restoring the single supported type check and prompt constants.
- Frontend configuration UI can be reverted independently because it uses existing Gateway endpoints.
- Seed copy/template changes are isolated to SQL seed/migration files and can be reverted without touching runtime code.

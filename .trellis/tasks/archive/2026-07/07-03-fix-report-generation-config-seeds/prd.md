# Fix report generation configuration and seeded demo data

## Goal

Make report generation demo data bilingual/Chinese-facing, support both seeded report types, remove stale capability warning, and add report generation model configuration wired to AI Gateway profiles.

## Background

- The report generation page currently shows a stale "capability boundary" notice even though the backend AI outline/content workflow is implemented for at least one report type.
- Local demo seed data still exposes English report type and material names in the UI, while the product surface is Chinese-facing.
- Document service seeds two fixed report types: `summer_peak_inspection` and `coal_inventory_audit`, but AI generation currently rejects `coal_inventory_audit`.
- QA/LLM settings can select an enabled AI Gateway chat profile and publish a runtime config. Report generation has no equivalent UI, so Document falls back to `DOCUMENT_AI_GATEWAY_PROFILE_ID` when `report_settings.llm.profileId` is empty.
- Provider base URLs and API keys remain owned by AI Gateway model profiles. Document settings may store only the profile reference, model display value, and timeout metadata.

## Requirements

- R1: The local demo seed and Document default template seed must present report type, material, report, and template names/descriptions in Chinese-facing copy.
- R2: Both seeded report types, `summer_peak_inspection` and `coal_inventory_audit`, must be accepted by the Document AI outline/content generation path.
- R3: The seeded placeholder templates for both report types must contain more realistic outline structures while still being clearly placeholder/local defaults and free of secrets or internal file references.
- R4: The report generation page must remove the stale top warning that says real AI generation is not ready.
- R5: The report generation UI must add a document generation model configuration module similar to QA/LLM config: list enabled chat model profiles, select one, show the effective profile/model, and publish it through Gateway `PATCH /api/v1/report-settings`.
- R6: Publishing document generation model settings must update Document `report_settings.llm.profileId` through the existing Report Settings API. The frontend must not attempt to mutate process environment variables or send provider API keys/base URLs.
- R7: Existing report generation workflows and QA model profile configuration must remain compatible.
- R8: Documentation that currently claims only `summer_peak_inspection` is supported must be aligned with the new two-report-type behavior.

## Acceptance Criteria

- [ ] `coal_inventory_audit` outline/content generation is covered by a failing-then-passing service test and no longer returns an unsupported report type validation error.
- [ ] Prompt construction uses report-type-specific or generic report wording instead of hard-coding every generation as an "迎峰度夏检查报告".
- [ ] `deploy/seeds/001-local-demo-seed.sql` uses Chinese-facing demo labels for the two report types, local demo material, and local demo report.
- [ ] `services/document/migrations/0003_seed_initial_report_defaults.sql` seeds realistic placeholder outline structures for both default templates without storing API keys, provider URLs, `file_ref`, object keys, or raw prompts.
- [ ] The report generation page no longer renders the stale "能力边界" warning.
- [ ] The report generation page renders a document generation model config module backed by enabled chat model profiles and report settings.
- [ ] Selecting a model profile and publishing sends only Document report settings data, including `provider: "ai-gateway"` and `profileId`, through Gateway `/api/v1/report-settings`.
- [ ] Frontend tests cover the report settings read/update flow and absence of the stale warning.
- [ ] Service-local, frontend, seed, and whitespace checks run or are explicitly reported if blocked.

## Notes

- Frontend code must continue to call Gateway `/api/v1/**`, never AI Gateway internal `/internal/v1/**`.
- Runtime provider credentials stay in AI Gateway model profiles and must not appear in Document settings, frontend state, logs, or seed data.

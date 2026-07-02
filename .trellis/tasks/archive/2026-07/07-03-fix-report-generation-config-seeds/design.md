# Design

## Scope

This task spans Document service generation logic, Document/local seed data, frontend report generation UI, and supporting status documentation. It does not add new public Gateway routes because `/api/v1/report-settings` and AI Gateway model profile admin routes already exist in the public Gateway contract.

## Backend Generation

Document generation keeps the existing asynchronous job flow:

`report job -> ReportGenerationService -> ReportSettings -> AI Gateway chat client -> outline/section persistence`

The only behavior change is the supported report type policy and prompt wording. A small report-type metadata map will declare the fixed report codes that are allowed for AI generation:

- `summer_peak_inspection` -> `迎峰度夏检查报告`
- `coal_inventory_audit` -> `煤库存审计报告`

Outline and section prompts will use the selected label when building model messages so coal inventory jobs are not described as summer inspection jobs. Unknown report types still fail with `validation_error`.

## Report Settings Data Flow

The report generation configuration UI follows the same public-boundary pattern as QA/LLM config, but writes Document-owned settings:

`ReportGeneratePage -> Gateway /api/v1/admin/model-profiles?purpose=chat&enabled=true`

`ReportGeneratePage -> Gateway GET/PATCH /api/v1/report-settings -> Document report_settings.llm_json`

The UI selects an enabled chat profile and publishes:

```json
{
  "llm": {
    "provider": "ai-gateway",
    "profileId": "<selected-profile-id>"
  }
}
```

Document service validates and enriches the profile reference through AI Gateway, preserving the existing backend semantics. The browser never sends provider API keys, provider base URLs, or model-provider raw details. The user's wording about updating `DOCUMENT_AI_GATEWAY_PROFILE_ID` is implemented through persistent `report_settings.llm.profileId`; runtime environment variables are process startup fallback values and are not mutable from the UI.

## Seed Data

Local demo seed SQL will update Chinese-facing rows for the local demo database. Service-owned migration seed SQL will keep deterministic report type/template IDs and idempotent insert/merge behavior. Placeholder template structures will be more realistic operational outlines for both report types while still avoiding `file_ref`, provider credentials, object keys, or fake production claims.

## Frontend UI

The report generation page remains the first usable workflow screen. A compact model configuration panel will be added near the generation controls, using TanStack Query hooks under `features/reports` for report settings and the existing `useModelProfiles('chat', true)` hook for selectable profiles. It will include loading/error/success states and explicit publish feedback.

The stale top "能力边界" notice will be removed. Other legitimate action feedback, such as job creation success/failure, remains.

## Compatibility

- Existing QA/LLM settings stay unchanged.
- Existing Document `PATCH /report-settings` clear-vs-omit semantics stay unchanged.
- Existing reports that rely on fallback `DOCUMENT_AI_GATEWAY_PROFILE_ID` keep working when no persisted profile is configured.
- Existing local databases may need either reseeding/migration re-run for display seed changes or a one-time manual update if rows were already customized.

## Risks

- Frontend may need careful typing because generated OpenAPI schemas already exist but feature wrappers have not exposed report settings yet.
- Seed migrations must preserve user modifications. Existing rows inserted by earlier migrations use `ON CONFLICT DO NOTHING`; realistic template improvements should not overwrite customized databases unless applied through local demo seed or a later explicit migration policy.
- Full real-provider report generation smoke depends on the user's local AI Gateway profile and provider availability, so automated checks should rely on fake-backed service tests.

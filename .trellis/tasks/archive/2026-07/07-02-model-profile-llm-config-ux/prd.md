# Improve model profile and LLM config UX

## Goal

Make the model profile -> QA/LLM activation path discoverable and hard to misconfigure. After an admin creates a chat model profile, they should be able to see its Profile ID, understand that QA/LLM must reference it, select it from the LLM settings page, test it, and publish the active LLM config without manually copying hidden or ambiguous values.

## Background

- User-confirmed issue: model chat succeeded only after the model profile Base URL used the provider API root including `/v1`, and QA/LLM referenced the real AI Gateway profile id.
- Previous confusing path: the model profile page created profiles but did not expose the created `profileId`; QA/LLM config expected users to manually enter `profileId` and `modelName`.
- Current frontend evidence:
  - `apps/web/src/pages/admin/model-profiles.tsx` lists model profile rows without a Profile ID column or copy affordance.
  - `apps/web/src/pages/admin/qa-settings.tsx` exposes manual `profileId` and `modelName` inputs, mixes English labels, and shows verbose version metadata including UUID/version number.
  - `useModelProfiles(purpose, enabled)` already supports fetching enabled chat profiles through Gateway.
- Contract evidence:
  - `.trellis/spec/frontend/type-safety.md` requires QA/LLM config to submit only `provider: "ai-gateway"`, `profileId`, `modelName`, and optional generation/timeout fields. It must not include provider API key, credential placeholders, Base URL, or raw provider details.
  - Model profile CRUD must continue using Gateway `/api/v1/admin/model-profiles`.

## Requirements

1. Model management rows must show each profile's Profile ID.
2. The displayed Profile ID must be copyable from the row without opening edit mode.
3. Model management must guide admins that chat profiles need to be selected and published in QA/LLM settings before taking effect.
4. QA/LLM settings must provide a primary "select chat model" flow using enabled `purpose=chat` model profiles.
5. Selecting a chat model must populate both `profileId` and `modelName`; users should not need to manually discover or type Profile ID.
6. QA/LLM visible copy must be Chinese-first. Technical labels may keep canonical names where helpful, for example `Profile ID`.
7. Version/status metadata in QA and LLM panels must be simplified to creation time and whether the config is effective.
8. LLM connection test and publish requests must continue to use the Gateway schema and must not send model profile secrets, provider Base URL, or provider raw configuration.
9. The UI should remain compact, operational, and visually consistent with existing admin pages.

## Acceptance Criteria

- [ ] Model management table renders a visible `Profile ID` field for each model profile row.
- [ ] Clicking the copy control for a row copies that row's Profile ID and provides visible feedback.
- [ ] Model management page includes concise guidance telling admins to configure QA/LLM after creating or updating a chat profile.
- [ ] QA/LLM settings fetch enabled chat model profiles and renders them as a selectable list.
- [ ] Choosing a model profile updates the LLM form's `Profile ID` and model name.
- [ ] LLM connection test sends only `provider`, `profileId`, `modelName`, and optional `timeoutSeconds`.
- [ ] Publishing LLM config sends only generated-contract fields and does not include API key or Base URL.
- [ ] QA/LLM page labels in the LLM panel are Chinese, with status metadata reduced to active state and creation time.
- [ ] Existing model profile edit behavior still omits blank `apiKey` on update.
- [ ] Relevant unit/component tests fail before implementation and pass after implementation.

## Out Of Scope

- Backend API changes.
- Auto-creating or auto-publishing QA/LLM config immediately after creating a model profile.
- Changing AI Gateway health readiness semantics for missing embedding/rerank profiles.
- Persisting provider API Key, Base URL, or raw provider errors in QA/LLM configuration.

## Open Questions

None blocking. User approved the proposed UX direction in chat and requested starting the workflow.

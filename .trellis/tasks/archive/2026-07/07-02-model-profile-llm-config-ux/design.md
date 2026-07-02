# Design

## Overview

This is a frontend-only UX improvement across two admin pages:

- `apps/web/src/pages/admin/model-profiles.tsx`
- `apps/web/src/pages/admin/qa-settings.tsx`

The backend contract remains unchanged. Model profile management continues to own provider URL and credentials. QA/LLM settings only reference AI Gateway profiles by `profileId` and `modelName`.

## Model Management Page

Add a Profile ID column or compact row field next to the model name. The value should be rendered in a monospace, wrapping-safe chip so long IDs do not break the table. Add an icon button using `lucide-react` for copy-to-clipboard. Copy feedback can reuse the existing notification state with a short Chinese message.

Add a compact guidance notice near the page header:

> 聊天模型创建或更新后，需要到 QA / LLM 配置中选择该 Profile 并发布后才会生效。

On successful chat profile create/update, keep the success toast concise but include the same next-step hint.

## QA/LLM Settings Page

Fetch enabled chat profiles with `useModelProfiles('chat', true)`.

Replace the primary manual `profileId` + `modelName` workflow with:

1. A `聊天模型` select control listing enabled chat profiles.
2. Read-only display of selected `Profile ID`, with copy button.
3. Read-only or auto-populated model name display/input based on selected profile.
4. Existing generation parameters: timeout, temperature, max token count, activate flag.

The form state can keep `profileId` and `modelName` because the Gateway payload requires those fields. The selected profile id is derived from the select value. If the current active LLM config references a profile not present in enabled chat profiles, preserve the current form values and show a fallback option or display text so the existing config is not erased.

If no enabled chat profiles exist, show a compact warning/empty state telling the user to create and enable a chat model in model management. Do not disable viewing current config metadata, but disable test/publish until a valid `profileId` and `modelName` are present.

## Status Metadata

Replace verbose version metadata with a small status strip:

- `生效中` or `未生效`
- creation time formatted with `toLocaleString()`, or `-`

Do not show internal config UUID or version number in normal view.

## Data Flow

- Model profile list:
  - GET `/api/v1/admin/model-profiles`
  - Render `ModelProfile.id` as Profile ID.
- QA/LLM settings:
  - GET `/api/v1/llm-config-versions/current`
  - GET `/api/v1/admin/model-profiles?purpose=chat&enabled=true`
  - Select model -> set `llmForm.profileId = profile.id` and `llmForm.modelName = profile.model`.
  - Test -> POST `/api/v1/llm-connection-tests` with `provider`, `profileId`, `modelName`, optional `timeoutSeconds`.
  - Publish -> POST `/api/v1/llm-config-versions` with generated-schema fields only.

## Compatibility

Existing current LLM config values must still render even if their referenced profile is disabled, deleted, or not in the fetched list. This avoids accidentally clearing a production config.

Clipboard API may not exist in every test/browser environment. The copy helper should guard `navigator.clipboard?.writeText` and show an error notification if unavailable.

## Tests

Use React Testing Library component tests:

- Extend `model-profiles.test.tsx` to assert Profile ID visibility and copy feedback.
- Add `qa-settings.test.tsx` to assert enabled chat profile selection, Chinese labels, simplified metadata, and sanitized test/publish payloads.

Targeted tests are enough before full frontend checks because this task is page-level behavior, not API generation or backend integration.

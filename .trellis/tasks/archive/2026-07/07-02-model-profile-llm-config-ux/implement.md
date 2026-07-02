# Implementation Plan

## Preconditions

- Branch: `Liutest/feat/model-profile-llm-config-ux`
- Base branch: `upstream/develop`
- Task path: `.trellis/tasks/07-02-model-profile-llm-config-ux`

## Checklist

1. Start Trellis task and refresh applicable frontend specs.
2. Write failing tests first:
   - model profile table displays/copies Profile ID.
   - QA/LLM page selects enabled chat profile and sends sanitized test/publish payloads.
   - QA/LLM version metadata excludes UUID/version and shows active state/time.
3. Implement model profile row Profile ID display, copy action, and next-step hint.
4. Implement QA/LLM chat-profile select, read-only Profile ID display, Chinese labels, and simplified metadata.
5. Keep payload builders and connection test sanitized per generated schema.
6. Run targeted tests:
   - `bun run --cwd apps/web test:unit -- src/pages/admin/model-profiles.test.tsx src/pages/admin/qa-settings.test.tsx`
7. Run required frontend checks:
   - `bun run --cwd apps/web check`
   - `bun run --cwd apps/web build`
   - `git diff --check`
8. Record any key findings into this task before final report.

## Risk Points

- `navigator.clipboard` may be unavailable in Vitest/jsdom and should be stubbed or guarded.
- Existing current LLM config may refer to a disabled/deleted profile; do not erase the form on chat-profile list load.
- Long Profile IDs must not cause table layout overflow.
- Do not include `apiKey`, `baseUrl`, or provider internals in QA/LLM requests.

## Rollback

Revert changes in:

- `apps/web/src/pages/admin/model-profiles.tsx`
- `apps/web/src/pages/admin/model-profiles.test.tsx`
- `apps/web/src/pages/admin/qa-settings.tsx`
- `apps/web/src/pages/admin/qa-settings.test.tsx`

## Progress Notes

- Red tests were added first and failed for the expected missing UX:
  - `model-profiles.test.tsx` could not find `Profile ID`.
  - `qa-settings.test.tsx` could not find the `聊天模型` selector and still saw verbose version metadata.
- Implemented frontend-only changes:
  - Model profile rows render a copyable Profile ID chip and next-step QA/LLM guidance.
  - QA/LLM settings fetch enabled chat profiles with `useModelProfiles('chat', true)`.
  - LLM settings use a chat-model selector, read-only Profile ID display, Chinese labels, simplified status metadata, and sanitized payloads.
- Targeted verification passed:
  - `bun run --cwd apps/web test:unit -- src/pages/admin/model-profiles.test.tsx src/pages/admin/qa-settings.test.tsx`
- Full verification passed:
  - `bun run --cwd apps/web check`
  - `bun run --cwd apps/web build`
  - `bun run --cwd apps/web test:unit`
  - `git diff --check`
- Build note: Vite emitted the existing large chunk warning for the bundled app, but the build exited successfully.
- UI polish after review:
  - Removed the special monospace/small-font styling from Profile ID displays so they match surrounding admin text.
  - Added rounded, theme-colored checkbox styling to the touched admin forms.
  - Re-ran `bun run --cwd apps/web test:unit -- src/pages/admin/model-profiles.test.tsx src/pages/admin/qa-settings.test.tsx`, `bun run --cwd apps/web check`, `bun run --cwd apps/web build`, and `git diff --check`; all exited successfully.

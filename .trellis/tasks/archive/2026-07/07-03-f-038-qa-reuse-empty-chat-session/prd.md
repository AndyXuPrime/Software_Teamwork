# F-038 QA chat reuse empty session

## Goal

Fix the QA chat "new conversation" interaction so repeated clicks while the
current conversation is still an empty new session reuse the selected session
instead of creating duplicate empty sessions in the sidebar.

## Requirements

- Scope is frontend only, centered on
  `apps/web/src/pages/qa/chat/page.tsx` and focused tests.
- When the active session is reusable, the "new conversation" action must not
  call the create-session mutation, must not append another sidebar session, and
  must keep the current session selected.
- A reusable empty new session must be the current active session, must exist in
  the session list, must still represent the default new conversation, and must
  have no actual chat content.
- The empty-session predicate must account for message count, local messages,
  visible or in-progress attachments, submitting state, and the current active
  session's streaming state. A stream still running in a different session must
  not make the selected empty session create duplicate empty conversations.
- If there is no active session, the active session is missing from the list, or
  the active session has actual content, the existing create-session behavior
  must continue to create and select a new session.
- Existing delete, rename, send-message, attachment upload, streaming, and error
  handling behavior must not regress.
- Do not change QA backend session creation contracts.
- Do not clean up historical duplicate empty sessions; only prevent new
  duplicates from repeated clicks.

## Acceptance Criteria

- [x] In an empty new QA session, clicking "new conversation" repeatedly keeps
      only the current empty session in the sidebar.
- [x] In an empty new QA session, repeated "new conversation" clicks do not call
      the create-session API/mutation again.
- [x] In a session with messages, clicking "new conversation" still creates and
      selects a new empty session.
- [x] With no active session, clicking "new conversation" still creates and
      selects a new session.
- [x] A session with attachments or active submit/streaming state is not treated
      as reusable empty content.
- [x] If a different session is still streaming, repeated "new conversation"
      clicks while the selected session is empty still reuse that selected empty
      session instead of appending duplicate empty items.
- [x] Focused frontend tests cover empty-session reuse, non-empty creation, and
      no-active creation.
- [x] `bun run --cwd apps/web check` passes.
- [x] `bun run --cwd apps/web build` passes.
- [x] `git diff --check` passes.

## Notes

- Baseline before this change:
  `bun run --cwd apps/web test:unit -- src/pages/qa/chat/page.test.ts src/components/chat/chat-input.test.tsx src/components/chat/chat-messages.test.tsx src/features/qa/capability.test.ts`
  passed, `bun run --cwd apps/web check` passed, `bun run --cwd apps/web build`
  passed, and `git diff --check` passed.
- Full `bun run --cwd apps/web test:unit` currently fails on unrelated
  `src/pages/password/change-required.test.tsx` logout tests because
  `chat-store` Zustand persist attempts to write unavailable `localStorage` in
  this Bun/Vitest environment.
- Validation after the change:
  `bun run --cwd apps/web test:unit -- src/pages/qa/chat/page.test.ts src/pages/qa/chat/page.create-session.test.tsx src/components/chat/chat-input.test.tsx src/components/chat/chat-messages.test.tsx src/features/qa/capability.test.ts`
  passed, `bun run --cwd apps/web check` passed, `bun run --cwd apps/web build`
  passed, and `git diff --check` passed.

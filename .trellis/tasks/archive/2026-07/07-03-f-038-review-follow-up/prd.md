# F-038 review follow-up

## Goal

Address review feedback on PR #582 so QA chat empty-session reuse remains
session-scoped during cross-session attachment upload, and archived Trellis
artifacts do not retain placeholder context rows.

## Requirements

- Fix the QA chat empty-session reuse guard so page-level attachment upload
  progress from another session does not block reusing the currently selected
  empty session.
- Expose or derive the session that owns the current attachment upload state,
  including the `uploading` phase before an attachment object exists.
- Keep the original rule that an empty session with its own active upload or
  polling attachment is not reusable.
- Add regression coverage for this review path: session A uploading or polling,
  current selected session B empty, repeated "new conversation" clicks must not
  create duplicate empty sessions.
- Clean archived task placeholder `_example` rows from the previous F-038
  Trellis task `implement.jsonl` and `check.jsonl`.
- Do not change backend contracts or session creation APIs.

## Acceptance Criteria

- [x] Cross-session attachment upload/polling state does not cause duplicate
      empty QA sessions.
- [x] Current-session attachment upload/polling state still prevents treating
      that current session as reusable empty content.
- [x] Regression tests cover cross-session upload/polling behavior.
- [x] Archived F-038 `implement.jsonl` and `check.jsonl` no longer contain
      `_example` placeholder rows.
- [x] `bun run --cwd apps/web test:unit -- src/pages/qa/chat/page.test.ts src/pages/qa/chat/page.create-session.test.tsx` passes.
- [x] `bun run --cwd apps/web test:unit` passes.
- [x] `bun run --cwd apps/web check` passes.
- [x] `bun run --cwd apps/web build` passes.
- [x] `git diff --check` passes.

## Notes

- Review source:
  - P1: `apps/web/src/pages/qa/chat/page.tsx:82` used page-level
    `uploadState.phase` directly, but `useAttachmentUpload` state is not
    derived from the active session.
  - P3: archived task context files retained `_example` placeholder rows.

## Validation

- `bun run --cwd apps/web test:unit -- src/pages/qa/chat/page.test.ts src/pages/qa/chat/page.create-session.test.tsx`
- `bun run --cwd apps/web test:unit`
- `bun run --cwd apps/web check`
- `bun run --cwd apps/web build`
- `git diff --check`

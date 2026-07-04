# Implementation Plan

## Checklist

1. Read frontend and API/SSE specs before editing.
2. Inspect current `streamChat` error dispatch and existing `chat.test.ts`
   coverage.
3. Update `streamChat` to dispatch a normalized fatal payload when
   `isFatalErrorEvent(...)` is true.
4. Add focused API-level regression tests for missing `fatal`, and keep
   explicit `fatal: false` covered.
5. Run focused and full frontend checks.
6. Decide whether `.trellis/spec/` needs a narrow update for SSE error
   normalization.
7. Commit, archive the task, record journal, push branch, and open PR to
   upstream `develop`.

## Files Expected To Change

- `apps/web/src/api/chat.ts`
- `apps/web/src/api/chat.test.ts`
- `.trellis/tasks/07-04-07-04-f-049-qa-sse-default-fatal-error/*`

## Validation Commands

- `bun run --cwd apps/web test:unit -- src/api/chat.test.ts`
- `bun run --cwd apps/web test:unit`
- `bun run --cwd apps/web check`
- `bun run --cwd apps/web build`
- `git diff --check`

## Rollback Point

If the API-level change breaks existing stream sequencing or non-fatal replay
behavior, revert the normalization patch and reassess whether the page/hook
consumer contract needs a different typed representation.

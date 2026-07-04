# F-049 QA SSE default fatal error normalization

## Goal

Fix the frontend QA SSE error normalization bug tracked by GitHub issue #642.
When Gateway/QA sends an `event: error` SSE payload without an explicit
`fatal` field, `streamChat` already treats it as fatal through
`payload.fatal !== false`, but it currently dispatches the original payload to
page and hook consumers. Consumers then see `fatal: undefined`, take the
non-fatal branch, and leave streaming UI stuck in the generating state.

## Requirements

- Public issue: `#642`.
- Scope is limited to `apps/web` frontend SSE handling and tests.
- Normalize the fatal decision at the API boundary in
  `apps/web/src/api/chat.ts`.
- Preserve explicit `fatal: false` as non-fatal and allow subsequent SSE events
  to continue dispatching.
- Preserve existing transport/network, malformed SSE, stream-ended-before-
  completion, stale sequence, and fatal-after-completion behavior.
- Keep QA page and `useStreamChat` consumers simple; they should not each
  reimplement missing-`fatal` semantics.
- Add regression coverage for `event: error` payloads that omit `fatal`.
- Run the required frontend checks before commit.

## Acceptance Criteria

- [ ] `streamChat` dispatches missing-`fatal` SSE `error` payloads to
  `onError` with `fatal: true`.
- [ ] `streamChat` continues to dispatch explicit `fatal: false` SSE `error`
  payloads as non-fatal.
- [ ] Fatal default error events stop later event dispatch.
- [ ] QA chat page receives a fatal error shape for missing-`fatal` error
  payloads, so its existing fatal branch can clear `streaming` and
  `activeStream`.
- [ ] `useStreamChat` receives a fatal error shape for missing-`fatal` error
  payloads, so its existing fatal branch can clear hook-local `isStreaming`.
- [ ] `bun run --cwd apps/web test:unit -- src/api/chat.test.ts` passes.
- [ ] `bun run --cwd apps/web test:unit` passes.
- [ ] `bun run --cwd apps/web check` passes.
- [ ] `bun run --cwd apps/web build` passes.
- [ ] `git diff --check` passes.

## Notes

- The issue was discovered while probing local QA chat concurrency/error
  recovery, but this task is not an enterprise pressure-test task.
- The root cause is frontend normalization mismatch, not a required backend
  contract change.

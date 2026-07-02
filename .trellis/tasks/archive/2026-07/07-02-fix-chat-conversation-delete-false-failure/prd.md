# Fix chat conversation delete false failure

## Goal

Deleting a conversation from the Q&A sidebar should not show a failure message when
the Gateway successfully deletes the session.

User value: the chat UI should reflect the actual backend outcome. A successful
delete should remove the session from the sidebar without a false
"delete failed" toast/error.

## Confirmed Facts

- User symptom: clicking delete in the Q&A sidebar shows the delete failure
  message, but refreshing the page shows the session is gone.
- UI call chain:
  - `apps/web/src/pages/qa/chat/page.tsx` `handleDelete` catches mutation errors
    and sets the failure message.
  - `apps/web/src/features/qa/hooks/use-conversations.ts` `useDeleteSession`
    calls `deleteSession(id)`.
  - `apps/web/src/api/conversations.ts` `deleteSession` currently calls
    `gatewayRequest<void>(..., { method: 'DELETE' })`.
- Gateway contract: `DELETE /api/v1/qa-sessions/{sessionId}` returns `204` with
  no response body (`content?: never`) in `docs/services/gateway/api/public.openapi.yaml`
  and `apps/web/src/api/generated/gateway.ts`.
- `gatewayRequest` unwraps a JSON envelope through `requestEnvelope`, which calls
  `response.json()`. A successful `204 No Content` response therefore rejects
  during parsing even though `response.ok` is true.
- Other no-content delete wrappers in the frontend already use `requestVoid`.
- Red test evidence: `bun run --cwd apps/web test:unit -- src/api/conversations.test.ts`
  failed on the stale working branch with `SyntaxError: Unexpected end of JSON input`
  from `requestEnvelope -> requestJson -> deleteSession`.
- Upstream status: after fetching `upstream/develop`, commit `1d588f6`
  (`fix(frontend): delete session uses requestVoid for 204 empty responses`) already
  contains the production fix. The current branch is based on `upstream/develop`
  and adds regression coverage instead of duplicating the production fix.

## Requirements

- `deleteSession(sessionId)` must resolve when Gateway returns `204 No Content`.
- `deleteSession(sessionId)` must continue to reject on non-2xx Gateway errors,
  preserving normalized `ApiError` details.
- The Q&A sidebar delete flow must keep its existing UI behavior: remove from
  local UI only after the mutation resolves, and keep showing the failure message
  for real request failures.
- The fix must stay inside the frontend Gateway API wrapper layer; do not change
  the generated OpenAPI file or bypass Gateway.
- Add a regression test that reproduces the `204` delete success case before
  changing production code.

## Acceptance Criteria

- [x] A focused unit test fails on the current implementation because
  `deleteSession('session-1')` rejects when `fetch` returns `204`.
- [x] After the fix, the same test passes and asserts that the request method is
  `DELETE` and the URL is `/api/v1/qa-sessions/session-1`.
- [x] Existing Gateway error behavior remains covered by the API client tests.
- [x] Relevant frontend checks pass: focused unit test, `bun run --cwd apps/web check`,
  `bun run --cwd apps/web build`, and `git diff --check`.

## Verification

- Red: `bun run --cwd apps/web test:unit -- src/api/conversations.test.ts`
  failed on the stale branch with `SyntaxError: Unexpected end of JSON input`.
- Green on latest develop branch:
  `bun run --cwd apps/web test:unit -- src/api/conversations.test.ts src/api/client.test.ts`
  passed, 9 tests.
- Full unit on latest develop branch: `bun run --cwd apps/web test:unit`
  passed, 80 tests in 25 files.
- Quality on latest develop branch: `bun run --cwd apps/web check` passed.
- Build on latest develop branch: `bun run --cwd apps/web build` passed with the
  existing Vite large chunk warning.
- Patch hygiene on latest develop branch: `git diff --check` passed.

## Out Of Scope

- Changing the chat sidebar confirmation UX.
- Changing backend delete semantics.
- Editing generated OpenAPI output.

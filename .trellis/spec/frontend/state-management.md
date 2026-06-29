# State Management

> Rules for local, URL, server, persisted, and global state.

## State Categories

| Category                | Owner                                     | Examples                                                                    |
| ----------------------- | ----------------------------------------- | --------------------------------------------------------------------------- |
| Server state            | TanStack Query                            | Knowledge bases, documents, report records, user profile, settings.         |
| Local UI state          | React state                               | Dialog open state, active tab, local form toggles.                          |
| URL state               | TanStack Router search params             | Filters, pagination, selected tab when shareable/bookmarkable.              |
| Global client state     | Zustand                                   | Sidebar collapsed state, theme, chat draft/session cache, auth shell hints. |
| Form state              | React Hook Form                           | Login, knowledge base settings, model config, report parameters.            |
| Persisted browser state | Zustand persist or IndexedDB/localStorage | Chat session list, local drafts, UI preferences.                            |

## Server State

- Server-owned data belongs in TanStack Query.
- Prefer query invalidation after mutations over manually synchronizing many local stores.
- Use query keys that include all relevant filters and route params.
- Use optimistic updates only for low-risk UI interactions with clear rollback behavior.

## URL State

Put state in the URL when:

- The state affects list contents.
- The state should survive refresh.
- The state should be shareable or bookmarkable.

Examples: table page, page size, search keyword, document status filter, knowledge base type, retrieval test parameters when useful.

## Zustand

Use Zustand for small global client state only:

- Current UI theme and sidebar state.
- Chat local session list and unsent drafts.
- Lightweight auth shell state derived from `/api/me`.
- Cross-route UI preferences.

Do not put paginated lists, documents, report records, model settings, or permission matrices in Zustand if they come from the backend.

## Chat State

- Store server-backed chat history through backend APIs when available.
- Use local persistence for client-only drafts and temporary session recovery.
- Message state should distinguish `pending`, `streaming`, `done`, and `error`.
- Persist enough metadata to restore sessions after refresh, but avoid caching sensitive documents or credentials in localStorage.

Recommended shape:

```ts
type ChatMessage = {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  status: 'pending' | 'streaming' | 'done' | 'error'
  citations?: Citation[]
  reasoningSteps?: ReasoningStep[]
  createdAt: string
}
```

## Long-Running Tasks

- Document parsing/vectorization and report generation must expose explicit frontend states.
- Prefer backend task status endpoints or SSE events over inferred frontend timers.
- UI should show progress, retry/failure actions, and final output links where available.

## Common Mistakes

- Mirroring TanStack Query data into Zustand.
- Keeping filters only in component state when they should survive refresh.
- Persisting secrets, API keys, or sensitive source content in browser storage.
- Treating streaming content as a final answer before the `done` event.

## Scenario: Auth Session Store And Route Guards

### 1. Scope / Trigger

- Trigger: frontend authentication, session restore, logout, AppShell rendering,
  or RBAC route/menu filtering.
- Scope: browser code under `apps/web/src/`; backend identity propagation remains
  owned by gateway and downstream services.

### 2. Signatures

- `POST /api/v1/sessions` creates a login session.
- `POST /api/v1/users` creates a user and returns a session.
- `GET /api/v1/users/me` restores the current user from the opaque Bearer token.
- `DELETE /api/v1/sessions/current` deletes the current session.
- Store actions should expose `login`, `register`, `restoreSession`, `logout`,
  and `clearSession`.

### 3. Contracts

- Persist only the opaque `accessToken`; do not persist roles, permissions, or
  user profile as durable browser state.
- Send the token only as `Authorization: Bearer <accessToken>`.
- Never decode the token or treat it as JWT claims.
- Browser code must not send `X-User-Id`, `X-User-Roles`, or
  `X-User-Permissions`; gateway injects those headers downstream.
- Route guards must wait for `restoreSession` before deciding whether to render,
  redirect to login, or show forbidden state.
- Menu filtering improves UX only; route guards and backend authorization remain
  the enforcement layers.

### 4. Validation & Error Matrix

- No local token -> mark auth state anonymous and redirect protected routes to
  login.
- `GET /users/me` returns `401 unauthorized` -> clear the local token and auth
  state, then return to login flow.
- `403 forbidden` or failed RBAC predicate -> render a permission denied state.
- Non-auth restore failure -> keep the token in memory, set an auth error state,
  and show retry UI rather than silently logging the user out.
- Logout failure -> clear local auth state in `finally` so the UI cannot remain
  stuck in a stale authenticated state.

### 5. Good/Base/Bad Cases

- Good: a pathless authenticated route restores the session once, and child
  routes declare permission requirements close to route definitions.
- Base: AppShell hides navigation items the current user cannot access.
- Bad: individual pages call `/users/me` independently and race each other.
- Bad: route access is enforced only by hiding sidebar links.

### 6. Tests Required

- `bun run --cwd apps/web check` must pass after auth shell changes.
- `bun run --cwd apps/web build` must pass after route or store changes.
- Future unit/component tests should assert login success stores only the token,
  `401` clears auth state, forbidden routes render denied UI, and logout clears
  local state even when the API request fails.

### 7. Wrong vs Correct

#### Wrong

```ts
const payload = JSON.parse(atob(accessToken.split('.')[1]))
useAuthStore.setState({ user: payload, permissions: payload.permissions })
```

#### Correct

```ts
apiClient.setToken(session.accessToken)
const user = await getCurrentUser()
useAuthStore.setState({ status: 'authenticated', user })
```

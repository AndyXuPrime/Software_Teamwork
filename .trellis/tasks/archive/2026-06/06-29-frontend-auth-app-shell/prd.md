# Frontend Auth App Shell and RBAC Navigation

## Context

Issue #109 requests the frontend side of authentication, session restore,
authenticated AppShell, and RBAC navigation. The issue is assigned to another
team member, so this task is scoped as an assistance implementation on the
current frontend branch.

Authoritative references:

- `docs/collaboration/frontend-workflow.md`
- `.trellis/spec/frontend/index.md`
- `.trellis/spec/frontend/directory-structure.md`
- `.trellis/spec/frontend/quality-guidelines.md`
- `docs/services/auth/README.md`
- `docs/architecture/frontend-backend-contract.md`
- `docs/services/gateway/api/openapi.yaml`

## Goals

- Add frontend API wrappers for public Gateway auth resources:
  - `POST /api/v1/sessions`
  - `GET /api/v1/users/me`
  - `DELETE /api/v1/sessions/current`
  - keep existing `POST /api/v1/users` support.
- Store only the opaque `accessToken` in the browser and send it as
  `Authorization: Bearer <accessToken>`.
- Never decode the token and never send `X-User-Id`, `X-User-Roles`, or
  `X-User-Permissions` from browser code.
- Implement auth store actions for login, registration session creation,
  current-user restore, logout, and local session clearing on `401`.
- Add route guard behavior:
  - restore session before rendering protected routes,
  - redirect unauthenticated users to login,
  - show forbidden state for authenticated users without required permissions.
- Build an authenticated AppShell with user identity, logout, and navigation
  filtered by roles/permissions.

## Non-Goals

- Do not implement backend auth or Gateway cache behavior.
- Do not edit generated OpenAPI files manually.
- Do not add frontend calls to auth internal addresses or other internal
  service URLs.
- Do not claim primary ownership of the GitHub issue.

## Acceptance Criteria

- `bun run --cwd apps/web check` passes.
- `bun run --cwd apps/web build` passes.
- `git diff --check` passes.
- Protected pages cover restoring/loading, unauthenticated login, forbidden,
  and API error states where applicable.
- Browser requests use the public Gateway base `/api/v1` only.

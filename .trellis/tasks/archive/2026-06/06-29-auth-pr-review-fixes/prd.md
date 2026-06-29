# Fix Auth PR Review Issues

## Goal

Address the three blocking review findings on the auth PR before merge: align
the internal auth contract with implementation, make successful user/session
creation semantics reliable when security event recording fails, and synchronize
readiness response documentation with the handler.

## What I Already Know

- The existing PR implements Auth as the source of truth for users, sessions,
  token hashes, RBAC reads, and security events.
- Review finding 1: OpenAPI declares `serviceTokenAuth` as Bearer auth, but the
  server authenticates service-to-service calls with `X-Service-Token`.
- Review finding 2: `CreateUser` / `CreateSession` can commit user/session data
  and then return an error if post-commit security event recording fails.
- Review finding 3: `/readyz` OpenAPI documents `data.checks` as an object, but
  the handler returns `data.dependencies` as an array.
- Required local checks are `go test ./...`, `go build ./cmd/server`, and
  `git diff --check`.

## Requirements

- Update `services/auth/api/openapi.yaml` so internal auth uses the same
  `X-Service-Token` header contract as the implementation.
- Keep internal route requirements for `X-Caller-Service`, and preserve
  `X-Request-Id` propagation.
- Ensure user/session creation does not return a failure after the durable
  business state has already succeeded solely because a security event write
  fails after commit.
- Prefer transactionally recording security events with the business write when
  practical; if an event is intentionally post-commit, its failure must not make
  a successful business operation look failed.
- Update `/readyz` OpenAPI response schemas to match the actual
  `dependencies` array returned by the handler.
- Add or update focused tests for the changed behavior.
- Push the fix commit to the existing PR branch.

## Acceptance Criteria

- [x] Generated-client contract for service auth points to `X-Service-Token`,
      not `Authorization: Bearer`.
- [x] `POST /internal/v1/users` still creates a user, default role, session,
      and security events on the happy path.
- [x] `POST /internal/v1/sessions` still validates credentials and records
      success/failure events where possible.
- [x] A security event failure after a successful durable user/session write no
      longer causes the handler to return a failed create response.
- [x] `/readyz` OpenAPI schema matches handler response shape.
- [x] `go test ./...` passes in `services/auth`.
- [x] `go build ./cmd/server` passes in `services/auth`.
- [x] `git diff --check` passes for the fix.

## Definition of Done

- Tests cover the event-failure consistency behavior.
- Contract documentation matches implementation.
- The existing PR branch is updated with a conventional fix commit.
- Unrelated local files remain uncommitted.

## Technical Approach

- Inspect current repository/service interfaces to determine whether security
  event writes can be included in existing repository transactions without a
  larger repository abstraction change.
- If the transaction path is larger than appropriate for this PR review fix,
  classify successful creation as authoritative and downgrade post-commit
  security event write failures to structured logging. This avoids retry
  hazards while keeping the event write best-effort for the current slice.
- Update tests to lock the chosen semantics.

## Out of Scope

- Gateway Redis session cache integration.
- Public gateway auth endpoint implementation.
- Full outbox/eventing infrastructure.
- Changing auth token format or RBAC seed content beyond what is needed for the
  review findings.

## Technical Notes

- Relevant files are expected under `services/auth/internal/service`,
  `services/auth/internal/http`, `services/auth/api/openapi.yaml`, and backend
  contract specs if needed.
- Existing work commit on the PR: `070bc5e feat(auth): add user session rbac source of truth`.

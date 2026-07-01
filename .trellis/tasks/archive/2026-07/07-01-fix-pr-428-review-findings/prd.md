# Fix PR 428 review findings

## Goal

Make PR #428's QA session attachment flow conform to the published Gateway contract and remain safe under concurrent uploads and partial failures.

## Background

- PR: `Sakayori-Iroha-168/Software_Teamwork#428`, head `JerryTeam/feat/qa-session-attachments`.
- The current PR is mergeable and its existing CI checks pass at head `26e0360`, but the latest Codex review found five contract/runtime gaps.
- Code inspection also confirmed two earlier findings are still present: quota checks are not atomic, and a successful File upload is not compensated when metadata creation fails.
- The Gateway OpenAPI contract is authoritative for the public attachment shape and lifecycle semantics.

## Requirements

- R1: Allow a full 20 MiB file plus the documented multipart envelope through the Gateway while preserving the normal default body limit for other routes.
- R2: Parse, validate, and apply the attachment `status` list filter through HTTP, service, and repository layers. Supported values are the documented attachment lifecycle statuses.
- R3: Return only the public `SessionAttachmentSummary` fields. Map internal parse failure information to sanitized `errorCode` / `errorMessage` fields and do not expose internal counters, update timestamps, deletion metadata, or file references.
- R4: Represent `purged` in the database lifecycle and mark TTL-cleaned rows as purged while retaining the existing hidden/deleted behavior.
- R5: Add a dependency-safe `goose Down` section for migration `0011_session_attachments.sql`.
- R6: Enable `search_session_attachments` in both new default agent configs and the persisted system default config without changing explicit user tool selections.
- R7: Enforce per-session active attachment count and byte quotas atomically in PostgreSQL by serializing creates on the owning conversation.
- R8: If metadata creation fails after File upload succeeds, attempt an uncancelled compensating File delete and preserve both failures when cleanup also fails.
- R9: Preserve existing MIME validation, current-message attachment search scoping, deletion/chunk cleanup, and public error classification.

## Acceptance Criteria

- [x] A multipart request containing a 20 MiB file is not rejected by the Gateway body limiter solely because of multipart overhead; a body above the route envelope limit is rejected.
- [x] `GET .../attachments?status=ready` reaches the repository with the filter, and an unsupported status returns `400 validation_error`.
- [x] Upload, list, and get responses match `SessionAttachmentSummary`; failed attachments expose sanitized failure fields and private/internal fields are absent.
- [x] Migration 0011 can migrate down in dependency order, and TTL cleanup records `purged` before hiding attachment rows.
- [x] Default agent configuration and the system-default migration include `search_session_attachments`; explicit user-created tool arrays remain untouched.
- [x] Concurrent metadata creates for one session cannot exceed 10 active attachments or 100 MiB, and quota failures remain `409 conflict`.
- [x] Metadata-create failure deletes the just-uploaded file object; a cleanup failure remains observable through the returned error chain.
- [x] Gateway and QA service tests pass, goose Up/Down validation passes, Gateway contract/Compose checks pass, and unrelated pre-existing policy findings are recorded without expanding this PR.

## Out of Scope

- Changing the public 20 MiB / 10 attachments / 100 MiB limits.
- Replacing the current goroutine parser dispatch with an external queue.
- Frontend attachment UI work or unrelated QA behavior.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.

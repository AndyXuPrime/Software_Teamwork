# Fix attachment size limit drift

## Goal

Keep QA attachment upload and processing limits consistent with the authoritative 20 MiB public Gateway contract, and never parse silently truncated file content.

## Background

- PR #428 review at head `e2a5f40` found that Gateway is fixed to 20 MiB plus multipart overhead while `QA_SESSION_ATTACHMENT_MAX_BYTES` can currently be configured above 20 MiB.
- `FileHTTPClient.Read` independently truncates reads at 21 MiB without reporting overflow.
- The Gateway OpenAPI contract defines a 20 MiB single-file maximum, so deployments must not expand the QA-only limit beyond that public contract.

## Requirements

- Reject `QA_SESSION_ATTACHMENT_MAX_BYTES` values above 20 MiB during QA startup validation while continuing to allow smaller positive deployment limits.
- Pass the validated attachment maximum into `FileHTTPClient` as its maximum content read size.
- Read at most `max+1` bytes and return an explicit error when File Service content exceeds the configured maximum; never pass truncated bytes to Parser.
- Keep Gateway's 20 MiB plus 1 MiB multipart envelope unchanged because it implements the public contract ceiling.
- Add regression tests for configuration rejection, exact-limit reads, and over-limit reads.

## Acceptance Criteria

- [x] Default and smaller QA attachment limits load successfully; values above 20 MiB fail with an actionable configuration error.
- [x] `cmd/server` injects the validated attachment limit into `FileHTTPClient`.
- [x] File content at the configured limit is returned intact.
- [x] File content above the configured limit returns an error and no partial data.
- [x] QA and Gateway test/build checks remain green.

## Out of Scope

- Increasing the public OpenAPI 20 MiB limit.
- Adding a separate Gateway environment variable for this route.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.

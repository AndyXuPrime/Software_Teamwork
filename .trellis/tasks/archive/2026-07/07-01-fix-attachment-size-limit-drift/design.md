# Design: Attachment size limit alignment

## Decision

Treat the Gateway OpenAPI 20 MiB single-file maximum as the ceiling. QA's environment variable remains useful only for lowering that ceiling per deployment. This avoids a second Gateway configuration source and preserves the published API contract.

## Data Flow

1. QA config parses `QA_SESSION_ATTACHMENT_MAX_BYTES` and rejects values above 20 MiB.
2. Server wiring passes the validated value into both the attachment service/handler and `FileHTTPClient`.
3. `FileHTTPClient.Read` reads through a `max+1` limiter.
4. A payload larger than max returns a dependency error path; `AttachmentService.Process` marks the attachment failed and does not invoke Parser.

## Compatibility

- Default behavior remains 20 MiB.
- Deployments using lower limits retain their configured value end-to-end.
- Deployments currently setting values above 20 MiB fail fast instead of accepting uploads that the public Gateway cannot serve correctly.

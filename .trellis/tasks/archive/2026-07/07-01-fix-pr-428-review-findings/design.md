# Design: Fix PR 428 review findings

## Boundaries and Data Flow

`Gateway body limiter -> QA attachment handler -> AttachmentService -> File client -> PostgreSQL metadata transaction -> async parser`

- Gateway uses a route-specific 21 MiB request-body envelope: the 20 MiB file contract plus the same 1 MiB multipart allowance used by QA.
- The QA handler owns query parsing and public DTO mapping. Domain/repository structs retain internal fields but are not serialized directly at this boundary.
- The service validates lifecycle status filters before repository access.
- PostgreSQL owns the final quota decision. `CreateAttachment` locks the owning conversation row, counts active attachment metadata, checks aggregate bytes, and inserts in one transaction.
- File upload remains outside the database transaction. Any metadata failure triggers a best-effort uncancelled delete; cleanup failure is joined to the original error.

## Contract Decisions

- Keep the existing public OpenAPI semantics rather than weakening the contract.
- Add `purged` to the database status constraint and set it during TTL cleanup, while continuing to hide `deleted_at` rows from list/get.
- Map failed internal attachments to a stable public `attachment_parse_failed` code and the sanitized internal summary. Non-failed attachments return null failure fields.
- Status filtering accepts the documented enum. `purged` normally returns an empty page because purged rows are hidden by the public deletion semantics.

## Default Tool Migration

- Code defaults include both `search_knowledge` and `search_session_attachments` for newly created default configurations.
- A new migration upgrades only the system-owned version-1 config whose tool list exactly matches the previous default. Its Down section restores that exact previous default. Explicit user arrays are not modified.

## Compatibility and Rollback

- No public route or field is removed from OpenAPI.
- Migration 0011 gains a dependency-safe Down section. The new default-tool migration is independently reversible.
- Repository interface changes are service-local and all fakes/tests are updated together.
- Rollback consists of reverting the Go changes and running the new default-tool migration Down; rolling 0011 Down removes attachment tables by design.

## Test Strategy

- Gateway boundary tests at the route-specific limit.
- QA handler tests for filter propagation, validation response, and DTO privacy/shape.
- Service tests for status validation, default configuration, metadata-failure compensation, and joined cleanup failures.
- Repository/migration checks for transactional quota SQL and lifecycle/down migration behavior, plus full service test suites.

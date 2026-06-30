# Report Base Resources Implementation

## Goal

Finish issue #159 for the document service by verifying and completing report base resource APIs: report types, report templates, template structure, report materials, report statistics, and operation logs. The implementation must match the documented OpenAPI contracts and avoid exposing storage or prompt internals.

## Requirements

- Implement or verify `GET /report-types`, including enabled status and default template references.
- Implement or verify report template CRUD plus `GET/PATCH /report-templates/{reportTemplateId}/structure`.
- Implement or verify report material upload, lookup/list, and delete behavior through file service references only.
- Implement or verify report overview and daily statistics endpoints.
- Implement or verify report operation log query filters and paginated response.
- Ensure file-backed resources do not expose object keys, bucket names, internal URLs, API keys, or raw file internals.
- Ensure operation logs store and return sanitized parameter summaries and metadata.
- Ensure delete behavior for templates and materials is soft-delete or otherwise traceable, so historical reports are not broken.
- Keep scope inside `services/document/` unless contract tests require doc-only alignment.

## Acceptance Criteria

- [ ] `/report-types` response schema matches the authoritative OpenAPI contract.
- [ ] `/report-statistics/daily` response schema matches the authoritative OpenAPI contract.
- [ ] `/report-operation-logs` response schema matches the authoritative OpenAPI contract.
- [ ] Report template delete and report material delete hide resources from normal queries without physically breaking history.
- [ ] Operation logs do not include prompt text, object keys, bucket names, API keys, internal URLs, or raw file references.
- [ ] Relevant unit/integration tests pass for document service.
- [ ] `go test ./...` and `go build ./cmd/server` pass in `services/document`.
- [ ] `git diff --check` passes.

## Definition of Done

- Code and API contract changes are limited to issue #159.
- Tests cover any behavior added or corrected during this task.
- Existing public gateway contract remains respected.
- PR description includes completed scope, verification commands, risks, and `Closes #159`.

## Technical Approach

1. Compare current implementation with `docs/services/document/api/openapi.yaml` and `docs/services/gateway/api/openapi.yaml`.
2. Run baseline tests before behavior edits to identify whether #159 is already mostly implemented.
3. Add failing tests first for any missing schema, sanitization, or soft-delete behavior.
4. Implement minimal fixes in document service.
5. Re-run document service tests and build.

## Out of Scope

- Frontend UI work.
- Authentication implementation.
- Model service or AI Gateway feature work beyond respecting existing interfaces.
- Issue #160 or #162 downstream behavior.
- Unrelated gateway, knowledge, file, QA, or web changes.

## Technical Notes

- Issue: https://github.com/Sakayori-Iroha-168/Software_Teamwork/issues/159
- Depends on closed #158 and #154.
- Related AI Gateway issue #121 is closed; this task must not reimplement AI Gateway behavior.
- Public routing still goes through gateway; document service owns the internal document implementation and service-local OpenAPI.
- Current latest `upstream/develop` already contains many relevant handlers and repository methods, so the main work is verification and closing gaps.

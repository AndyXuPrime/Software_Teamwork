# C-010 Document Cross-Service Integration Contract Tests

## Goal

Complete issue #106 by adding repeatable Document contract tests and handoff
documentation that protect the service boundary between Document, File Service,
Knowledge, and AI Gateway. This is a verification task only; it must not add new
product behavior.

## Background

Issue #106 is `[C-010] Document 跨服务集成契约测试`. The authority documents are:

- `docs/services/document/README.md`
- `docs/services/file/README.md`
- `docs/services/knowledge/README.md`
- `docs/services/ai-gateway/README.md`
- `docs/architecture/service-boundaries.md`

Current `develop` already has Document report templates, materials, report files,
jobs/events, basic DOCX creation, File Service client, Knowledge retrieval
client, and AI Gateway clients. The remaining C-010 work is to make those
boundaries harder to regress through targeted tests and README notes.

## Requirements

1. Add or strengthen Document-to-File contract tests.
   - Verify Document file clients use `/internal/v1/files/**`.
   - Verify `X-Service-Token` is propagated when configured.
   - Verify dependency failures are mapped to `dependency_error`.

2. Add active operation contract tests for Document public responses.
   - JSON responses must use the project envelope and include `requestId`.
   - Public DTOs must use safe field names and must not expose File internal IDs,
     `file_ref`, bucket names, object keys, MinIO URLs, signed URLs, storage
     credentials, provider internals, Qdrant internals, prompts, or internal
     service URLs.
   - Permission and authentication boundaries must stay visible through
     `401`/`403` style errors where existing handlers support them.

3. Add report file content contract tests.
   - Successful content reads return binary content, not a JSON success envelope.
   - Failed content reads return the standard JSON error envelope.
   - The test must cover at least one fake File Service failure path.

4. Use fake File/Knowledge/AI Gateway dependencies where real services are not
   required.
   - Cover template upload, material upload, report file creation/content, and
     generation orchestration boundaries enough to catch accidental direct
     MinIO/provider/Qdrant/internal URL exposure.

5. Update Document README or service docs with local integration guidance for
   frontend/QA handoff.
   - Document the repeatable test command.
   - Document required local services/config for a fuller smoke path.
   - Preserve the current limitation: rich DOCX via Pandoc/LibreOffice and
     Document MCP tools remain follow-up work.

## Out Of Scope

- Adding new report-generation product features.
- Implementing Document MCP tools.
- Implementing the `coal_inventory_audit` generation strategy.
- Implementing Pandoc/LibreOffice rich DOCX generation.
- Starting real Docker Compose or external AI/model providers unless already
  available locally.

## Acceptance Criteria

- [ ] Document File Service clients are covered for `/internal/v1/files/**` and
      `X-Service-Token` propagation.
- [ ] Report file/template/material public responses and operation-log summaries
      are protected by tests against File storage internals and provider/Qdrant
      internals.
- [ ] Report file content success is tested as a binary stream response.
- [ ] Report file content failure is tested as a JSON error envelope.
- [ ] Fake dependency failures are tested as `dependency_error` where applicable.
- [ ] Document README or implementation docs include repeatable C-010 local
      integration guidance.
- [ ] `cd services/document && go test ./...` passes.
- [ ] `cd services/document && go build ./cmd/server` passes.
- [ ] `git diff --check` passes.

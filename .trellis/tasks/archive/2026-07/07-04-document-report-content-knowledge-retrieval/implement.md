# Implementation Plan

## Order

1. Load Trellis package specs for frontend, backend, and shared guides.
2. Add backend failing tests:
   - retrieval failure does not fail content generation and records a degraded event;
   - successful retrieval stores sanitized `KnowledgeSources` on the AI section version.
3. Implement backend service changes:
   - add source snapshot model and sanitization helpers;
   - change retrieval failure to degraded context;
   - record a warning event;
   - attach sources when creating AI section versions.
4. Add persistence/API tests for section version `knowledgeSources`.
5. Implement persistence/API changes:
   - migration for `knowledge_sources_json`;
   - repository marshal/scan support;
   - HTTP DTO and OpenAPI schema updates.
6. Add frontend failing tests:
   - selected Knowledge bases are submitted in `options.knowledgeBaseIds`;
   - no selected Knowledge base omits `knowledgeBaseIds`;
   - Knowledge list load failure does not block content generation.
7. Implement frontend selector and request payload changes.
8. Update `.local/report-content-knowledge-retrieval-progress.md` after major milestones.

## Validation

- `go test ./services/document/internal/service`
- `go test ./services/document/internal/http`
- `go test ./services/document/internal/repository`
- `bun run --cwd apps/web test:unit -- src/pages/reports/generate/page.test.tsx`
- `bun run --cwd apps/web check`
- `bun run --cwd apps/web build`
- `git diff --check`

## Risk Points

- OpenAPI generated frontend types may need regeneration if `knowledgeSources` is added to public schemas.
- Repository integration tests may require local Postgres; if unavailable, report the skipped verification and rely on service/http tests.
- User-facing degraded notice depends on the existing event/result-summary surfaces; avoid failing generation just to surface the notice.

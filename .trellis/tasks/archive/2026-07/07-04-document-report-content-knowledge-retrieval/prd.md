# Document report content knowledge retrieval

## Goal

Report authors can decide whether report body generation should use Knowledge retrieval, select one or more knowledge bases when desired, and still generate report content when the Knowledge dependency is unavailable.

## Background

- The report generation UI is `apps/web/src/pages/reports/generate/page.tsx`.
- `content_generation` currently submits `options: { preserveManualEdits: true, saveResult: true }` without Knowledge selection.
- `useKnowledgeBases` already lists Knowledge bases from Gateway.
- Gateway `CreateReportJobRequest.options` is an open object, so the frontend can send `knowledgeBaseIds` without a public contract break.
- Document generation already reads `options` / `retrieval` and calls `ReportKnowledgeRetriever` when `knowledgeBaseIds` are present.
- Current Document retrieval failures return a dependency error from `loadGenerationContext`, which can fail section generation.
- QA citation code provides useful sanitization/snapshot patterns, but Document must not import QA internal packages.

## Requirements

- Add a compact Knowledge reference selector to the report body generation area.
- The selector must support:
  - no Knowledge reference;
  - one selected knowledge base;
  - multiple selected knowledge bases.
- The selector must match the existing report generation page style: bordered card surface, small label text, chip/button selection pattern, existing `StateBlock` and `InlineNotice` usage where appropriate.
- When one or more knowledge bases are selected, `content_generation` must include the selected IDs in `options.knowledgeBaseIds`.
- When no knowledge base is selected, `content_generation` must omit `knowledgeBaseIds` rather than sending an empty list as a retrieval request.
- If Knowledge retrieval fails during body generation, Document must continue generating the section without Knowledge snippets and must not fail the job for that retrieval error.
- A user-visible, non-fatal signal must make clear that this generation did not use Knowledge for the affected run or section.
- Successful Knowledge retrieval used for generated AI section versions must be traceable with a sanitized source snapshot similar in spirit to QA citation snapshots:
  - knowledge base ID;
  - document ID;
  - chunk ID;
  - document name;
  - section path;
  - content preview;
  - retrieval score.
- Traceability snapshots must not expose service tokens, runtime object keys, internal URLs, raw provider errors, or full prompt content.

## Acceptance Criteria

- [ ] The report body generation UI displays a Knowledge reference selector near the "generate content" action.
- [ ] Selecting knowledge bases and clicking "generate content" sends their IDs under `payload.options.knowledgeBaseIds`.
- [ ] Leaving the selector in "no Knowledge reference" mode creates the same body generation request shape as before, except for unrelated existing options.
- [ ] A Knowledge retrieval error is recorded as a warning/degraded event and body generation continues through the chat generation path.
- [ ] If a section is generated after retrieval failed, the section version records no Knowledge sources and the job does not fail because of the retrieval error.
- [ ] If retrieval succeeds, generated AI section versions include sanitized Knowledge source snapshots.
- [ ] Section version API DTOs and OpenAPI docs expose the safe source snapshot as `knowledgeSources`.
- [ ] Frontend and backend tests cover selected IDs, no-selection behavior, retrieval degradation, and successful source snapshot persistence.

## Out Of Scope

- No changes to Knowledge indexing, embedding, reranking, or runtime configuration.
- No new citation rendering panel inside the report editor beyond the selector and degraded-generation notice.
- No changes to QA service internals or cross-service package imports.
- No Docker/business-service startup changes.

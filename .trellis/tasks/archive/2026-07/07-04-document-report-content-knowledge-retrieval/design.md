# Design

## Frontend

`ReportGeneratePage` will import `useKnowledgeBases` and keep a local `selectedKnowledgeBaseIds` state alongside `selectedMaterialIds`. The value is saved in the existing session snapshot so refresh/resume preserves the user's selection.

The selector will live in the outline step because the user starts body generation from that screen. It will use the existing chip selection style already used by report materials:

- a primary "不引用知识库" chip clears the selection;
- each knowledge base appears as a selectable chip;
- loading, empty, and error states use `StateBlock`;
- if the list fails to load, content generation remains available and falls back to no Knowledge reference.

`handleGenerateContent` will build `options` from the existing body generation options and add `knowledgeBaseIds` only when `selectedKnowledgeBaseIds.length > 0`.

## Backend Degradation

`loadGenerationContext` already owns retrieval. It will be changed so retrieval failure returns a normal generation context plus a degradation warning instead of returning an error.

The generation loop will record a non-fatal report event when retrieval is skipped due to failure. Chat generation then continues without snippets. The prompt only receives `contentPreview`, preserving the existing safe prompt boundary.

## Traceability

Document will introduce a service-local `ReportKnowledgeSource` snapshot modeled after QA citation snapshot principles, not by importing QA internals. The snapshot is safe to expose:

- `knowledgeBaseId`
- `documentId`
- `chunkId`
- `documentName`
- `sectionPath`
- `contentPreview`
- `score`

`ReportSectionVersion` will gain `KnowledgeSources []ReportKnowledgeSource`. AI-generated section versions created after successful retrieval will store sanitized snapshots derived from `ReportKnowledgeSnippet`. Manual section versions keep an empty list.

Persistence will use a new `knowledge_sources_json jsonb NOT NULL DEFAULT '[]'::jsonb` column on `report_section_versions`. API DTOs and OpenAPI docs will expose it as `knowledgeSources`.

## Compatibility

- Existing clients may ignore `knowledgeSources`.
- Existing rows get an empty array via the migration default.
- Existing job `options` remain valid.
- If Knowledge is not configured or no bases are selected, generation behavior remains unchanged.

## Tradeoffs

- Section-version JSONB snapshots are simpler than a normalized citation table and fit the medium task scope.
- Events provide run-level user visibility, while section versions provide durable traceability.
- The frontend does not render a full citation explorer in this task; it exposes selection and degraded-run notice while preserving the source data for existing version APIs.

## Rollback

- Revert frontend selector and request option changes to return to no-selection behavior.
- Revert backend degradation and source snapshot handling.
- If needed, the JSONB migration can be rolled back by dropping `knowledge_sources_json`; no existing core report content data depends on it.

# Fix knowledge document content type display

## Goal

Fix the document type shown on `/admin/knowledge/documents` so users see the actual uploaded file kind such as `PDF`, `TXT`, `MD`, `DOCX`, or `XLSX` instead of multiple unrelated files being displayed as `DOC`.

The durable contract is that Gateway/Knowledge `DocumentSummary.contentType` represents a MIME type. The UI may use filename fallback for compatibility, but the backend must stop exposing Knowledge runtime's broad document class value (`doc`) as a MIME content type.

## Confirmed Facts

- The current frontend table displays `doc.contentType` directly after MIME-to-label mapping in `apps/web/src/pages/knowledge/documents/page.tsx`.
- Live Gateway data already returns `contentType: "doc"` for `.txt`, `.md`, and `.docx` documents, so the primary bug is not a frontend-only formatting mistake.
- Knowledge adapter maps runtime `raw["type"]` directly to public `contentType` in `services/knowledge/internal/adapter/map.go`.
- Knowledge runtime stores `Document.type` as a broad file-extension class. Its `filename_type()` groups many extensions such as `txt`, `md`, `docx`, `xlsx`, and `csv` under `doc`.
- Project data-model documentation says public `contentType` is a MIME type.

## Requirements

- R1. Knowledge adapter must expose a normalized MIME value in `DocumentSummary.contentType`.
- R2. Normalization must prefer real MIME fields if the runtime provides them in the future, then infer from filename/suffix, and only use runtime type as a conservative last fallback.
- R3. Runtime broad type `doc` must not be exposed directly as `contentType`.
- R4. Upload forwarding to Knowledge runtime must preserve the incoming file part content type when available.
- R5. The frontend document table must be resilient to legacy or mixed backend data by deriving the displayed label from `contentType` plus filename fallback.
- R6. Regression tests must cover backend mapping and frontend display for `.txt`, `.md`, `.docx`, and related cases.

## Acceptance Criteria

- [ ] Gateway/Knowledge document list responses for files named `.txt`, `.md`, `.docx`, `.xlsx`, and `.pdf` contain MIME-shaped `contentType` values rather than `doc`.
- [ ] The `/admin/knowledge/documents` table displays distinct file labels for mixed document names even if a legacy response still contains `contentType: "doc"`.
- [ ] Backend unit/contract tests fail on the old `raw["type"] -> contentType` behavior and pass with normalized content types.
- [ ] Frontend page tests cover the legacy `contentType: "doc"` fallback path.
- [ ] No generated OpenAPI files are manually edited.
- [ ] Required checks for touched backend/frontend areas are run or explicitly reported if blocked.

## Notes

- Out of scope: changing Knowledge runtime schema or retroactively rewriting runtime database rows.
- Out of scope: changing the public API shape unless existing `contentType` semantics are insufficient. Current plan keeps the existing field and fixes its value.

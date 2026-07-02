# Adapt parser configs for RAGFlow knowledge runtime

## Goal

Make the existing admin parser-config surface meaningful after PR #440 replaces
legacy Knowledge ingestion with the RAGFlow runtime. When an enabled/default
parser config exists in Knowledge, new knowledge bases created through the
adapter should receive a RAGFlow-compatible `parser_config` so document parsing
can select DeepDOC, PaddleOCR, MinerU, OpenDataLoader, or plain text behavior
instead of always falling back to the vendor runtime default.

The user value is preserving the current parser configuration workflow while
making the new RAGFlow-backed Knowledge implementation functionally cover the
old parser-driven ingestion path.

## Confirmed Facts

- PR #440 changes `services/knowledge` to a contract adapter and moves actual
  ingestion, parsing, chunking, and retrieval to `services/knowledge-runtime`.
- PR #440 removes the standalone `services/parser` runtime and old Go ingestion
  worker/Qdrant path.
- `services/knowledge` still exposes `/internal/v1/parser-configs`, backed by
  the legacy `parser_configs` table when `DATABASE_URL` is set.
- Adapter create/update knowledge-base requests already map public
  `chunkStrategy` to vendor `parser_config`.
- RAGFlow runtime selects parser behavior through
  `parser_config.layout_recognize`, with supported code paths for `DeepDOC`,
  `PaddleOCR`, `MinerU`, `OpenDataLoader`, and plain text.
- RAGFlow runtime can auto-register PaddleOCR/MinerU/OpenDataLoader OCR models
  from environment variables, but compose does not currently pass those
  variables into `knowledge-runtime-api` or `knowledge-runtime-worker`.

## Requirements

- If a caller creates a knowledge base without an explicit `chunkStrategy`, the
  adapter must resolve the effective parser config and set a RAGFlow-compatible
  `parser_config`.
- If a caller provides `chunkStrategy`, explicit user input must win; the
  adapter may merge only safe defaults that do not override supplied keys.
- Parser backend mapping:
  - `builtin` should use RAGFlow DeepDOC defaults.
  - `local_ocr` should use RAGFlow PaddleOCR parsing.
  - `remote_compatible` should map to a RAGFlow parser based on
    `defaultParameters.layoutRecognize` / `layout_recognize` when provided, and
    otherwise default to PaddleOCR for OCR-compatible remote behavior.
  - `tika` and `unstructured` should map to plain text because RAGFlow runtime
    does not implement those exact backends in this PR.
- Supported content types, concurrency, endpoint URL, and parser config identity
  should be preserved in parser_config metadata where practical so the selected
  admin config remains auditable.
- `PADDLEOCR_*`, `MINERU_*`, and `OPENDATALOADER_*` environment variables should
  be passed to both RAGFlow runtime containers from compose.
- Existing parser-config CRUD behavior must continue to work.
- Keep the change scoped to PR #440's adapter/runtime architecture. Do not
  restore `services/parser` or the old Go ingestion worker.

## Acceptance Criteria

- [x] Creating a knowledge base without `chunkStrategy` uses the effective
  default parser config to send vendor `parser_config.layout_recognize`.
- [x] Creating or updating with explicit `chunkStrategy.layout_recognize`
  preserves the caller-supplied value.
- [x] Unit tests cover backend mapping for `builtin`, `local_ocr`,
  `remote_compatible`, `tika`, and explicit chunk strategy overrides.
- [x] Compose config passes OCR provider environment variables to
  `knowledge-runtime-api` and `knowledge-runtime-worker`.
- [x] Focused Go tests for `services/knowledge` pass.
- [x] Docker policy and compose config checks relevant to the compose/env change
  pass or are reported with a concrete blocker.

## Out Of Scope

- Building a new frontend control for RAGFlow parser choices.
- Restoring or running the deleted `services/parser` implementation.
- Proving OCR quality across a document corpus.
- Full live RAGFlow runtime E2E if local runtime dependencies are unavailable;
  focused unit and compose checks are the required baseline for this task.

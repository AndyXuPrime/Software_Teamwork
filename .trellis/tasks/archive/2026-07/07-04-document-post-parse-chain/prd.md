# Document post-parse chain

## Goal

Make this branch deliver the Knowledge runtime post-parse architecture for
layout-aware ingestion:

```text
PaddleOCR raw result
  -> canonical layout blocks
  -> deterministic normalizer
  -> confidence/quality gate
       clean -> validated sections
       dirty -> LLM repair -> content-fidelity validation -> validated sections
  -> hierarchy-aware chunker
  -> chunks with inherited context
  -> embedding/index
```

The implementation must improve document structure carried into retrieval while
keeping Knowledge service boundaries intact. `services/knowledge` remains the Go
business adapter. `services/knowledge-runtime` remains the Python runtime owner
for parsing, chunking, embedding, indexing, and retrieval.

This round may add unit/contract tests and opt-in smoke hooks, but it must not
build RAGAS, offline retrieval evaluation, score dashboards, or ground-truth
quality metrics. The gate may use internal routing signals to decide whether a
span needs repair; those signals are not product evaluation metrics.

## Confirmed Facts

- The old standalone `services/parser` runtime is retired. Knowledge document
  parsing must route through `services/knowledge` and
  `services/knowledge-runtime`, not a restored parser service
  (`.trellis/spec/backend/index.md`, `docs/architecture/service-boundaries.md`).
- The current runtime README already documents the intended PaddleOCR data flow
  as raw result -> adapter -> markdown/layout normalizer -> semantic sections
  with metadata -> chunker -> embedding/index.
- PaddleOCR already has a first implementation slice:
  `deepdoc/parser/paddleocr_adapter.py` adapts response variants into
  `PaddleOCRPage`/`PaddleOCRBlock`; `paddleocr_normalizer.py` converts blocks to
  `SemanticSection`; this branch migrates `paddleocr_parser.py` to produce
  layout-aware chunks through the post-parse chain rather than relying on the
  old tuple-based chunking path.
- The active worker path is
  `rag/svr/task_executor.py::do_handle_task` -> `build_chunks` ->
  `rag/svr/task_executor_refactor/chunk_builder.py::run_chunking` ->
  `rag/app/*.py` parser/chunker -> post-processing -> embedding -> index insert.
- `rag/app/naive.py` currently merges parsed PDF/Markdown sections with
  `naive_merge` and tokenizes them into dictionaries consumed by embedding and
  the doc engine.
- `services/knowledge/internal/adapter/map.go` already has a `SectionPath`
  response field on document chunks, but the current runtime mapping does not
  populate it.
- Runtime chunk list/get routes build their own chunk response fields, so any
  new section context fields must be added to runtime response mapping and Go
  adapter mapping intentionally.
- Runtime model calls should go through the existing `LLMBundle` / AI Gateway
  provider path. This feature must not introduce direct provider clients in
  parser code.

## Requirements

### R1. Canonical layout block contract

Define a runtime-local canonical block contract for PaddleOCR output. It must
preserve raw text, normalized text, source block IDs, page number, order, block
type, bbox/position, raw label, and parser source without leaking provider
credentials or full raw responses past the parser boundary.

### R2. Deterministic normalizer first

Use deterministic parsing and cleanup for normal documents before any LLM call.
The normalizer must handle headings, paragraphs, lists, formulas, captions,
images without text, HTML tables, pipe tables, and Markdown fragments.

### R3. Quality gate for routing, not evaluation

Add a confidence/quality gate that classifies blocks or block windows as clean
or dirty using bounded deterministic signals. It must decide repair routing only;
it must not create RAGAS-style evaluation metrics, dashboards, labeled datasets,
or retrieval-quality scores.

### R4. Default dirty-only LLM repair

LLM repair is default-on for the PaddleOCR post-parse chain, but it may run only
on dirty block windows with minimal neighboring context. It must return strict
structured output and must not rewrite an entire document or bypass canonical
block IDs. If no chat model is available, the repair call times out, or the
repair fails validation, the pipeline must fall back to deterministic output.

### R5. Content-fidelity validation

Every repaired output must pass a deterministic content-fidelity validator before
it becomes a validated section. The validator must reject repaired text that
drops numeric facts, dates, units, identifiers, table cells, or large raw text
segments, or that adds unsupported facts. Rejection must not fail the whole
document when deterministic output can continue.

### R6. Validated semantic sections

Produce validated sections with stable metadata: section ID, block IDs, block
type, heading level, section title, section path, page range, bbox/positions,
repair status, and quality flags. These sections are the input to the chunker.

### R7. Hierarchy-aware chunking

Add a hierarchy-aware chunker for validated sections. It must keep tables and
formulas intact when possible, split oversized sections on paragraph/list
boundaries, avoid crossing unrelated sibling sections, and attach inherited
section context to each chunk.

### R8. Chunk context in embedding/index contract

Chunks must carry inherited context in runtime index payloads without exposing
vectors or provider internals. At minimum, chunk payloads should support
`section_path` and metadata for source page/block IDs. If embedding uses a
context-enriched text field, display content should remain readable and not be
silently replaced by hidden prompt-style text.

### R9. Scope containment and migration

The first implementation slice should target the PaddleOCR PDF path and migrate
that path to the post-parse chain. Existing non-PaddleOCR parsers must keep
their current behavior unless a small compatibility hook is needed for shared
chunk metadata. Legacy tuple rendering may remain only as parser compatibility
surface; PaddleOCR chunking must not fall back to the old tuple-based
`naive_merge` path.

### R10. Testable implementation

Add tests for pure functions and contract mappings. Tests may use fake OCR/LLM
responses and should not require real PaddleOCR cloud credentials or real model
providers in ordinary CI.

## Acceptance Criteria

- [ ] PaddleOCR raw layout results can be adapted into canonical blocks with
      stable page/order/type/bbox/source metadata.
- [ ] Deterministic normalization converts clean PaddleOCR blocks into validated
      sections without calling an LLM.
- [ ] Dirty gate classifies at least malformed table, garbled text, fragmented
      headings, and empty/low-information blocks in deterministic unit tests.
- [ ] Dirty-only LLM repair has a fake-client test proving valid repair is
      accepted and invalid repair falls back to deterministic output.
- [ ] PaddleOCR post-parse chain defaults to dirty-only LLM repair when a chat
      model is available, with deterministic fallback when it is unavailable or
      rejected.
- [ ] Content-fidelity validation preserves numeric/date/unit/identifier tokens
      and table shape in tests.
- [ ] Hierarchy-aware chunking produces chunks that inherit `section_path` and
      source block/page metadata.
- [ ] Runtime chunk list/get and Go adapter mapping expose safe section context
      fields without vectors, credentials, prompts, or raw provider payloads.
- [ ] Existing Knowledge adapter tests pass from `services/knowledge`.
- [ ] Targeted Knowledge runtime parser/chunker tests pass from
      `services/knowledge-runtime`.
- [ ] No RAGAS, ground-truth evaluation, retrieval-quality metric storage,
      dashboard, or offline benchmark artifact is added in this branch.

## Out of Scope

- RAGAS integration or any retrieval evaluation framework.
- Offline benchmark dataset design, human labeling workflow, or metric report.
- New service, restored `services/parser`, or agent orchestration.
- Direct provider SDK/client addition for LLM repair.
- Historical automatic reparse of already indexed documents.
- Frontend UI for inspecting layout quality, repair traces, or evaluation
  results.

## Decisions

- LLM repair defaults on for the PaddleOCR post-parse chain.
- Default-on does not mean always trust the LLM. The deterministic normalizer
  remains the source fallback, and content-fidelity validation must accept the
  repair before it reaches validated sections.
- Parser config must not provide a whole-chain off switch that routes PaddleOCR
  back to the old tuple chunking path. Operators may disable dirty-window LLM
  repair only; the post-parse chain remains the active PaddleOCR path.

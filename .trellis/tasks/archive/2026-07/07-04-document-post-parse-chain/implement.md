# Implementation Plan - Document post-parse chain

## Phase 1 - Canonical blocks and deterministic sections

- [ ] Add or extend runtime-local layout data structures for canonical blocks
      and validated sections.
- [ ] Extend `PaddleOCRResultAdapter` to emit stable block IDs and all metadata
      needed by the downstream chain.
- [ ] Extend deterministic normalization for headings, tables, formulas,
      captions, lists, and markdown fragments.
- [ ] Keep section tuple rendering only as parser compatibility output; do not
      use it as a PaddleOCR chunking fallback.
- [ ] Add unit tests for clean PaddleOCR layout, markdown-only response
      variants, OCR-only response variants, HTML table conversion, and
      bbox/page/order preservation.

## Phase 2 - Quality gate and fidelity validation

- [ ] Implement deterministic dirty-window classification.
- [ ] Implement content-fidelity validation helpers for numeric/date/unit/
      identifier preservation, text coverage, table shape, and source block
      preservation.
- [ ] Add tests for dirty signals and repair rejection.
- [ ] Keep all gate outputs as internal routing/debug fields, not evaluation
      metrics.

## Phase 3 - LLM repair integration

- [ ] Add a small repair interface with fake and `LLMBundle` implementations.
- [ ] Read repair options from `parser_config.post_parse_chain.llm_repair`,
      defaulting `enabled` to true for PaddleOCR post-parse chain configs.
- [ ] Build a strict JSON repair prompt for dirty windows only.
- [ ] Add timeout, max-block, and fallback behavior.
- [ ] Add tests with a fake repairer proving accepted repair and rejected repair
      paths.

## Phase 4 - Hierarchy-aware chunker

- [ ] Build a section tree from validated sections.
- [ ] Add chunking that respects hierarchy, table/formula boundaries, token
      budget, and sibling boundaries.
- [ ] Add inherited context fields to chunks: `section_path`,
      `source_block_ids`, `repair_status`, and optional `embedding_text`.
- [ ] Ensure `content_with_weight` remains readable user-facing content.
- [ ] Add tests for nested headings, oversized sections, tables, short headings,
      and inherited context.

## Phase 5 - Embedding/index/API contract mapping

- [ ] Update runtime embedding text extraction to prefer `embedding_text` when
      present.
- [ ] Add safe section context fields to runtime chunk list/get field selection
      and response mapping.
- [ ] Map runtime `section_path` into Go adapter `SectionPath`.
- [ ] Update Knowledge OpenAPI only for fields that become part of stable API
      responses.
- [ ] Add Go adapter mapping tests for section path and safe metadata filtering.

## Phase 6 - Verification and handoff

- [ ] Run targeted Knowledge runtime tests:

```bash
cd services/knowledge-runtime
PYTHONPATH=. uv run --no-project --with pytest --with pytest-asyncio \
  python -m pytest \
  test/unit_test/deepdoc/parser/test_paddleocr_cloud.py \
  test/unit_test/deepdoc/parser/test_layout_post_parse_chain.py \
  test/unit_test/rag/test_layout_hierarchy_chunker.py \
  -q
```

- [ ] Run parser/chunker syntax check if runtime changes are broad:

```bash
rg --files services/knowledge-runtime --glob '*.py' \
  --glob '!**/__pycache__/**' --glob '!services/knowledge-runtime/.venv/**' |
  xargs -r python3 -m py_compile
```

- [ ] Run Knowledge adapter checks:

```bash
cd services/knowledge
go test ./...
go build ./cmd/adapter
```

- [ ] Run `git diff --check`.
- [ ] If a real runtime smoke is available, record it separately as functional
      evidence, not as RAGAS/evaluation evidence.

## Risk Controls

- Keep LLM repair default-on only for dirty PaddleOCR windows, with deterministic
  repair fallback. Do not add a parser-config switch that routes PaddleOCR back
  to old tuple-based chunking.
- Do not restore or reference `services/parser`.
- Do not add direct provider clients in parser code.
- Do not log raw dirty windows by default.
- Do not add RAGAS, ground-truth evaluation datasets, dashboards, or retrieval
  quality metric storage in this branch.

## Rollback

- The PaddleOCR runtime chain is the migrated path. If the new chain causes
  issues, revert the branch commits; do not roll forward by re-enabling old
  tuple-based PaddleOCR chunking.
- Existing documents are not migrated, so rollback does not require database
  cleanup.

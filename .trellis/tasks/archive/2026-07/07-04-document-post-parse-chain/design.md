# Design - Document post-parse chain

## Summary

Do not send the full PaddleOCR raw result directly to an LLM as the primary
normalizer. The safer design is:

1. Canonicalize raw PaddleOCR response variants into small layout blocks.
2. Run deterministic normalization for the normal path.
3. Use a quality gate to isolate dirty block windows.
4. Use LLM repair by default only on dirty windows, then validate content
   fidelity before accepting the repair.
5. Feed validated sections into a hierarchy-aware chunker.
6. Preserve inherited context in chunk payloads for embedding/index and expose
   only safe context fields through Knowledge APIs.

This keeps the LLM as a bounded repair tool instead of making it the source of
truth for document content.

## Evidence From Current Code

- Runtime ownership is documented in `services/knowledge-runtime/README.md`:
  runtime API plus worker own parsing, chunking, embedding, indexing, retrieval,
  and rerank support.
- `deepdoc/parser/paddleocr_adapter.py:25-41` defines `PaddleOCRBlock` and
  `PaddleOCRPage`; `paddleocr_adapter.py:47-51` already adapts layout or OCR
  response shapes.
- `deepdoc/parser/paddleocr_normalizer.py:28-48` defines `SemanticSection` and
  compatibility tuple rendering; `paddleocr_normalizer.py:51-62` normalizes
  `PaddleOCRPage` blocks into sections.
- `deepdoc/parser/paddleocr_parser.py:442-448` previously converted PaddleOCR
  raw result to tuple output through adapter plus normalizer; this branch
  migrates PaddleOCR chunking to post-parse layout chunks instead.
- `rag/app/naive.py:260-306` resolves PaddleOCR through `LLMBundle` model
  configuration and calls `pdf_parser.parse_pdf`.
- `rag/app/naive.py:951-977` parses PDFs into sections/tables, then
  `rag/app/naive.py:1185-1191` calls `naive_merge`/`tokenize_chunks`.
- `rag/svr/task_executor_refactor/chunk_builder.py:70-105` is the parser
  invocation boundary for worker chunking.
- `rag/svr/task_executor_refactor/chunk_service.py:89-180` records raw chunks,
  prepares docs, runs metadata/tag post-processing, and returns chunks for
  embedding.
- `rag/svr/task_executor_refactor/embedding_service.py:58-125` embeds chunks
  after `EmbeddingUtils.prepare_texts_for_embedding`.
- `rag/svr/task_executor.py:1351-1397` standard tasks build chunks, embed them,
  and insert them.
- `api/apps/restful_apis/chunk_api.py:155-190` enumerates list chunk fields and
  maps runtime chunk fields into API chunks.
- `services/knowledge/internal/adapter/map.go:219-229` already includes
  `SectionPath` in the Go chunk DTO, while `map.go:357-388` currently does not
  populate it.

## Data Contracts

### CanonicalLayoutBlock

Runtime-local Python dataclass or pydantic-like shape:

```python
{
    "id": "p1-b0003",
    "source": "paddleocr.layoutParsingResults.prunedResult.parsing_res_list",
    "page_number": 1,
    "order": 3,
    "block_type": "heading|paragraph|list|table|formula|caption|image|unknown",
    "raw_label": "paragraph_title",
    "raw_text": "...",
    "normalized_text": "...",
    "bbox": [left, top, right, bottom],
    "raw_ref": {"block_id": "...", "group_id": "..."}
}
```

Keep this inside the runtime parser/chunker boundary. Do not store full raw
PaddleOCR payloads in the doc engine or Go adapter responses.

### ValidatedSection

```python
{
    "id": "sec-0007",
    "title": "4 Scrap Criteria",
    "level": 2,
    "section_path": ["4 Scrap Criteria", "4.1 Moisture"],
    "block_type": "paragraph",
    "text": "...",
    "page_start": 3,
    "page_end": 4,
    "positions": [[3, left, right, top, bottom]],
    "source_block_ids": ["p3-b0010", "p3-b0011"],
    "repair_status": "clean|repaired|repair_rejected|repair_skipped",
    "quality_flags": ["garbled_text", "table_shape_suspicious"]
}
```

`quality_flags` are internal routing/debug signals. They are not RAGAS metrics
or retrieval evaluation fields.

### Chunk Payload

Recommended runtime index fields:

- `content_with_weight`: user-visible chunk content, still readable.
- `embedding_text`: optional context-enriched text used for embeddings when
  present.
- `section_path`: stable display/filter context, e.g. `"4 Scrap Criteria > 4.1 Moisture"`.
- `section_title`, `section_level`, `source_block_ids`, `repair_status`.
- Existing fields remain: `doc_id`, `kb_id`, `docnm_kwd`, `position_int`,
  `page_num_int`, `important_kwd`, `question_kwd`, vectors, etc.

Update `EmbeddingUtils._extract_content` to prefer `embedding_text` before
`content_with_weight`, while runtime chunk API and Go adapter continue to expose
display content from `content_with_weight`.

## Pipeline Details

### 1. Adapter

Extend the existing PaddleOCR adapter rather than replacing it. It should:

- assign stable IDs by page/order;
- preserve raw label, block ID, group ID, bbox, markdown source, and page size;
- normalize response variants from `layoutParsingResults`, `markdownText`, and
  `ocrResults`;
- keep ordering deterministic even when PaddleOCR returns missing or string
  order fields.

### 2. Deterministic normalizer

Move the current `PaddleOCRLayoutNormalizer` behavior into a more explicit
post-parse chain module, or extend it in place if the diff stays smaller.

It should handle the 95 percent clean path:

- headings: normalize heading markers and infer levels from markdown markers,
  block label, numbering pattern, and neighboring headings;
- paragraphs/lists: preserve line breaks only where they carry structure;
- tables: convert HTML tables to pipe tables and preserve row/cell order;
- formulas: preserve formula blocks as opaque text;
- images/captions: do not hallucinate image content; keep captions and source
  metadata;
- markdown fragments: use the existing Markdown parser protections for code
  fences and tables where useful.

### 3. Quality gate

The gate returns clean sections plus dirty block windows. Candidate dirty
signals:

- garbled text: replacement characters, extreme punctuation/control-character
  density, or repeated broken OCR tokens;
- table shape issues: failed HTML table parse, inconsistent row widths after
  deterministic conversion, or table text collapsed into one paragraph;
- fragmented headings: several adjacent short heading-like blocks without body;
- block order conflicts: repeated order IDs or obvious bbox/order inversion;
- low-information sections: empty text after image stripping, whitespace-only
  blocks, or non-text image blocks without captions;
- unsupported provider output shape: raw result contains text but no usable
  canonical blocks.

This gate must be cheap, deterministic, and bounded. It must not persist
evaluation scores.

### 4. LLM repair

Use an internal `LayoutRepairer` abstraction with a fake implementation for
tests and an `LLMBundle` implementation for runtime. The prompt should receive:

- dirty block window only;
- neighboring clean section titles or short context;
- page/order/type/bbox metadata;
- explicit instruction to preserve all source content and output strict JSON.

Do not include credentials, raw provider responses, or whole-document content in
logs. Recommended config:

```json
{
  "post_parse_chain": {
    "llm_repair": {
      "enabled": true,
      "llm_id": "",
      "max_blocks_per_call": 12,
      "timeout_seconds": 45
    }
  }
}
```

If `llm_repair.enabled` is false, no chat model is configured, or repair fails,
use deterministic output with `repair_skipped` or `repair_rejected`. Default-on
therefore means "attempt repair for dirty windows when safe", not "make LLM
output authoritative".

`post_parse_chain.enabled=false` is not a supported rollback path for PaddleOCR.
The whole PaddleOCR PDF path is migrated to the post-parse chain; only
dirty-window LLM repair can be disabled.

### 5. Content-fidelity validation

Validation should be hard-rule based:

- all numeric/date/unit/identifier tokens from the dirty window must appear in
  repaired output;
- repaired output must not introduce new numeric/date/unit/identifier tokens;
- raw text coverage must stay above a conservative threshold after whitespace
  and punctuation normalization;
- table row/cell count must not shrink unless deterministic extraction already
  proved empty cells;
- source block IDs and page range must be preserved;
- repaired text must parse as the declared block type.

If validation fails, keep deterministic output and mark the repaired attempt as
rejected. This keeps the document ingestible without trusting unverified LLM
rewrites.

### 6. Hierarchy-aware chunker

Build a section tree from validated sections:

- use explicit heading levels first;
- fallback to numbering patterns such as `1`, `1.2`, `I.`, Chinese chapter or
  section markers, and bbox/order cues;
- keep a stack of ancestor headings;
- group body blocks under the nearest heading.

Chunking rules:

- do not cross top-level or sibling section boundaries by default;
- keep table/formula blocks intact where token budget allows;
- split oversized paragraphs/lists by existing delimiters with overlap only
  inside the same section;
- merge very short headings with the following body;
- attach inherited context to every chunk:
  - `section_path` metadata for display/filtering;
  - `embedding_text` such as `"Section: A > B\n\n<content>"` for embedding;
  - original `content_with_weight` remains the chunk body.

### 7. Embedding, Index, Retrieval

Keep the existing embedding/index flow. The minimal integration is:

1. Chunker returns dictionaries compatible with current `ChunkService`.
2. `EmbeddingUtils._extract_content` prefers `embedding_text` when present.
3. Doc engine stores `section_path` and source metadata as ordinary fields.
4. Runtime `chunk_api` includes safe section context fields in list/get
   responses.
5. Go adapter maps `section_path` to the existing `SectionPath` field and keeps
   detailed source metadata inside the safe metadata map.

Do not expose vectors, raw OCR payloads, prompts, provider responses, or
credentials.

## Compatibility

- First slice should target PaddleOCR PDF documents.
- Existing DeepDOC, MinerU, Docling, TCADP, text, Markdown, DOCX, table, QA,
  and resume paths should keep current behavior.
- Legacy PaddleOCR section tuple rendering may remain as a compatibility surface
  for callers that still inspect parser sections, but PaddleOCR chunking must
  not fall back to the old tuple-based `naive_merge` path.
- Existing indexed documents are not migrated. Reparse is a separate product
  feature.

## Error Handling

- Deterministic normalizer errors should fail the affected block/window and
  continue where possible.
- LLM repair errors should not fail ingestion when deterministic output exists.
- If the pipeline produces zero indexable chunks, keep the existing failed
  ingestion behavior rather than marking success.
- Logs may include document ID, parser backend, block counts, and redacted
  repair status. Logs must not include credentials, full raw documents, prompts,
  vectors, or provider raw bodies.

## Testing Strategy

- Pure Python unit tests for canonical block adaptation, deterministic
  normalization, quality gate routing, content-fidelity validation, and
  hierarchy-aware chunking.
- Fake LLM repair tests: valid repair accepted; invalid repair rejected and
  fallback preserved.
- Runtime parser tests should not need real PaddleOCR cloud or real LLM
  credentials.
- Go adapter mapping tests for `section_path` and metadata filtering.
- Optional env-gated smoke can be documented, but ordinary CI stays offline and
  deterministic.

No RAGAS or retrieval evaluation metrics in this task.

## Risks

- Adding context to `content_with_weight` would improve retrieval but pollute
  display text. Prefer `embedding_text` plus explicit `section_path`.
- LLM repair can add latency and provider cost. Because this branch chooses
  default-on for dirty-window repair, the implementation must keep max-block,
  timeout, concurrency, and deterministic repair fallback behavior conservative.
- Runtime doc-engine field mapping is dynamic but retrieval/list APIs still need
  explicit field lists.
- Chunk IDs are hash-derived from content. Reparsed documents may get new chunk
  IDs when the new chunker changes content or context.

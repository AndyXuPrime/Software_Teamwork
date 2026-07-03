# Knowledge MCP tool schemas

## Goal

Complete Issue #528 with the field-level contract for the four model-facing Knowledge MCP tools.

## Requirements

- Define `knowledge__search`, `knowledge__list_documents`, `knowledge__get_document`, and `knowledge__get_chunk`.
- For every tool, provide valid JSON input schema, parameter table, defaults/ranges, output schema with required fields, safe examples, and error cases.
- Explain that the Knowledge server registers native names and QA aliasing creates the documented model-facing names.
- Align fields with current Knowledge service models, including actual document status values and retrieval behavior; do not invent rerank scores the service does not return.
- Update Knowledge docs navigation only; keep this PR documentation-only and independent of #525.

## Acceptance criteria

- All four schemas use object/properties/required/additionalProperties=false.
- Search is capped at topK 20, score threshold 0..1, and optional rerankTopN is bounded by topK.
- Pagination, conditional fields, safe error results, and no-sensitive-data guarantees are explicit.
- JSON examples parse successfully and Markdown links resolve.

## Out of scope

- MCP server implementation, deployment/startup guide, frontend, or general MCP protocol documentation.

# Knowledge MCP Server

## Goal

Expose Knowledge domain capabilities as MCP tools on `services/knowledge` for QA, Document, and other MCP clients. Keep architecture simple: MCP → adapter HTTP (+ AI Gateway for one tool); no product-facing MCP on `knowledge-runtime`.

## Parent context

Child of `07-01-ragflow-runtime-vendor`. Design confirmed in parent [`design-knowledge-mcp.md`](../07-01-ragflow-runtime-vendor/design-knowledge-mcp.md).

## Requirements

- MCP server colocated with Knowledge adapter (`services/knowledge`).
- v1 tool catalog (14 tools): retrieval (2), KB CRUD (5), document CRUD + read (7).
- Forward `X-User-Id`, `X-Request-Id` to adapter handlers.
- `search_knowledge` = pure retrieval via existing `knowledge-queries`.
- `answer_from_knowledge` = retrieval + AI Gateway chat (no provider keys in Knowledge).
- CRUD tools map 1:1 to existing adapter REST routes.

## Non-goals (v1)

- RAGFlow upstream MCP (`--enable-mcpserver`) as product entry.
- Direct MCP → `knowledge-runtime`.
- Parser-config admin on MCP.
- QA client wiring (Phase D follow-up).

## Acceptance Criteria

- [x] `tools/list` returns 14 v1 tools with JSON schemas.
- [x] `search_knowledge` parity with adapter contract tests.
- [x] `answer_from_knowledge` returns answer + citations via AI Gateway.
- [x] KB/document CRUD tools call same handler layer as REST adapter.
- [x] MCP contract tests (no live runtime required for Phase A–C unit tests).

## Definition of Done

- [x] Phases A–C implemented and checked; Phase D documented as follow-up.
- [x] Spec updated for Knowledge MCP ownership boundary (`.trellis/spec/backend/mcp-agent-runtime.md`).

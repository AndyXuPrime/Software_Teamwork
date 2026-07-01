# Implementation plan

## Phase A — MCP scaffold ✅

1. Add `internal/mcp` package: server setup, tool registry, context headers.
2. Wire MCP listener in `cmd/adapter` behind `KNOWLEDGE_MCP_ADDR`.
3. Register v1 catalog; implement `search_knowledge`.
4. Unit tests with fake vendor.

## Phase B — Retrieval ✅

- `answer_from_knowledge` + AI Gateway client (`internal/aigateway`).

## Phase C — CRUD ✅

- Remaining 12 tools; multipart `create_document` via `Bridge.DoMultipart`.

## Phase D — Integration (follow-up)

- QA MCP client endpoint config; E2E smoke.

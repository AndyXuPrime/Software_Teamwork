# Knowledge MCP Server — Confirmed Design (2026-07-01)

Status: **confirmed** by product owner. Follow-up implementation task; not part of Phase 6 runtime promotion.

## Goal

Expose Knowledge capabilities as MCP tools for QA, Document, and other products while keeping architecture simple:

```text
Products (MCP Client)
    → Knowledge MCP Server (new, on services/knowledge)
        → knowledge adapter HTTP (:8083)     # retrieve + CRUD
        → ai-gateway chat/completions        # answer_from_knowledge only
        → knowledge-runtime                  # engine, no product-facing MCP
```

Gateway/frontend continue to use REST on `knowledge :8083`. MCP is the **agent/product tool surface**.

## Non-goals

- Do not expose RAGFlow upstream MCP (`--enable-mcpserver`) to products.
- Do not let MCP clients call `knowledge-runtime` directly.
- Do not duplicate parser-config admin on MCP in v1 (REST-only).

## Tool catalog (v1)

### Retrieval

| MCP tool | Description | Backend |
| --- | --- | --- |
| `search_knowledge` | Pure retrieval; chunks + scores + citation fields; **no LLM** | `POST /internal/v1/knowledge-queries` |
| `answer_from_knowledge` | Retrieval + RAG answer synthesis | adapter retrieval → AI Gateway `chat/completions` |

### Knowledge base (container CRUD)

| MCP tool | HTTP |
| --- | --- |
| `list_knowledge_bases` | `GET /internal/v1/knowledge-bases` |
| `get_knowledge_base` | `GET /internal/v1/knowledge-bases/{knowledgeBaseId}` |
| `create_knowledge_base` | `POST /internal/v1/knowledge-bases` |
| `update_knowledge_base` | `PATCH /internal/v1/knowledge-bases/{knowledgeBaseId}` |
| `delete_knowledge_base` | `DELETE /internal/v1/knowledge-bases/{knowledgeBaseId}` |

### Document CRUD + read extensions

| MCP tool | HTTP | Notes |
| --- | --- | --- |
| `list_documents` | `GET /internal/v1/knowledge-bases/{knowledgeBaseId}/documents` | paginated |
| `get_document` | `GET /internal/v1/documents/{documentId}` | includes processing status |
| `create_document` | `POST /internal/v1/knowledge-bases/{knowledgeBaseId}/documents` | multipart upload; async parse/index |
| `update_document` | `PATCH /internal/v1/documents/{documentId}` | |
| `delete_document` | `DELETE /internal/v1/documents/{documentId}` | |
| `list_document_chunks` | `GET /internal/v1/documents/{documentId}/chunks` | read extension |
| `get_document_content` | `GET /internal/v1/documents/{documentId}/content` | read extension |

**Ingestion:** `create_document` covers upload → parse → embed (Phase 4 adapter path). Poll `get_document` for status. Separate `parse_document` / `index_document` tools deferred unless a product requires split orchestration.

## Tool schemas (summary)

### `search_knowledge`

**Input**

```json
{
  "query": "string (required, 1-2000)",
  "knowledgeBaseIds": ["string"],
  "documentIds": ["string"],
  "topK": 10,
  "scoreThreshold": 0.35,
  "rerank": false,
  "rerankTopN": null,
  "tags": ["string"],
  "metadataFilter": {}
}
```

**Output**

```json
{
  "queryId": "kq_...",
  "results": [
    {
      "score": 0.82,
      "knowledgeBaseId": "kb_...",
      "documentId": "doc_...",
      "chunkId": "chunk_...",
      "documentName": "...",
      "contentPreview": "...",
      "content": "..."
    }
  ]
}
```

Maps 1:1 to `KnowledgeQueryRequest` / `KnowledgeQueryResponse` (internal OpenAPI).

### `answer_from_knowledge`

**Input**

```json
{
  "question": "string (required)",
  "knowledgeBaseIds": ["string"],
  "documentIds": ["string"],
  "topK": 8,
  "scoreThreshold": 0.35,
  "modelProfileId": "string (required)",
  "systemPrompt": "optional",
  "maxTokens": 1024
}
```

**Output**

```json
{
  "answer": "string",
  "citations": [
    {
      "index": 1,
      "knowledgeBaseId": "kb_...",
      "documentId": "doc_...",
      "chunkId": "chunk_...",
      "excerpt": "..."
    }
  ],
  "retrieval": {
    "queryId": "kq_...",
    "resultCount": 5
  }
}
```

**Orchestration (Knowledge MCP only):**

1. Call adapter retrieval (same as `search_knowledge`).
2. Build RAG prompt from top chunks.
3. Call AI Gateway with service token + `modelProfileId`.
4. Return answer + structured citations; do not expose raw provider response.

### `create_document`

**Input:** `knowledgeBaseId`, file bytes or `fileRef` (future), optional metadata fields aligned with `UploadDocumentRequest`.

**Output:** document summary + `status` (`uploaded` / `processing` / …). Parse/index async via runtime worker.

Other CRUD tools mirror internal OpenAPI request/response bodies; MCP layer performs field naming only (no business logic).

## Auth and context

MCP server forwards to adapter:

| Header | Source |
| --- | --- |
| `X-User-Id` | MCP session / caller metadata |
| `X-Request-Id` | MCP session or generated |
| `X-User-Roles` / `X-User-Permissions` | optional, when caller provides |

QA validates permissions before `tools/call`; Knowledge MCP re-validates at adapter boundary.

## Product whitelists (examples)

| Product | Default enabled tools |
| --- | --- |
| QA (`knowledge_qa`) | `search_knowledge` only |
| Document / batch | `create_document`, `get_document`, `list_documents`, `search_knowledge` |
| Admin agents | full catalog |

Align with QA `enabled_tool_names_json` in `docs/services/qa/docs/data-models.md`.

## Implementation phases

### Phase A — MCP scaffold

- Add MCP server listener to `services/knowledge` (Streamable HTTP or SSE per project MCP SDK choice).
- In-process or loopback HTTP to existing adapter handlers (prefer shared handler layer over duplicate HTTP).
- Contract tests: fake MCP client → tool call → adapter fake vendor.

### Phase B — Retrieval tools

- `search_knowledge` (maps existing `knowledge-queries`).
- `answer_from_knowledge` + AI Gateway client in knowledge module.

### Phase C — CRUD tools

- KB + document tools mapping adapter routes.
- Multipart handling for `create_document` (size limits, error mapping).

### Phase D — Integration

- QA MCP client config pointing at Knowledge MCP endpoint.
- E2E smoke: `search_knowledge` + optional `answer_from_knowledge` with `ai` profile.

## Acceptance criteria

- [ ] MCP `tools/list` returns v1 catalog (14 tools).
- [ ] `search_knowledge` matches adapter `knowledge-queries` contract tests (field parity).
- [ ] `answer_from_knowledge` returns answer + citations without storing provider keys in Knowledge.
- [ ] Document/KB CRUD tools pass adapter contract parity tests.
- [ ] No MCP route calls `knowledge-runtime` except via adapter/vendorclient.
- [ ] QA doc cross-link: Knowledge owns MCP server; QA owns MCP client.

## References

- Adapter routes: `services/knowledge/internal/adapter/server.go`
- Internal OpenAPI: `docs/services/knowledge/api/internal.openapi.yaml`
- QA MCP ownership: `docs/services/qa/README.md`
- Runtime split: `services/knowledge-runtime/README.md`, Phase 6 `implement-6.md`

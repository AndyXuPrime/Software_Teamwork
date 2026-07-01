# Design — Knowledge MCP Server

Full tool schemas, adapter mapping, and auth: see parent
[`design-knowledge-mcp.md`](../07-01-ragflow-runtime-vendor/design-knowledge-mcp.md).

## Architecture

```text
MCP Client (QA, Document, …)
    → Knowledge MCP Server (new package under services/knowledge)
        → shared adapter handlers / in-process service layer
        → vendorclient (via existing adapter paths)
        → ai-gateway HTTP (answer_from_knowledge only)
```

## Implementation choice (Phase A)

- **Prefer in-process handler reuse** over loopback HTTP to `:8083` (lower latency, same tests).
- Extract or call existing `internal/adapter` handlers from a thin `internal/mcp` layer.
- Transport: Streamable HTTP (align with QA MCP client expectations); port via env `KNOWLEDGE_MCP_ADDR`.

## Tool → handler mapping

| MCP tool | Adapter route |
| --- | --- |
| `search_knowledge` | `POST /internal/v1/knowledge-queries` |
| `answer_from_knowledge` | retrieval handler + new `internal/aigateway` client |
| KB CRUD | `/internal/v1/knowledge-bases*` |
| Document CRUD | `/internal/v1/documents*`, `/internal/v1/knowledge-bases/{id}/documents` |

## Testing

- `internal/mcp/mcp_test.go`: tools/list, search_knowledge with fake vendor (reuse adapter test patterns).
- No integration tag required for Phase A scaffold.

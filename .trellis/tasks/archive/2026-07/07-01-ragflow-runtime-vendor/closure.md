# Task closure — ragflow-runtime-vendor (2026-07-01)

## Delivered

- RAGFlow runtime imported, trimmed, promoted to `services/knowledge-runtime/` (api + worker)
- Knowledge contract adapter (`cmd/adapter`) proxies Gateway `/internal/v1/*` to runtime
- Legacy Go knowledge server, parser client, Qdrant, and `services/parser/` removed from active path
- Compose profile `knowledge-v2`: elasticsearch, runtime-api, runtime-worker, knowledge adapter
- Child task `07-01-knowledge-mcp-server`: 14 MCP tools (archived separately)

## Key commits (branch `L1nggTeam/feat/ragflow-runtime-vendor`)

- Phase 1–5b: adapter, ingestion, legacy cleanup
- `7563b29` Phase 6: promote runtime to first-class service
- `f85b84a` MCP server (child scope, committed on same branch)

## Follow-up (new tasks)

| Item | Notes |
| --- | --- |
| Live vendor E2E | `go test -tags=integration` + full compose smoke |
| `knowledge-runtime` Docker CI | Heavy build; not in Phase 6 |
| Parser-config / goose bridge removal | Gateway admin UI dependency |
| OpenAPI `qdrantCollection` → `docEngine` | Gateway + frontend |
| Archive `docs/services/parser/**` | Docs drift |
| Sync `docs/services/knowledge/docs/implementation.md` | Still references legacy Qdrant/parser |
| QA MCP Phase D | Point QA client at `KNOWLEDGE_MCP_ADDR` |
| Legacy table data migration | goose vs vendor Peewee tables coexist |

## PRD note

Original `prd.md` describes Phase 1 vendor import only. Phases 2–6 expanded scope; see `implement.md`, `implement-5b.md`, `implement-6.md`.

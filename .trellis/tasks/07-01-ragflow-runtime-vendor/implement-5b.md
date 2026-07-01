# Phase 5b: Hard cleanup (no legacy dead code)

## Done

- [x] Repository trimmed to parser-config CRUD only (removed sqlc + goose KB methods)
- [x] Service types trimmed (`ParserConfigRepository` only)
- [x] Deleted `services/parser/` directory and CI workflow
- [x] Removed Qdrant + parser from compose and deploy docs
- [x] `go test ./...` passes under `services/knowledge`

## Remaining (requires contract/front-end change)

- [ ] Rename OpenAPI `qdrantCollection` → `docEngine` in trace (Gateway + frontend)
- [ ] Migrate parser-config admin to vendor chunk_method settings (remove goose bridge)
- [ ] Archive `docs/services/parser/**` and update architecture docs
- [ ] Drop unused goose migrations/tables when parser-config bridge is removed

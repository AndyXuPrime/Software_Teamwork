# Phase 5: Legacy cleanup and vendor-only runtime

## Product decisions (confirmed 2026-07-01)

| Topic | Decision |
| --- | --- |
| Object storage | **MinIO direct** via vendor runtime (`software-teamwork-knowledge` bucket); no File Service handoff in Knowledge upload path |
| Vector retrieval | **Vendor ES/Infinity only**; remove Qdrant client, env vars, and trace references |
| Auth / tenant | **Gateway + Auth service** owns identity; adapter forwards `X-User-Id`; no vendor login/JWT/lazy-create |
| Parser service | **`services/parser` retired**; deepdoc in vendor task executor replaces it |

## Scope

1. Knowledge container runs **adapter only** (remove legacy `cmd/server` binary and mode switch)
2. Compose default stack: adapter mode, no `parser` / Redis / Qdrant deps for Knowledge
3. Delete legacy Go packages: `cmd/server`, `internal/http`, `internal/config`, `internal/worker`, `internal/platform/*`
4. Slim `internal/service` to parser-config admin + shared types/errors used by adapter
5. Move `services/parser` to deprecated / remove from default compose and CI path labels
6. Update spec (`api-contracts.md`), runtime README, deploy README

## Out of scope (follow-up)

- Vendor runtime container in compose (still external `VENDOR_RUNTIME_URL` until vendor Dockerfile lands)
- Data migration from legacy goose KB tables to vendor `knowledgebase` tables
- Removing goose `parser_configs` table (admin CRUD stays via adapter + `DATABASE_URL`)

## Acceptance criteria

- [x] `go test ./internal/adapter/... ./internal/adapterconfig/... ./internal/service/...` passes
- [x] Dockerfile builds single Knowledge binary (adapter)
- [x] Default compose Knowledge service does not depend on `parser` or `redis`
- [x] No imports of `internal/platform/parser` or `internal/platform/vector` remain
- [x] Spec documents vendor-only ingestion and MinIO storage boundary

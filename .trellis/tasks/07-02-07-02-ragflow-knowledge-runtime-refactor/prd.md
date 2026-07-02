# Refactor RAGFlow knowledge runtime vendor

## Goal

Refactor the vendored RAGFlow-based Knowledge runtime so it is maintainable inside this project while preserving the current Knowledge module behavior.

## Scope

- Preserve existing Knowledge core behavior: knowledge base CRUD, document CRUD, upload and parse workflow, chunking, retrieval/RAG search, and vectorization.
- Remove RAGFlow business surfaces that are unrelated to this project's Knowledge module.
- Reduce runtime/container complexity without changing the Go Knowledge adapter contract.
- Add focused tests that protect route exposure, adapter/runtime contracts, parsing flow, and RAG retrieval behavior.
- Keep the local integration Docker Compose usable for project development; do not convert the root compose file into infrastructure-only services.

## Constraints

- Do not remove or rewrite behavior required by `services/knowledge` adapter APIs.
- Do not expose upstream RAGFlow MCP as the product tool surface; the project-owned Knowledge MCP remains in the Go adapter.
- Do not use floating `latest` images.
- Do not set `GOSUMDB=off`.
- Docker and Compose changes must follow `docs/runbooks/docker-build-environment.md` and be validated with the project policy scripts.
- The final PDF E2E must use `DL_T_673-1999.pdf` when that file is available in the worktree or provided test fixture path.

## Acceptance Criteria

- [ ] Runtime HTTP surface only exposes APIs required by the Knowledge adapter and preserved core workflows.
- [ ] Removed code has no remaining imports, package entries, runtime entrypoint paths, docs references, or tests that expect deleted surfaces.
- [ ] Containerization supports local/demo integration and the `knowledge-v2` runtime profile with clear service boundaries.
- [ ] Unit tests cover route registration allowlist and adapter/runtime contract assumptions.
- [ ] Integration tests cover document parse and retrieval/RAG path using deterministic fixtures or gated external dependencies.
- [ ] Docker policy checks, Compose config checks, Go tests/builds, and targeted Python tests pass locally.
- [ ] `DL_T_673-1999.pdf` parse/import E2E passes when the target PDF is present.
- [ ] PR targets `develop` from `L1nggTeam/feat/ragflow-runtime-vendor` and has no Severe/Critical Codex Review findings.

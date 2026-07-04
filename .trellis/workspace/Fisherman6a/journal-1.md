# Journal - Fisherman6a (Part 1)

> AI development session journal
> Started: 2026-06-27

---



## Session 1: Knowledge service local ingest stack

**Date**: 2026-06-28
**Task**: Knowledge service local ingest stack
**Branch**: `Fisherman6a/feat/knowledge-service-contracts`

### Summary

Implemented and verified the local Knowledge Service ingest, vectorization, retrieval, Docker Compose stack, and gateway knowledge contract.

### Main Changes

- Added `services/knowledge/` FastAPI local service with folder ingest, parsing, semantic chunking, local hashing embeddings, PostgreSQL records, and Qdrant upsert/retrieval.
- Added service-local Docker Compose stack for knowledge-api, knowledge-worker, PostgreSQL, Redis, Qdrant, MinIO, Adminer, and Redis Commander.
- Updated gateway OpenAPI and docs so knowledge base CRUD, document processing details, chunks, and knowledge queries are active RESTful contracts.
- Verified `/home/bao/projects/linux` subset into `kb_linux`: 2 ready documents, 31 chunks, Qdrant collection green, retrieval hits with source metadata.
- Left active `.trellis/tasks/*` uncommitted pending explicit task archive or task-record decision.


### Git Commits

| Hash | Message |
|------|---------|
| `54754d4` | (see git log) |

### Testing

- [OK] OpenAPI reference and RESTful path validation.
- [OK] Markdown relative link validation.
- [OK] `python3 -m compileall services/knowledge/app`.
- [OK] `bash -n services/knowledge/scripts/ingest_folder.sh`.
- [OK] `docker compose -f services/knowledge/docker-compose.yml config`.
- [OK] Local Docker stack, `readyz`, `kb_linux` status, PostgreSQL records, Qdrant collection, and `knowledge-queries` retrieval smoke checks.

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 2: Migrate Auth to pgx v5

**Date**: 2026-06-30
**Task**: Migrate Auth to pgx v5
**Branch**: `L1nggTeam/fix/auth-pgx-v5`

### Summary

Migrated services/auth from pgx v4 to pgx v5, regenerated sqlc code, updated repository mappings and docs, verified tests/build/migration smoke, and archived the Trellis task.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `175265f` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 3: A-014 Knowledge contract alignment

**Date**: 2026-06-30
**Task**: A-014 Knowledge contract alignment
**Branch**: `L1nggTeam/test/knowledge-contract-alignment`

### Summary

Aligned Knowledge active-operation contracts for chunks, content, and knowledge-queries; added seeded/fake-backed handler and gateway proxy tests; updated Knowledge/OpenAPI/deploy/Trellis docs with remaining real dependency smoke risk.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `f996fd2` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 4: A-021 Knowledge ingestion real dependency smoke

**Date**: 2026-07-01
**Task**: A-021 Knowledge ingestion real dependency smoke
**Branch**: `L1nggTeam/test/knowledge-ingestion-real-deps`

### Summary

Added env-gated Knowledge ingestion real dependency smoke and Gateway owner route smoke, documented Parser image cache prerequisites, updated backend smoke spec, and verified default/enabled smoke paths plus cleanup.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `ca055db` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 5: F-038 QA empty session reuse

**Date**: 2026-07-03
**Task**: F-038 QA empty session reuse
**Branch**: `Frontend/fix/qa-reuse-empty-chat-session`

### Summary

Implemented QA chat empty-session reuse, added regression coverage for repeated new-conversation clicks including cross-session streaming, and fixed frontend test storage setup so full unit tests pass.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `f4f8c4d5` | (see git log) |
| `fc6226db` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 6: F-038 review follow-up

**Date**: 2026-07-03
**Task**: F-038 review follow-up
**Branch**: `Frontend/fix/qa-reuse-empty-chat-session`

### Summary

Addressed PR review feedback by scoping QA attachment upload state to the owning session, adding cross-session upload regression tests, and cleaning archived F-038 placeholder context rows.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `72e72fda` | (see git log) |
| `c00fc391` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 7: Fix Knowledge runtime China libssl mirror

**Date**: 2026-07-04
**Task**: Fix Knowledge runtime China libssl mirror
**Branch**: `Special/fix/knowledge-runtime-china-libssl-mirror`

### Summary

Replaced the failing TUNA libssl1.1 China artifact URLs with Huawei Cloud HTTPS mirrors for issue #612, added a guard test, verified the new URLs with Mozilla/5.0, and archived the RAG/OCR smoke diagnostic task.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `f00c873b` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete


## Session 8: Fix QA SSE default fatal errors

**Date**: 2026-07-04
**Task**: Fix QA SSE default fatal errors
**Branch**: `Frontend/fix/qa-sse-default-fatal-error`

### Summary

Normalized QA SSE error events without fatal to fatal at the frontend stream API boundary, added regression coverage, updated streaming guidance, and archived F-049.

### Main Changes

(Add details)

### Git Commits

| Hash | Message |
|------|---------|
| `3334fb2c` | (see git log) |

### Testing

- [OK] (Add test results)

### Status

[OK] **Completed**

### Next Steps

- None - task complete

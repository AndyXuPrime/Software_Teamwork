# Design

## Boundaries

The Go service in `services/knowledge` is the project-owned API and adapter boundary. The Python runtime in `services/knowledge-runtime` is a vendored implementation detail used for parsing, chunking, embedding/vector operations, and retrieval support.

The refactor should narrow the Python runtime to the surfaces used by the Go adapter and local integration profiles. Unrelated RAGFlow products, UI/server stacks, upstream MCP, file-management APIs outside the adapter path, and demo-only scripts can be removed only after import and route references are checked.

## Compatibility Strategy

- Preserve existing Go adapter endpoints and request/response expectations.
- Prefer explicit allowlists for runtime HTTP routes over filesystem auto-registration.
- Keep model/provider configuration paths until vectorization setup is replaced or proven unused.
- Preserve database models that are still referenced by parse, chunk, retrieval, or model setup flows.
- Use tests to lock behavior before deleting deeper internals.

## Container Strategy

The root `deploy/docker-compose.yml` remains the project local/demo integration entrypoint. It may include application services under profiles. It should not be reduced to PostgreSQL, MinIO, Redis, and other infrastructure only.

The `knowledge-v2` profile is the integrated runtime path and should continue to include:

- Knowledge adapter service
- Knowledge runtime API service
- Knowledge runtime worker service
- Required infrastructure dependencies

Dockerfiles should optimize for reliable builds first, then speed, size, memory, and storage, matching the project Docker runbook priority.

## Testing Strategy

- Route allowlist unit test: deleted or excluded RAGFlow routes cannot be registered.
- Go adapter contract tests: generated runtime client paths and required API assumptions remain stable.
- Python parse/retrieval tests: focus on deterministic units where possible; gate external DB/object-store/vector dependencies explicitly.
- Compose checks: validate default, `knowledge-v2`, and affected profiles with `deploy/.env.example`.
- Final E2E: run the target PDF through upload/parse/import once the file is available.

## Rollback Strategy

Keep cleanups in small commits or reviewable slices. If a removal breaks core behavior, restore the deleted surface temporarily and add a contract test that captures why it is still required before attempting a narrower cleanup.

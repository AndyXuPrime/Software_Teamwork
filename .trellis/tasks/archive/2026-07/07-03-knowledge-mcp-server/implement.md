# Implementation plan

1. Add Knowledge MCP tool service, four schemas/results, and direct chunk-by-ID service/repository support.
2. Add stateless authenticated MCP handler, mount it in the Knowledge server, and add config validation.
3. Add QA optional Knowledge MCP configuration, discovery-first/fallback behavior, and allowlist constants.
4. Add deploy defaults and focused documentation references required by the code PR.
5. Add service, handler, repository, config, and QA manager tests.
6. Run gofmt, vet, full Knowledge/QA tests and builds, Docker/deploy checks, and diff review.

## Rollback points

- Do not alter existing REST paths or inline tool implementation.
- Do not store MCP credentials in PostgreSQL or logs.
- Keep MCP connection failure non-fatal for QA.

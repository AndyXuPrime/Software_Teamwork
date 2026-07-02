# Design

## Configuration and seed

`deploy/.env.example` supplies QA's existing generic bootstrap configuration:
`MCP_TRANSPORT`, `MCP_SERVER_ALIAS`, `MCP_SERVER_URL`, `MCP_SERVER_TOKEN`, and
`MCP_SERVER_TOKEN_HEADER`. The local seed stores non-secret Document MCP
metadata only. The token remains in the host process environment so seed SQL
does not hard-code ciphertext tied to one encryption key.

## Runtime merge

`ConfigService.LoadRuntimeConfiguration` treats database configuration as the
administrative authority while allowing bootstrap credentials to complete the
seeded local record:

- matching enabled alias + missing stored token: use the bootstrap token;
- matching enabled alias + stored token: keep the stored token;
- matching disabled alias: keep it disabled and do not append bootstrap;
- unrelated database aliases: append bootstrap as an additional server;
- no database records: preserve existing bootstrap behavior.

This changes configuration loading only. Tool discovery, prefixing, policy,
Agent iteration, and MCP calls remain untouched.

## Local workflow

`dev-up.sh` continues to start infrastructure only, applies service migrations,
then runs seeds 001, 002, and the new Document MCP seed. `run-backend.sh` starts
Document and QA on host ports 8085 and 8084. QA then connects to
`http://localhost:8085/mcp` and exposes discovered tools as `document__*` after
the active QA tool allowlist is applied.

## Compatibility and rollback

The seed is idempotent and does not store a token. Existing user-created
Document records retain their stored token. Removing the new seed row and env
variables restores the previous no-Document-bootstrap behavior. Root Compose
is unchanged.

## Dependency

The current branch can validate discovery of the existing nine tools. The
`document__generate_report_from_content` end-to-end call remains gated on
Issue #510 / PR #521 merging; documentation records that dependency explicitly.

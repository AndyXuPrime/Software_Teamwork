# Knowledge MCP protocol documentation

## Goal

Complete Issue #525 with an operator/developer guide for the Knowledge MCP Server: architecture, transport, authentication, startup, QA discovery, tool overview, degraded mode, and troubleshooting.

## Requirements

- Document the existing-process `/mcp` topology selected for #505 and explain server-native versus QA-prefixed tool names.
- Cover Streamable HTTP lifecycle, service token and trusted context headers, environment variables, host-run startup, discovery, timeout/fallback behavior, and safe error handling.
- Give a concise four-tool overview and link to the field-level schema document owned by #528 rather than duplicating it.
- Update Knowledge documentation navigation and remove the stale “later MCP server” placeholder.
- Keep this PR documentation-only and independently reviewable against `develop`.

## Acceptance criteria

- A developer can start, authenticate, inspect, and troubleshoot the Knowledge MCP endpoint from the docs.
- QA aliasing and degraded-mode fallback are unambiguous.
- Tool schema details have one canonical cross-reference, avoiding drift with #528.
- Markdown links resolve and no code/deploy files change.

## Out of scope

- Go implementation, JSON Schema field tables, frontend, Compose services, or protocol tutorial duplication.

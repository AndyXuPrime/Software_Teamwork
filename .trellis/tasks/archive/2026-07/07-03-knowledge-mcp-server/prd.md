# Knowledge MCP server

## Goal

Complete Issue #505 by exposing Knowledge read operations through a token-protected Streamable HTTP MCP endpoint and allowing QA to prefer those dynamically discovered tools while retaining the existing HTTP tool client as a degraded-mode fallback.

## Confirmed facts

- Current `develop` has no Knowledge MCP endpoint; QA builds `search_knowledge` inline over the Knowledge REST client.
- Document already provides the project pattern: stateless JSON Streamable HTTP, trusted user/request headers, safe structured results, and service-layer reuse.
- The S045 prototype is a standalone proxy with only `search_knowledge`; it is useful as protocol reference but does not satisfy the four-tool contract or current infra-only Docker policy.
- Knowledge already owns service methods for retrieval, document listing/detail, and chunk listing. A direct chunk-by-ID read is missing.

## Requirements

1. Add `/mcp` to the existing host-run Knowledge process; do not add a business container or public Gateway route.
2. Expose server-native `search`, `list_documents`, `get_document`, and `get_chunk`; QA alias `knowledge` yields model-facing `knowledge__*` names.
3. Validate inputs, preserve user ownership/permissions from trusted headers, and return bounded safe structured data without vector payloads, object keys, tokens, internal URLs, or stack traces.
4. Add the minimal repository/service read required for chunk-by-ID lookup with owner visibility.
5. Add QA configuration for an optional Knowledge MCP URL/token/header. When discovery succeeds, use MCP tools; when absent or unavailable, retain the existing inline Knowledge tool client.
6. Add local environment defaults for the host-run endpoint and update default Agent tool allowlisting without removing legacy fallback names.
7. Add unit/integration-style SDK tests for auth, `tools/list`, calls, schemas, fallback, and sanitization.

## Acceptance criteria

- Unauthorized `/mcp` requests return 401; authorized SDK initialization and `tools/list` expose all four tools.
- Search, list, document, and chunk calls reuse Knowledge service rules and return structured safe results.
- QA exposes `knowledge__*` when MCP is reachable and retains `search_knowledge` when MCP is unavailable.
- Existing Knowledge REST routes remain unchanged.
- Knowledge and QA `go vet`, `go test ./...`, and builds pass; deploy policy and Compose config remain valid.

## Out of scope

- stdio transport, frontend changes, public Gateway MCP routes, vector/index redesign, write tools, or copying S045 debug commands and local file artifacts.

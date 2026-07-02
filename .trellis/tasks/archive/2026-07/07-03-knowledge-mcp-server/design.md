# Design

## Architecture

Knowledge `cmd/server` mounts a package-owned Streamable HTTP handler at `KNOWLEDGE_MCP_PATH` (default `/mcp`). The handler authenticates with the existing Knowledge service token, derives a service `RequestContext` from QA-propagated headers, and delegates four read-only tools to a narrow `MCPToolService` that calls the existing Knowledge service layer.

QA receives optional `KNOWLEDGE_MCP_URL`, token, header, alias, and timeout configuration. During runtime state construction it attempts MCP initialization and discovery. Success adds a prefixed MCP provider and suppresses the inline Knowledge provider; failure closes the MCP client and installs the existing inline provider, so chat remains functional.

## Tool namespace

The server owns unprefixed names (`search`, `list_documents`, `get_document`, `get_chunk`). QA's generic prefix adapter owns the model namespace (`knowledge__...`). This follows the existing Document boundary and prevents aliases from being hard-coded into the service.

## Data and safety

- Search caps `topK` at 20 and returns ranked safe summaries.
- List responses preserve page/pageSize/total.
- Document results omit file references and raw internal failures.
- Chunk results return the requested chunk text but omit embedding provider/model, Qdrant point ID, and arbitrary unsafe metadata.
- Tool failures use stable safe codes/messages; protocol framing remains SDK-owned.

## Compatibility

The REST API is untouched. Empty Knowledge MCP URL preserves the current path. Connection or discovery failure also falls back rather than preventing QA startup. Removal of the MCP env values restores the old behavior.

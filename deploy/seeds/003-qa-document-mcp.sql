\connect qa_system

-- Keep credentials out of PostgreSQL. QA merges MCP_SERVER_TOKEN from the
-- host-run environment when this seeded alias has no encrypted token.
INSERT INTO mcp_servers (
    id,
    alias,
    display_name,
    transport,
    command,
    args_json,
    endpoint_url,
    token_encrypted,
    token_last4,
    token_header,
    tool_timeout_seconds,
    enabled,
    sort_order,
    tool_count,
    created_by_user_id
)
VALUES (
    '33333333-3333-4333-8333-333333333601'::uuid,
    'document',
    'Document MCP',
    'streamable_http',
    NULL,
    '[]'::jsonb,
    'http://localhost:8085/mcp',
    NULL,
    NULL,
    'Authorization',
    30,
    TRUE,
    10,
    0,
    'local-seed'
)
ON CONFLICT (alias) DO UPDATE
SET display_name = EXCLUDED.display_name,
    transport = EXCLUDED.transport,
    command = EXCLUDED.command,
    args_json = EXCLUDED.args_json,
    endpoint_url = EXCLUDED.endpoint_url,
    token_header = EXCLUDED.token_header,
    tool_timeout_seconds = EXCLUDED.tool_timeout_seconds,
    sort_order = EXCLUDED.sort_order,
    updated_at = now()
WHERE mcp_servers.created_by_user_id = 'local-seed';

-- +goose Up
-- Add missing fields to agent_tool_calls table for MCP tool policy implementation
ALTER TABLE agent_tool_calls
    ADD COLUMN IF NOT EXISTS mcp_server_name TEXT,
    ADD COLUMN IF NOT EXISTS error_code TEXT,
    ADD COLUMN IF NOT EXISTS error_message TEXT;

-- Add index for better query performance on tool calls by server name
CREATE INDEX IF NOT EXISTS idx_agent_tool_calls_tool_name
    ON agent_tool_calls(tool_name);

-- +goose Down
DROP INDEX IF EXISTS idx_agent_tool_calls_tool_name;
ALTER TABLE agent_tool_calls
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS error_code,
    DROP COLUMN IF EXISTS mcp_server_name;

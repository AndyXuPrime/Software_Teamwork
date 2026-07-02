-- +goose Up
-- Upgrade only the untouched system default. Explicit user tool selections,
-- including empty arrays, remain authoritative and are not changed.
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge", "search_session_attachments"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names = '["search_knowledge"]'::jsonb;

-- +goose Down
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names = '["search_knowledge", "search_session_attachments"]'::jsonb;

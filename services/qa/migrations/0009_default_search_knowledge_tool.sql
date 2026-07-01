-- +goose Up
-- Only migrate the system default config (version_no=1, created_by_user_id='system').
-- This config existed before the enabled_tool_names field was added, so its empty array
-- is a migration default value, not a user choice. User-created configs with empty arrays
-- should remain as-is (meaning "disable all tools").
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names = '[]'::jsonb;

-- +goose Down
-- Revert only the system default config
UPDATE qa_config_versions
SET enabled_tool_names = '[]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names = '["search_knowledge"]'::jsonb;

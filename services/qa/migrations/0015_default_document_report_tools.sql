-- +goose Up
-- Upgrade only the untouched system default. Explicit user tool selections,
-- including narrower document/report whitelists, remain authoritative.
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge", "search_session_attachments", "document__generate_report_outline", "document__generate_report_from_content", "document__generate_report_text", "document__get_generation_status", "document__export_report_docx", "document__get_report_result"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names = '["search_knowledge", "search_session_attachments"]'::jsonb;

-- +goose Down
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge", "search_session_attachments"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names = '["search_knowledge", "search_session_attachments", "document__generate_report_outline", "document__generate_report_from_content", "document__generate_report_text", "document__get_generation_status", "document__export_report_docx", "document__get_report_result"]'::jsonb;

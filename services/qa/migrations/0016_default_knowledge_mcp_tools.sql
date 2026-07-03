-- +goose Up
-- Migration 0015 added Document MCP tools to the default enabled_tool_names.
-- This migration adds the Knowledge MCP v1 read-only tools (search,
-- list_documents, get_document, get_chunk) that were defined in
-- DefaultKnowledgeMCPToolNames but never landed in the migration chain.
--
-- The WHERE clause uses version-lock guards aligned with 0015: only the
-- untouched system default (version_no=1, created_by='system') is upgraded.
-- Explicit user tool selections remain authoritative.
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge", "search_session_attachments", "knowledge__search", "knowledge__list_documents", "knowledge__get_document", "knowledge__get_chunk", "document__generate_report_outline", "document__generate_report_from_content", "document__generate_report_text", "document__get_generation_status", "document__export_report_docx", "document__get_report_result"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names @> '["search_knowledge", "search_session_attachments"]'::jsonb
  AND NOT (enabled_tool_names @> '["knowledge__search"]'::jsonb);

-- +goose Down
UPDATE qa_config_versions
SET enabled_tool_names = '["search_knowledge", "search_session_attachments", "document__generate_report_outline", "document__generate_report_from_content", "document__generate_report_text", "document__get_generation_status", "document__export_report_docx", "document__get_report_result"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names @> '["knowledge__search"]'::jsonb;

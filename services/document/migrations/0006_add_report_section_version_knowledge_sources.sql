-- +goose Up
ALTER TABLE report_section_versions
    ADD COLUMN knowledge_sources_json jsonb NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE report_section_versions
    DROP COLUMN IF EXISTS knowledge_sources_json;

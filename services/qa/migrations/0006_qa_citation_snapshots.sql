-- +goose Up
ALTER TABLE citations
    ADD COLUMN IF NOT EXISTS response_run_id UUID REFERENCES response_runs(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS content_preview TEXT,
    ADD COLUMN IF NOT EXISTS is_source_available BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS source_unavailable_reason TEXT;

CREATE INDEX IF NOT EXISTS idx_citations_response_run_id
    ON citations(response_run_id);

CREATE INDEX IF NOT EXISTS idx_citations_external_doc
    ON citations(external_doc_id);

-- +goose Down
DROP INDEX IF EXISTS idx_citations_external_doc;
DROP INDEX IF EXISTS idx_citations_response_run_id;

ALTER TABLE citations
    DROP COLUMN IF EXISTS source_unavailable_reason,
    DROP COLUMN IF EXISTS is_source_available,
    DROP COLUMN IF EXISTS content_preview,
    DROP COLUMN IF EXISTS response_run_id;

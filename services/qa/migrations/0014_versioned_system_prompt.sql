-- +goose Up
-- Add system_prompt column to qa_config_versions
ALTER TABLE qa_config_versions
    ADD COLUMN IF NOT EXISTS system_prompt TEXT NOT NULL DEFAULT '';

-- Add CHECK constraint for max 20000 bytes (drop first for idempotency)
ALTER TABLE qa_config_versions
    DROP CONSTRAINT IF EXISTS ck_qa_config_versions_system_prompt_length;

ALTER TABLE qa_config_versions
    ADD CONSTRAINT ck_qa_config_versions_system_prompt_length
    CHECK (octet_length(system_prompt) <= 20000);

-- Migrate existing system_prompt from qa_runtime_settings into the current
-- active qa_config_versions row. Truncate oversized values to 20000 bytes
-- rather than silently falling back to the bootstrap prompt.
UPDATE qa_config_versions
SET system_prompt = left(runtime.value, 20000)
FROM (
    SELECT value FROM qa_runtime_settings WHERE key = 'system_prompt'
) AS runtime
WHERE qa_config_versions.is_active = true
  AND (qa_config_versions.system_prompt = '' OR qa_config_versions.system_prompt IS NULL)
  AND runtime.value IS NOT NULL;

-- +goose Down
ALTER TABLE qa_config_versions
    DROP CONSTRAINT IF EXISTS ck_qa_config_versions_system_prompt_length;

ALTER TABLE qa_config_versions
    DROP COLUMN IF EXISTS system_prompt;

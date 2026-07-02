-- +goose Up
-- Add system_prompt column to qa_config_versions
ALTER TABLE qa_config_versions
    ADD COLUMN IF NOT EXISTS system_prompt TEXT NOT NULL DEFAULT '';

-- Add CHECK constraint for max 20000 bytes
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_qa_config_versions_system_prompt_length'
          AND conrelid = 'qa_config_versions'::regclass
    ) THEN
        ALTER TABLE qa_config_versions
            ADD CONSTRAINT ck_qa_config_versions_system_prompt_length
            CHECK (octet_length(system_prompt) <= 20000);
    END IF;
END $$;

-- Migrate existing system_prompt from qa_runtime_settings into the current
-- active qa_config_versions row, only if the active row's system_prompt is empty.
DO $$
DECLARE
    v_active_id UUID;
    v_runtime_prompt TEXT;
BEGIN
    -- Get the active QA config version ID
    SELECT id INTO v_active_id
    FROM qa_config_versions
    WHERE is_active = true
    ORDER BY version_no DESC
    LIMIT 1;

    -- Get the runtime system_prompt value
    SELECT value INTO v_runtime_prompt
    FROM qa_runtime_settings
    WHERE key = 'system_prompt';

    -- If we have both an active config and a runtime prompt, migrate it.
    -- Only migrate if the old prompt is within the 20000-byte limit.
    IF v_active_id IS NOT NULL AND v_runtime_prompt IS NOT NULL
       AND octet_length(v_runtime_prompt) <= 20000 THEN
        UPDATE qa_config_versions
        SET system_prompt = v_runtime_prompt
        WHERE id = v_active_id
          AND (system_prompt = '' OR system_prompt IS NULL);
    END IF;
END $$;

-- +goose Down
ALTER TABLE qa_config_versions
    DROP CONSTRAINT IF EXISTS ck_qa_config_versions_system_prompt_length;

ALTER TABLE qa_config_versions
    DROP COLUMN IF EXISTS system_prompt;

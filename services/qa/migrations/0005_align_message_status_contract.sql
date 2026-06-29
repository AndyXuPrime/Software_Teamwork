-- +goose Up
ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_status_check;

ALTER TABLE message_content_blocks
    DROP CONSTRAINT IF EXISTS message_content_blocks_status_check;

UPDATE messages
SET status = 'streaming'
WHERE status = 'generating';

UPDATE message_content_blocks
SET status = 'streaming'
WHERE status = 'generating';

ALTER TABLE messages
    ADD CONSTRAINT messages_status_check CHECK (
        status IN ('queued', 'streaming', 'completed', 'stopped', 'failed', 'cancelled')
    );

ALTER TABLE message_content_blocks
    ADD CONSTRAINT message_content_blocks_status_check CHECK (
        status IN ('queued', 'streaming', 'completed', 'stopped', 'failed', 'cancelled')
    );

-- +goose Down
ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_status_check;

ALTER TABLE message_content_blocks
    DROP CONSTRAINT IF EXISTS message_content_blocks_status_check;

UPDATE messages
SET status = 'generating'
WHERE status IN ('queued', 'streaming', 'stopped');

UPDATE message_content_blocks
SET status = 'generating'
WHERE status IN ('queued', 'streaming', 'stopped');

ALTER TABLE messages
    ADD CONSTRAINT messages_status_check CHECK (
        status IN ('queued', 'generating', 'completed', 'failed', 'cancelled')
    );

ALTER TABLE message_content_blocks
    ADD CONSTRAINT message_content_blocks_status_check CHECK (
        status IN ('generating', 'completed', 'cancelled', 'failed')
    );

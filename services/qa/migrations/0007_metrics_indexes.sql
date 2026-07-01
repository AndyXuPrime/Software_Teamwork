-- +goose NO TRANSACTION
-- +goose Up
-- Indexes for QA metrics aggregation queries.  The combination (role,
-- created_at) speeds up trend, top-queries, and intent-distribution
-- which all filter on role='user' and a date range.  Response run
-- started_at supports the average-latency subquery in overview.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_role_created
    ON messages (role, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_response_runs_started
    ON response_runs (started_at DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_role_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_response_runs_started;

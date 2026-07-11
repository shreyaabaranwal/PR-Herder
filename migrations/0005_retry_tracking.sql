-- 0005_retry_tracking.sql
-- Layer 9 hardening: bounded retries + dead letter queue for
-- webhook_events. Without this, a permanently-broken event (malformed
-- payload, a bug that always errors) retries forever, every worker poll
-- cycle, indefinitely -- this caps it and marks it for manual inspection.

ALTER TABLE webhook_events ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE webhook_events ADD COLUMN IF NOT EXISTS dead_lettered_at TIMESTAMPTZ;

-- Replaces the narrower idx_webhook_events_unprocessed from 0001_init.sql
-- (that one only excludes processed rows; this also excludes
-- dead-lettered ones, since GetUnprocessedEvents' query now filters on
-- both). Keeping the old index too is harmless -- Postgres will just
-- prefer whichever fits the query planner best -- but this one is what
-- actually matches the new WHERE clause.
CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed_not_dead
    ON webhook_events (received_at)
    WHERE processed_at IS NULL AND dead_lettered_at IS NULL;

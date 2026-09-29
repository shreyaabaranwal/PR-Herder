

ALTER TABLE webhook_events ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE webhook_events ADD COLUMN IF NOT EXISTS dead_lettered_at TIMESTAMPTZ;


CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed_not_dead
    ON webhook_events (received_at)
    WHERE processed_at IS NULL AND dead_lettered_at IS NULL;


CREATE TABLE IF NOT EXISTS webhook_events (
    id            BIGSERIAL PRIMARY KEY,
    delivery_id   TEXT NOT NULL UNIQUE,
    event_type    TEXT NOT NULL,
    action        TEXT,
    repo_owner    TEXT NOT NULL,
    repo_name     TEXT NOT NULL,
    pr_number     INT,
    raw_payload   JSONB NOT NULL,
    received_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at  TIMESTAMPTZ,
    process_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_unprocessed
    ON webhook_events (received_at)
    WHERE processed_at IS NULL;

CREATE TABLE IF NOT EXISTS pull_requests (
    id              BIGSERIAL PRIMARY KEY,
    repo_owner      TEXT NOT NULL,
    repo_name       TEXT NOT NULL,
    number          INT NOT NULL,
    title           TEXT NOT NULL,
    body            TEXT NOT NULL DEFAULT '',
    author_login    TEXT NOT NULL,
    association     TEXT NOT NULL,
    base_branch     TEXT NOT NULL,
    head_branch     TEXT NOT NULL,
    head_sha        TEXT NOT NULL,
    additions       INT NOT NULL DEFAULT 0,
    deletions       INT NOT NULL DEFAULT 0,
    changed_files   TEXT[] NOT NULL DEFAULT '{}',
    ci_status       TEXT NOT NULL DEFAULT 'pending',
    state           TEXT NOT NULL DEFAULT 'open',
    merged          BOOLEAN NOT NULL DEFAULT false,
    last_event_id   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (repo_owner, repo_name, number)
);

CREATE INDEX IF NOT EXISTS idx_pull_requests_open
    ON pull_requests (repo_owner, repo_name)
    WHERE state = 'open';

CREATE INDEX IF NOT EXISTS idx_pull_requests_stale
    ON pull_requests (updated_at)
    WHERE state = 'open';


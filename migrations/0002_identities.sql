

CREATE TABLE IF NOT EXISTS identities (
    id            BIGSERIAL PRIMARY KEY,
    slack_user_id TEXT NOT NULL UNIQUE,
    github_login  TEXT NOT NULL,
    linked_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_identities_github_login
    ON identities (github_login);

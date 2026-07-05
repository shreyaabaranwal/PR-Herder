-- 0002_identities.sql
-- Layer 4: Slack user ↔ GitHub identity mapping.
--
-- Design notes:
-- 1. A Slack user ID alone must NEVER be sufficient to authorize a
--    GitHub write (see docs/SECURITY.md invariant #3). This table is
--    the explicit, server-side link that makes authorization possible —
--    it is populated only via a one-time OAuth confirmation flow, never
--    inferred from a Slack display name or email guess.
-- 2. github_login is stored, not a GitHub user ID, because Layer 5's
--    authorization check (checking repo collaborator permissions) is
--    naturally keyed by login in GitHub's REST API responses.
-- 3. linked_at lets us show "connected since" in a future settings UI,
--    and gives an audit trail of when trust was established.

CREATE TABLE IF NOT EXISTS identities (
    id            BIGSERIAL PRIMARY KEY,
    slack_user_id TEXT NOT NULL UNIQUE,
    github_login  TEXT NOT NULL,
    linked_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_identities_github_login
    ON identities (github_login);

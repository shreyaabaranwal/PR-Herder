-- 0003_flaky_history.sql
-- Layer 6: Flaky-CI classifier -- historical check-run records.
--
-- Design notes:
-- 1. One row per check-run observation (not per-PR aggregate). This lets
--    the flaky classifier compute "fail-then-pass-on-retry rate" as a
--    query over raw history, rather than maintaining a running counter
--    that could drift out of sync with reality.
-- 2. Keyed by (repo_owner, repo_name, check_name) for the classifier's
--    query pattern: "show me this check's history across PRs," not
--    "show me all checks for one PR" (that's pull_requests' job).
-- 3. commit_sha is stored because retries on the same PR often re-run
--    against a new commit (force-push, new commit) -- distinguishing
--    "same commit retried" from "new commit, coincidentally same result"
--    matters for an accurate flaky signal.

CREATE TABLE IF NOT EXISTS check_run_history (
    id            BIGSERIAL PRIMARY KEY,
    repo_owner    TEXT NOT NULL,
    repo_name     TEXT NOT NULL,
    pr_number     INT NOT NULL,
    check_name    TEXT NOT NULL,
    commit_sha    TEXT NOT NULL,
    conclusion    TEXT NOT NULL,  -- "success", "failure", "neutral", "cancelled", etc.
    observed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The classifier's primary query: "give me this check's recent history,
-- ordered by time" -- this index makes that a fast index scan rather
-- than a sequential scan as history grows.
CREATE INDEX IF NOT EXISTS idx_check_run_history_lookup
    ON check_run_history (repo_owner, repo_name, check_name, observed_at DESC);

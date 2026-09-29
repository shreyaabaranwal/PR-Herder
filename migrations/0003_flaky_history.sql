

CREATE TABLE IF NOT EXISTS check_run_history (
    id            BIGSERIAL PRIMARY KEY,
    repo_owner    TEXT NOT NULL,
    repo_name     TEXT NOT NULL,
    pr_number     INT NOT NULL,
    check_name    TEXT NOT NULL,
    commit_sha    TEXT NOT NULL,
    conclusion    TEXT NOT NULL,  
    observed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_check_run_history_lookup
    ON check_run_history (repo_owner, repo_name, check_name, observed_at DESC);

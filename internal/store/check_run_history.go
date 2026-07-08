package store

import (
	"context"
	"fmt"

	"github.com/shreyaabaranwal/pr-herder/internal/triage"
)

// RecordCheckRun appends one check-run observation to history. Called
// each time ingest processes a check_run webhook event (Layer 1's
// eventsWeCareAbout map already reserves "check_run"/"check_suite" for
// this -- currently disabled there until this write path existed).
func (s *Store) RecordCheckRun(ctx context.Context, repoOwner, repoName string, prNumber int, checkName, commitSHA, conclusion string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO check_run_history
			(repo_owner, repo_name, pr_number, check_name, commit_sha, conclusion)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, repoOwner, repoName, prNumber, checkName, commitSHA, conclusion)
	if err != nil {
		return fmt.Errorf("record check run: %w", err)
	}
	return nil
}

// GetCheckRunHistory retrieves a check's history for the flaky
// classifier, oldest-first (ClassifyFlaky's commit-grouping logic
// assumes chronological order). Limited to a bounded window (not "all
// history ever") since flaky-ness is a recent-behavior signal --
// something flaky a year ago and fixed since shouldn't still show as
// flaky today.
func (s *Store) GetCheckRunHistory(ctx context.Context, repoOwner, repoName, checkName string, limit int) ([]triage.CheckRunObservation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT commit_sha, conclusion
		FROM check_run_history
		WHERE repo_owner = $1 AND repo_name = $2 AND check_name = $3
		ORDER BY observed_at ASC
		LIMIT $4
	`, repoOwner, repoName, checkName, limit)
	if err != nil {
		return nil, fmt.Errorf("query check run history: %w", err)
	}
	defer rows.Close()

	var observations []triage.CheckRunObservation
	for rows.Next() {
		var obs triage.CheckRunObservation
		if err := rows.Scan(&obs.CommitSHA, &obs.Conclusion); err != nil {
			return nil, fmt.Errorf("scan check run row: %w", err)
		}
		observations = append(observations, obs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate check run rows: %w", err)
	}

	return observations, nil
}

// Package store — repos.go adds a read query over webhook_events to
// answer "which repos have we ever seen," used by the Layer 8 scheduler
// to know which repos to poll for stale PRs. No new table: this reuses
// data that ingest already writes on every webhook delivery.
package store

import (
	"context"
	"fmt"
)

// RepoRef identifies a repo we've seen at least one webhook event from.
type RepoRef struct {
	Owner string
	Name  string
}

// GetDistinctRepos returns every repo_owner/repo_name pair that appears
// in webhook_events. This is a live "which repos are we watching"
// signal, not a config list -- if PR Herder has never received a
// webhook from a repo, that repo won't show up here (which is correct:
// we shouldn't poll repos we're not actually installed on).
func (s *Store) GetDistinctRepos(ctx context.Context) ([]RepoRef, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT repo_owner, repo_name
		FROM webhook_events
		WHERE repo_owner != '' AND repo_name != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("query distinct repos: %w", err)
	}
	defer rows.Close()

	var repos []RepoRef
	for rows.Next() {
		var r RepoRef
		if err := rows.Scan(&r.Owner, &r.Name); err != nil {
			return nil, fmt.Errorf("scan repo row: %w", err)
		}
		repos = append(repos, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repo rows: %w", err)
	}
	return repos, nil
}
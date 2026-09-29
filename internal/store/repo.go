
package store

import (
	"context"
	"fmt"
)


type RepoRef struct {
	Owner string
	Name  string
}


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
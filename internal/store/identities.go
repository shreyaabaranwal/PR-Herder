package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)


var ErrIdentityNotFound = errors.New("no linked GitHub identity for this Slack user")


func (s *Store) LinkIdentity(ctx context.Context, slackUserID, githubLogin string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO identities (slack_user_id, github_login)
		VALUES ($1, $2)
		ON CONFLICT (slack_user_id) DO UPDATE SET github_login = EXCLUDED.github_login, linked_at = now()
	`, slackUserID, githubLogin)
	if err != nil {
		return fmt.Errorf("link identity: %w", err)
	}
	return nil
}


func (s *Store) GetGitHubLogin(ctx context.Context, slackUserID string) (string, error) {
	var githubLogin string
	err := s.pool.QueryRow(ctx, `
		SELECT github_login FROM identities WHERE slack_user_id = $1
	`, slackUserID).Scan(&githubLogin)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrIdentityNotFound
	}
	if err != nil {
		return "", fmt.Errorf("query identity: %w", err)
	}
	return githubLogin, nil
}
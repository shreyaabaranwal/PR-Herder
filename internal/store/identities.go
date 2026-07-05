package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrIdentityNotFound is returned when no GitHub identity is linked to
// the given Slack user. Callers (the authz layer) must treat this as
// "cannot authorize" — never fall back to allowing the action.
var ErrIdentityNotFound = errors.New("no linked GitHub identity for this Slack user")

// LinkIdentity records that a Slack user has confirmed ownership of a
// GitHub account, via whatever one-time OAuth confirmation flow calls
// this (not built yet — this is the storage primitive it will use).
// Upserts on slack_user_id: a user can only ever be linked to one
// GitHub login at a time, and re-linking overwrites the old mapping
// rather than creating ambiguity.
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

// GetGitHubLogin looks up the GitHub login linked to a Slack user ID.
// Returns ErrIdentityNotFound if no link exists — this is the query the
// authz layer calls before allowing any GitHub-mutating action, per
// docs/SECURITY.md invariant #3.
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
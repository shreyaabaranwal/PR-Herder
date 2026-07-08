// Package authz implements PR Herder's authorization gate: the check
// that runs before ANY GitHub-mutating action, per docs/SECURITY.md.
// A Slack identity is never sufficient on its own — this package is
// where a Slack user ID becomes a verified, repo-scoped GitHub
// permission check.
package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shreyaabaranwal/pr-herder/internal/store"
)

// PermissionLevel mirrors GitHub's own collaborator permission levels
// from the repo collaborator-permission API. Kept as a distinct type so
// callers can't accidentally compare against a typo'd raw string.
type PermissionLevel string

const (
	PermissionAdmin PermissionLevel = "admin"
	PermissionWrite PermissionLevel = "write"
	PermissionRead  PermissionLevel = "read"
	PermissionNone  PermissionLevel = "none"
)

// canWrite reports whether a permission level is sufficient for
// mutating actions (approve, label, request-reviewers). admin implies
// write; read and none do not.
func (p PermissionLevel) canWrite() bool {
	return p == PermissionAdmin || p == PermissionWrite
}

// Authorizer checks whether a given GitHub login currently has write
// access to a specific repo, by calling GitHub's REST API directly.
// This check is deliberately NOT cached indefinitely — repo access can
// be revoked, and a stale "yes" would be a privilege-escalation bug, not
// just a staleness bug. (Layer 5 may add a short-TTL cache; Layer 4
// keeps this simple and correct first.)
type Authorizer struct {
	store       *store.Store
	githubToken string // a token with read access to check collaborator permissions
	httpClient  *http.Client
}

func NewAuthorizer(s *store.Store, githubToken string) *Authorizer {
	return &Authorizer{
		store:       s,
		githubToken: githubToken,
		httpClient:  &http.Client{},
	}
}

// CanActOnRepo is the single entry point every mutating Slack interaction
// must call before executing a GitHub write. It performs, in order:
//  1. Resolve the Slack user to a linked GitHub identity (fails closed if
//     unlinked — see store.ErrIdentityNotFound).
//  2. Query GitHub's live collaborator-permission API for that repo.
//  3. Require write-or-above permission.
//
// Any error or insufficient permission returns (false, reason, nil) or
// a non-nil error for unexpected failures — callers must treat both
// "false" and "err != nil" as "do not proceed."
func (a *Authorizer) CanActOnRepo(ctx context.Context, slackUserID, repoOwner, repoName string) (allowed bool, reason string, err error) {
	githubLogin, err := a.store.GetGitHubLogin(ctx, slackUserID)
	if err != nil {
		// Deliberately generic reason string returned to the caller (which
		// will show it in Slack) — we don't want to leak internal error
		// detail to a potentially-untrusted Slack interaction payload.
		return false, "no linked GitHub account — link your account first", nil
	}

	// Repo owner always has write access -- GitHub's collaborator-permission
	// API sometimes does not include the owner in collaborator listings
	// (owner is a distinct category from collaborator), so check this
	// explicitly rather than relying solely on the collaborator API.
	if githubLogin == repoOwner {
		return true, "", nil
	}

	level, err := a.getCollaboratorPermission(ctx, repoOwner, repoName, githubLogin)
	if err != nil {
		return false, "", fmt.Errorf("check github permission: %w", err)
	}

	if !level.canWrite() {
		return false, fmt.Sprintf("@%s does not have write access to %s/%s", githubLogin, repoOwner, repoName), nil
	}

	return true, "", nil
}

type collaboratorPermissionResponse struct {
	Permission string `json:"permission"`
}

// getCollaboratorPermission calls GitHub's
// GET /repos/{owner}/{repo}/collaborators/{username}/permission
// endpoint — the authoritative, real-time source for a user's repo
// permission level. Deliberately not using a cached/derived value.
func (a *Authorizer) getCollaboratorPermission(ctx context.Context, owner, repo, username string) (PermissionLevel, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s/permission", owner, repo, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PermissionNone, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.githubToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return PermissionNone, fmt.Errorf("call github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 404 here typically means "not a collaborator at all" — treat
		// any non-200 as "no permission" rather than erroring the whole
		// flow, since a maintainer clicking a button shouldn't see a
		// confusing 500-style failure for a simple "you don't have
		// access" case.
		return PermissionNone, nil
	}

	var parsed collaboratorPermissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return PermissionNone, fmt.Errorf("decode github response: %w", err)
	}

	return PermissionLevel(parsed.Permission), nil
}
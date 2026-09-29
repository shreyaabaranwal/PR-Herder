
package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/shreyaabaranwal/pr-herder/internal/store"
)


type PermissionLevel string

const (
	PermissionAdmin PermissionLevel = "admin"
	PermissionWrite PermissionLevel = "write"
	PermissionRead  PermissionLevel = "read"
	PermissionNone  PermissionLevel = "none"
)


func (p PermissionLevel) canWrite() bool {
	return p == PermissionAdmin || p == PermissionWrite
}


type Authorizer struct {
	store       *store.Store
	githubToken string 
	httpClient  *http.Client
}

func NewAuthorizer(s *store.Store, githubToken string) *Authorizer {
	return &Authorizer{
		store:       s,
		githubToken: githubToken,
		httpClient:  &http.Client{},
	}
}


func (a *Authorizer) CanActOnRepo(ctx context.Context, slackUserID, repoOwner, repoName string) (allowed bool, reason string, err error) {
	githubLogin, err := a.store.GetGitHubLogin(ctx, slackUserID)
	if err != nil {
		
		return false, "no linked GitHub account — link your account first", nil
	}

	
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
	
		return PermissionNone, nil
	}

	var parsed collaboratorPermissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return PermissionNone, fmt.Errorf("decode github response: %w", err)
	}

	return PermissionLevel(parsed.Permission), nil
}
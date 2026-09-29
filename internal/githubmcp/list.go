
package githubmcp

import (
	"context"
	"time"
)

type listPullRequestsArgs struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	State   string `json:"state"` 
	PerPage int    `json:"perPage,omitempty"`
}


type PullRequestSummary struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	AuthorLogin string    `json:"user"`
	CreatedAt   time.Time `json:"created_at"`
	RepoOwner   string    `json:"-"`
	RepoName    string    `json:"-"`
}


func (c *Client) ListOpenPullRequests(ctx context.Context, owner, repo string) ([]PullRequestSummary, error) {
	args := listPullRequestsArgs{
		Owner:   owner,
		Repo:    repo,
		State:   "open",
		PerPage: 100,
	}
	var result struct {
		PullRequests []PullRequestSummary `json:"pull_requests"`
	}
	if err := c.CallTool(ctx, "list_pull_requests", args, &result); err != nil {
		return nil, err
	}
	for i := range result.PullRequests {
		result.PullRequests[i].RepoOwner = owner
		result.PullRequests[i].RepoName = repo
	}
	return result.PullRequests, nil
}
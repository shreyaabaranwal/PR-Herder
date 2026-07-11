// Package githubmcp — list.go adds repo-wide PR listing, used by the
// Layer 8 scheduler for stale-PR detection. UNVERIFIED tool name/shape:
// unlike pull_request_read (confirmed against the live MCP server, see
// reads.go), this tool name is a best guess based on common MCP server
// naming conventions and needs live verification before trusting it in
// the scheduler -- write a small test harness (same pattern as
// cmd/testllm) hitting this against a real repo before wiring it in.
package githubmcp

import (
	"context"
	"time"
)

type listPullRequestsArgs struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	State   string `json:"state"` // "open" | "closed" | "all"
	PerPage int    `json:"perPage,omitempty"`
}

// PullRequestSummary is the minimal shape needed for stale-PR digest —
// deliberately smaller than PullRequestDetails since the digest doesn't
// need diff stats or CI status, just enough to render "this PR has been
// open N days."
type PullRequestSummary struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	AuthorLogin string    `json:"user"`
	CreatedAt   time.Time `json:"created_at"`
	RepoOwner   string    `json:"-"` // filled in by caller, not from response
	RepoName    string    `json:"-"`
}

// ListOpenPullRequests fetches all open PRs for a repo. TODO: verify the
// real tool name -- try "list_pull_requests" first; if the MCP server
// rejects it, check available tools via whatever discovery mechanism you
// used originally for pull_request_read.
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
package githubmcp

import "context"

// reviewWriteArgs mirrors the real "pull_request_review_write" MCP
// tool's schema (verified 2026-07-06). Like pull_request_read, this is
// method-based: "create" with an "event" submits the review directly;
// omitting "event" creates a pending review instead. PR Herder always
// submits directly (event set), since Layer 4's authz gate already
// confirmed the actor before this is called — no reason to leave a
// pending, unsubmitted review hanging.
type reviewWriteArgs struct {
	Method     string `json:"method"`
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pullNumber"`
	Event      string `json:"event,omitempty"` // "APPROVE", "REQUEST_CHANGES", "COMMENT"
	Body       string `json:"body,omitempty"`
}

// SubmitReview submits a pull request review (approve, request changes,
// or comment) via pull_request_review_write(method=create, event=...).
// event must be one of GitHub's review event constants: "APPROVE",
// "REQUEST_CHANGES", "COMMENT".
func (c *Client) SubmitReview(ctx context.Context, owner, repo string, prNumber int, event, body string) error {
	args := reviewWriteArgs{
		Method:     "create",
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
		Event:      event,
		Body:       body,
	}
	return c.CallTool(ctx, "pull_request_review_write", args, nil)
}

// issueWriteArgs mirrors the real "issue_write" MCP tool's schema.
// GitHub represents a PR's labels through its underlying Issue object —
// there is no separate "add_labels_to_pr" tool; issue_write with
// method=update and issue_number set to the PR's number is the correct
// call. This is a real GitHub API quirk, not an MCP-specific one — PRs
// and Issues share the same labels/comments substrate on GitHub itself.
type issueWriteArgs struct {
	Method      string   `json:"method"`
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
	IssueNumber int      `json:"issue_number"`
	Labels      []string `json:"labels,omitempty"`
}

// AddLabels adds labels to a PR (via the underlying issue) using
// issue_write(method=update).
func (c *Client) AddLabels(ctx context.Context, owner, repo string, prNumber int, labels []string) error {
	args := issueWriteArgs{
		Method:      "update",
		Owner:       owner,
		Repo:        repo,
		IssueNumber: prNumber,
		Labels:      labels,
	}
	return c.CallTool(ctx, "issue_write", args, nil)
}

// requestReviewersArgs mirrors "update_pull_request" — reviewer requests
// go through the general PR-update tool, not a dedicated one.
type requestReviewersArgs struct {
	Owner      string   `json:"owner"`
	Repo       string   `json:"repo"`
	PullNumber int      `json:"pullNumber"`
	Reviewers  []string `json:"reviewers"`
}

// RequestReviewers requests reviews from the given GitHub usernames via
// update_pull_request.
func (c *Client) RequestReviewers(ctx context.Context, owner, repo string, prNumber int, reviewers []string) error {
	args := requestReviewersArgs{
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
		Reviewers:  reviewers,
	}
	return c.CallTool(ctx, "update_pull_request", args, nil)
}
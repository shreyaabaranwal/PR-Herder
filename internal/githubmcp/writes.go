package githubmcp

import "context"

type reviewWriteArgs struct {
	Method     string `json:"method"`
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pullNumber"`
	Event      string `json:"event,omitempty"` 
	Body       string `json:"body,omitempty"`
}


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


type issueWriteArgs struct {
	Method      string   `json:"method"`
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
	IssueNumber int      `json:"issue_number"`
	Labels      []string `json:"labels,omitempty"`
}


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

type requestReviewersArgs struct {
	Owner      string   `json:"owner"`
	Repo       string   `json:"repo"`
	PullNumber int      `json:"pullNumber"`
	Reviewers  []string `json:"reviewers"`
}


func (c *Client) RequestReviewers(ctx context.Context, owner, repo string, prNumber int, reviewers []string) error {
	args := requestReviewersArgs{
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
		Reviewers:  reviewers,
	}
	return c.CallTool(ctx, "update_pull_request", args, nil)
}
package githubmcp

import (
	"context"
	"fmt"
)

type Executor struct {
	client *Client
}

func NewExecutor(client *Client) *Executor {
	return &Executor{client: client}
}


func (e *Executor) Execute(ctx context.Context, action, repoOwner, repoName string, prNumber int) error {
	switch action {
	case "approve":
		return e.client.SubmitReview(ctx, repoOwner, repoName, prNumber, "APPROVE", "Approved via PR Herder")
	case "request_changes":
		return e.client.SubmitReview(ctx, repoOwner, repoName, prNumber, "REQUEST_CHANGES", "Changes requested via PR Herder")
	default:
		return fmt.Errorf("unknown action: %q", action)
	}
}

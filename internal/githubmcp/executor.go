package githubmcp

import (
	"context"
	"fmt"
)

// Executor implements slackui.ActionExecutor using real MCP tool calls.
// This is what replaces slackui.StubActionExecutor once Layer 4's authz
// gate has already approved the action — Executor itself does no
// authorization, trusting that its caller (interactions.go) only
// invokes it after CanActOnRepo returned true.
type Executor struct {
	client *Client
}

func NewExecutor(client *Client) *Executor {
	return &Executor{client: client}
}

// Execute maps a Slack button's action_id to the corresponding MCP
// write call. action_id values ("approve", "request_changes",
// "request_reviewers") are PR Herder's own naming (set when blocks.go
// builds button values, a Layer 4/5 follow-up) — not GitHub or MCP
// terminology.
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

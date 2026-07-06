package slackui

import (
	"context"
	"log/slog"
)

// StubActionExecutor is a placeholder ActionExecutor used until Layer 5
// (GitHub MCP client) exists. It logs the action instead of calling
// GitHub, so Layer 4's authz + interaction-handling flow can be fully
// wired and tested end-to-end before Layer 5 is built. Delete this file
// once a real githubmcp.Executor implements the same interface.
type StubActionExecutor struct {
	log *slog.Logger
}

func NewStubActionExecutor(log *slog.Logger) *StubActionExecutor {
	return &StubActionExecutor{log: log}
}

func (s *StubActionExecutor) Execute(ctx context.Context, action, repoOwner, repoName string, prNumber int) error {
	s.log.Info("STUB: would execute GitHub action here",
		"action", action, "repo", repoOwner+"/"+repoName, "pr", prNumber)
	return nil
}

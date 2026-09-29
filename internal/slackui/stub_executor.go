package slackui

import (
	"context"
	"log/slog"
)

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

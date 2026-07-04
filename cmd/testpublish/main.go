// Command testpublish is a throwaway manual-test harness for Layer 3.
// It builds a fake triage.Result and posts it to Slack, so we can verify
// blocks.go + publisher.go work end-to-end before wiring them into the
// real webhook flow. Not meant to be a permanent part of the codebase —
// delete once Layer 3 is confirmed working, or once Layer 4 supersedes it.
package main

import (
	"context"
	"log"

	"github.com/shreyaabaranwal/pr-herder/internal/config"
	"github.com/shreyaabaranwal/pr-herder/internal/domain"
	"github.com/shreyaabaranwal/pr-herder/internal/slackui"
	"github.com/shreyaabaranwal/pr-herder/internal/triage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	pr := domain.PullRequest{
		Number:       1423,
		Title:        "Add rate-limiting to auth middleware",
		Additions:    120,
		Deletions:    15,
		ChangedFiles: []string{"internal/auth/middleware.go"},
		Association:  domain.AssocFirstTimer,
	}

	engine := triage.NewEngine()
	result := engine.Triage(pr)

	publisher := slackui.NewPublisher(cfg.SlackBotToken, cfg.SlackDefaultChan)
	if err := publisher.PublishTriageCard(context.Background(), result); err != nil {
		log.Fatalf("publish failed: %v", err)
	}

	log.Println("published successfully — check Slack!")
}

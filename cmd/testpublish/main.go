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
		RepoOwner:    "shreyaabaranwal",
		RepoName:     "PR-Herder",
		Number:       1,
		Title:        "test: dummy file for Layer 4/5 end-to-end verification",
		Additions:    1,
		Deletions:    0,
		ChangedFiles: []string{"TEST_LAYER5.md"},
		Association:  domain.AssocOwner,
	}

	engine := triage.NewEngine()
	result := engine.Triage(pr)

	publisher := slackui.NewPublisher(cfg.SlackBotToken, cfg.SlackDefaultChan)
	if err := publisher.PublishTriageCard(context.Background(), result); err != nil {
		log.Fatalf("publish failed: %v", err)
	}

	log.Println("published successfully — check Slack!")
}

package main

import (
	"context"
	"log"

	"github.com/shreyaabaranwal/pr-herder/internal/config"
	"github.com/shreyaabaranwal/pr-herder/internal/llm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	client := llm.NewClient(cfg.GeminiAPIKey, "gemini-2.0-flash")
	summarizer := llm.NewSummarizer(client)

	input := llm.PRSummaryInput{
		Title:        "Refactor auth middleware and add rate limiting across 12 files",
		Body:         "This is a large cleanup touching multiple unrelated subsystems.",
		Additions:    450,
		Deletions:    120,
		ChangedFiles: []string{"internal/auth/middleware.go", "internal/ratelimit/limiter.go", "cmd/server/main.go"},
	}

	summary, err := summarizer.Summarize(context.Background(), input)
	if err != nil {
		log.Fatalf("summarize failed: %v", err)
	}

	log.Println("SUMMARY:", summary)
}

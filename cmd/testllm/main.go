
package main

import (
	"context"
	"log"

	"github.com/shreyaabaranwal/pr-herder/internal/llm"
)

func main() {
	ollamaClient := llm.NewOllamaClient("http://localhost:11434", "llama3.2:3b")
	summarizer := llm.NewSummarizer(ollamaClient)

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
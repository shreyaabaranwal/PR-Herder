package main

import (
	"context"
	"fmt"
	"log"

	"github.com/shreyaabaranwal/pr-herder/internal/config"
	"github.com/shreyaabaranwal/pr-herder/internal/githubmcp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	client := githubmcp.NewClient(cfg.GitHubMCPURL, cfg.GitHubReadToken)

	prs, err := client.ListOpenPullRequests(context.Background(), "shreyaabaranwal", "PR-Herder")
	if err != nil {
		log.Fatalf("ListOpenPullRequests failed: %v", err)
	}
	fmt.Printf("Found %d open PRs:\n", len(prs))
	for _, pr := range prs {
		fmt.Printf("  #%d %s (created %s)\n", pr.Number, pr.Title, pr.CreatedAt)
	}
}

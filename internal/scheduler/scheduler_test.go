package scheduler

import (
	"testing"
	"time"

	"github.com/shreyaabaranwal/pr-herder/internal/githubmcp"
)

func TestFilterStale(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)

	prs := []githubmcp.PullRequestSummary{
		{
			Number:      1,
			Title:       "Fresh PR, should not be stale",
			AuthorLogin: "alice",
			CreatedAt:   now.Add(-2 * 24 * time.Hour), // 2 days old
			RepoOwner:   "shreyaabaranwal",
			RepoName:    "PR-Herder",
		},
		{
			Number:      2,
			Title:       "Old PR, should be stale",
			AuthorLogin: "bob",
			CreatedAt:   now.Add(-10 * 24 * time.Hour), // 10 days old
			RepoOwner:   "shreyaabaranwal",
			RepoName:    "PR-Herder",
		},
		{
			Number:      3,
			Title:       "Exactly at threshold, should be stale",
			AuthorLogin: "carol",
			CreatedAt:   now.Add(-7 * 24 * time.Hour), // exactly 7 days
			RepoOwner:   "shreyaabaranwal",
			RepoName:    "PR-Herder",
		},
	}

	stale := FilterStale(prs, now, 7)

	if len(stale) != 2 {
		t.Fatalf("expected 2 stale PRs, got %d: %+v", len(stale), stale)
	}

	if stale[0].Number != 2 || stale[0].DaysOpen != 10 {
		t.Errorf("expected PR #2 with 10 days open, got #%d with %d days", stale[0].Number, stale[0].DaysOpen)
	}
	if stale[1].Number != 3 || stale[1].DaysOpen != 7 {
		t.Errorf("expected PR #3 with 7 days open, got #%d with %d days", stale[1].Number, stale[1].DaysOpen)
	}
}

func TestFilterStale_EmptyInput(t *testing.T) {
	stale := FilterStale(nil, time.Now(), 7)
	if stale != nil {
		t.Errorf("expected nil for empty input, got %+v", stale)
	}
}

func TestBuildDigestBlocks_NoStalePRs(t *testing.T) {
	blocks := BuildDigestBlocks(nil)
	if blocks != nil {
		t.Errorf("expected nil blocks when no stale PRs, to avoid spamming an empty digest daily, got %+v", blocks)
	}
}

func TestBuildDigestBlocks_WithStalePRs(t *testing.T) {
	stale := []StalePR{
		{RepoOwner: "shreyaabaranwal", RepoName: "PR-Herder", Number: 2, Title: "Old PR", AuthorLogin: "bob", DaysOpen: 10},
	}
	blocks := BuildDigestBlocks(stale)

	if len(blocks) != 2 { // header + 1 PR section
		t.Fatalf("expected 2 blocks (header + 1 PR), got %d", len(blocks))
	}
	if blocks[0]["type"] != "header" {
		t.Errorf("expected first block to be header, got %v", blocks[0]["type"])
	}
}
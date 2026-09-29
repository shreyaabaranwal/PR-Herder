package scheduler

import (
	"fmt"
	"time"

	"github.com/shreyaabaranwal/pr-herder/internal/githubmcp"
)

type StalePR struct {
	RepoOwner   string
	RepoName    string
	Number      int
	Title       string
	AuthorLogin string
	DaysOpen    int
}

func FilterStale(prs []githubmcp.PullRequestSummary, now time.Time, thresholdDays int) []StalePR {
	var stale []StalePR
	for _, pr := range prs {
		days := int(now.Sub(pr.CreatedAt).Hours() / 24)
		if days >= thresholdDays {
			stale = append(stale, StalePR{
				RepoOwner:   pr.RepoOwner,
				RepoName:    pr.RepoName,
				Number:      pr.Number,
				Title:       pr.Title,
				AuthorLogin: pr.AuthorLogin,
				DaysOpen:    days,
			})
		}
	}
	return stale
}

func BuildDigestBlocks(stale []StalePR) []map[string]any {
	if len(stale) == 0 {
		return nil
	}

	blocks := []map[string]any{
		{
			"type": "header",
			"text": map[string]any{
				"type": "plain_text",
				"text": fmt.Sprintf(":coffee: Stale PR Digest — %d PR(s) need attention", len(stale)),
			},
		},
	}

	for _, pr := range stale {
		text := fmt.Sprintf("*%s/%s#%d* — %s\nOpened by @%s · open %d days",
			pr.RepoOwner, pr.RepoName, pr.Number, pr.Title, pr.AuthorLogin, pr.DaysOpen)
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": text,
			},
		})
	}

	return blocks
}
package slackui

import (
	"fmt"

	"github.com/shreyaabaranwal/pr-herder/internal/triage"
)

// BuildTriageCardBlocks renders a triage.Result as a Slack Block Kit
// payload. Layer 4 added real interactivity — action buttons whose
// value encodes "owner/repo#number", matching what
// interactions.go's parseActionValue expects.
func BuildTriageCardBlocks(result triage.Result) []map[string]any {
	pr := result.PR

	header := fmt.Sprintf("#%d · %s", pr.Number, pr.Title)
	if result.PathMatch.Matched {
		header += fmt.Sprintf(" — `%s`", result.PathMatch.Label)
	}

	var bodyLines []string
	if result.PathMatch.Matched {
		bodyLines = append(bodyLines,
			fmt.Sprintf("Touches `%s` → routed to %s", result.PathMatch.Path, result.PathMatch.Team))
	}
	bodyLines = append(bodyLines, fmt.Sprintf("Size: `%s` (+%d/-%d)", result.SizeCategory, pr.Additions, pr.Deletions))
	if result.Contributor.SuggestedWelcome {
		bodyLines = append(bodyLines, "First-time contributor — welcome message auto-suggested")
	}
	if result.Ambiguous {
		bodyLines = append(bodyLines, "_Large, unrouted diff — flagged for LLM summary (Layer 7)_")
	for checkName, verdict := range result.FlakyChecks {
		if verdict.IsFlaky {
			bodyLines = append(bodyLines, fmt.Sprintf("CI failing, but on `%s`, historically flaky (safe to retry)", checkName))
		}
	}
	}

	bodyText := ""
	for i, line := range bodyLines {
		if i > 0 {
			bodyText += "\n"
		}
		bodyText += line
	}

	blocks := []map[string]any{
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": "*" + header + "*",
			},
		},
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": bodyText,
			},
		},
	}

	if len(result.Labels) > 0 {
		labelText := "Labels: "
		for i, l := range result.Labels {
			if i > 0 {
				labelText += ", "
			}
			labelText += "`" + l + "`"
		}
		blocks = append(blocks, map[string]any{
			"type": "context",
			"elements": []map[string]any{
				{"type": "mrkdwn", "text": labelText},
			},
		})
	}

	// actionValue encodes "owner/repo#number" — the exact format
	// interactions.go's parseActionValue expects. Kept as a local helper
	// here so the encoding logic lives in exactly one place per direction
	// (encode here, decode in interactions.go) rather than scattered.
	actionValue := fmt.Sprintf("%s/%s#%d", pr.RepoOwner, pr.RepoName, pr.Number)

	blocks = append(blocks, map[string]any{
		"type": "actions",
		"elements": []map[string]any{
			{
				"type": "button",
				"text": map[string]any{
					"type": "plain_text",
					"text": "Approve",
				},
				"style":     "primary",
				"action_id": "approve",
				"value":     actionValue,
			},
			{
				"type": "button",
				"text": map[string]any{
					"type": "plain_text",
					"text": "Request changes",
				},
				"style":     "danger",
				"action_id": "request_changes",
				"value":     actionValue,
			},
		},
	})

	return blocks
}
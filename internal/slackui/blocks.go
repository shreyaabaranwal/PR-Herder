// Package slackui builds Slack Block Kit payloads and posts them. It is
// the only package that knows Slack's wire format (blocks, sections,
// buttons) — triage.Result stays Slack-agnostic, and this package
// translates it at the boundary, mirroring the same anti-corruption
// pattern used for GitHub's webhook payloads in internal/ingest.
package slackui

import (
	"fmt"

	"github.com/shreyaabaranwal/pr-herder/internal/triage"
)

// BuildTriageCardBlocks renders a triage.Result as a Slack Block Kit
// payload. Layer 3 is read-only — no action buttons yet. Those arrive in
// Layer 4, gated behind the authz work.
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

	return blocks
}

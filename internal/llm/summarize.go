package llm

import (
	"context"
	"fmt"
	"strings"
)

// PRSummaryInput is the minimal PR data needed to generate a summary --
// decoupled from domain.PullRequest so this package doesn't need to
// import domain just to describe what it needs (title, size, changed
// files are all that matter for a useful summary).
type PRSummaryInput struct {
	Title        string
	Body         string
	Additions    int
	Deletions    int
	ChangedFiles []string
}

// Summarizer wraps a Client with PR-Herder-specific prompt construction.
type Summarizer struct {
	client *Client
}

func NewSummarizer(client *Client) *Summarizer {
	return &Summarizer{client: client}
}

// Summarize produces a short, factual summary of an ambiguous PR (per
// ADR 0001 -- only called when triage.Result.Ambiguous is true). The
// prompt is deliberately restrictive: no routing/label suggestions, no
// opinions on whether to merge -- just "what does this PR do," since
// that's the one thing deterministic rules couldn't confidently
// determine on their own.
func (s *Summarizer) Summarize(ctx context.Context, input PRSummaryInput) (string, error) {
	prompt := buildSummaryPrompt(input)

	text, err := s.client.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("generate summary: %w", err)
	}

	return strings.TrimSpace(text), nil
}

func buildSummaryPrompt(input PRSummaryInput) string {
	var filesList string
	if len(input.ChangedFiles) > 0 {
		filesList = strings.Join(input.ChangedFiles, ", ")
	} else {
		filesList = "(file list unavailable)"
	}

	body := input.Body
	if body == "" {
		body = "(no description provided)"
	}

	// The prompt is explicit about scope on purpose: a maintainer reading
	// this summary needs to trust it's describing the change, not
	// recommending an action. Keeping the model's output to 2-3 sentences
	// also keeps it fitting cleanly inside a Slack Block Kit section.
	return fmt.Sprintf(`You are summarizing a GitHub pull request for a maintainer who hasn't read the diff yet.

Write a short, factual summary in 2-3 sentences: what does this PR change and why (if stated). Do not recommend whether to approve, merge, or request changes. Do not suggest labels. Just describe what the PR does.

Title: %s
Description: %s
Diff size: +%d/-%d lines
Changed files: %s

Summary:`, input.Title, body, input.Additions, input.Deletions, filesList)
}

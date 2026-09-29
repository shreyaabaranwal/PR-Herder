package llm

import (
	"context"
	"fmt"
	"strings"
)


type PRSummaryInput struct {
	Title        string
	Body         string
	Additions    int
	Deletions    int
	ChangedFiles []string
}

type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}


type Summarizer struct {
	client Generator
}

func NewSummarizer(client Generator) *Summarizer {
	return &Summarizer{client: client}
}


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


	return fmt.Sprintf(`You are summarizing a GitHub pull request for a maintainer who hasn't read the diff yet.

Write a short, factual summary in 2-3 sentences: what does this PR change and why (if stated). Do not recommend whether to approve, merge, or request changes. Do not suggest labels. Just describe what the PR does.

Title: %s
Description: %s
Diff size: +%d/-%d lines
Changed files: %s

Summary:`, input.Title, body, input.Additions, input.Deletions, filesList)
}

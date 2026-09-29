package slackui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shreyaabaranwal/pr-herder/internal/triage"
)

const slackPostMessageURL = "https://slack.com/api/chat.postMessage"


type Publisher struct {
	botToken       string
	defaultChannel string
	httpClient     *http.Client
}

func NewPublisher(botToken, defaultChannel string) *Publisher {
	return &Publisher{
		botToken:       botToken,
		defaultChannel: defaultChannel,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (p *Publisher) PublishTriageCard(ctx context.Context, result triage.Result) error {
	blocks := BuildTriageCardBlocks(result)
	text := fmt.Sprintf("PR #%d triaged: %s", result.PR.Number, result.PR.Title)
	return p.send(ctx, blocks, text)
}


func (p *Publisher) PublishBlocks(ctx context.Context, blocks []map[string]any) error {
	return p.send(ctx, blocks, "PR Herder digest")
}


func (p *Publisher) send(ctx context.Context, blocks []map[string]any, fallbackText string) error {
	payload := map[string]any{
		"channel": p.defaultChannel,
		"blocks":  blocks,
		"text":    fallbackText,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, slackPostMessageURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+p.botToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call slack api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read slack response: %w", err)
	}

	var sr slackResponse
	if err := json.Unmarshal(respBody, &sr); err != nil {
		return fmt.Errorf("parse slack response: %w", err)
	}
	if !sr.OK {
		return fmt.Errorf("slack api error: %s", sr.Error)
	}

	return nil
}

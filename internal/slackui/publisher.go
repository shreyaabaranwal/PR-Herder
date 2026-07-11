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

// Publisher posts triage results to a Slack channel via chat.postMessage.
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

// slackResponse is the minimal shape of chat.postMessage's response.
// Slack's API always returns HTTP 200 even on failure — the real
// success/failure signal is the "ok" field, not the status code.
type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// PublishTriageCard posts a triage.Result as a Block Kit message to the
// configured default channel.
func (p *Publisher) PublishTriageCard(ctx context.Context, result triage.Result) error {
	blocks := BuildTriageCardBlocks(result)
	text := fmt.Sprintf("PR #%d triaged: %s", result.PR.Number, result.PR.Title)
	return p.send(ctx, blocks, text)
}

// PublishBlocks posts arbitrary pre-built Block Kit blocks to the
// configured default channel -- used by Layer 8's scheduler for the
// stale-PR digest, which isn't shaped like a triage.Result. Satisfies
// scheduler.Publisher.
func (p *Publisher) PublishBlocks(ctx context.Context, blocks []map[string]any) error {
	return p.send(ctx, blocks, "PR Herder digest")
}

// send is the shared chat.postMessage call -- both PublishTriageCard and
// PublishBlocks post to the same channel via the same auth, they only
// differ in what blocks/fallback text they send.
func (p *Publisher) send(ctx context.Context, blocks []map[string]any, fallbackText string) error {
	payload := map[string]any{
		"channel": p.defaultChannel,
		"blocks":  blocks,
		"text":    fallbackText, // shown in notifications/unfurls where blocks don't render
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

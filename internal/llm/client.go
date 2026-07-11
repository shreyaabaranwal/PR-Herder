// Package llm wraps Google's Gemini API for Layer 7 summary generation.
// Per ADR 0001, this is only ever called for PRs the deterministic
// triage engine marked Ambiguous -- never for routing decisions, only
// to produce a factual, maintainer-facing summary.
package llm

import (
	"time"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient builds a Gemini client. model is e.g. "gemini-2.0-flash" --
// passed explicitly rather than hardcoded so callers can pick a
// free-tier-eligible model without a code change.
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []part `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *geminiError `json:"error,omitempty"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Generate sends a single-turn prompt to Gemini and returns the model's
// text response. No conversation history, no system prompt separation --
// Layer 7's use case (summarize this PR) is single-shot, so the simplest
// wire shape that works is used rather than a fuller chat API.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := generateContentRequest{
		Contents: []content{
			{Parts: []part{{Text: prompt}}},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf(geminiEndpoint, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Header-based auth (x-goog-api-key) rather than the ?key= query
	// param -- more reliable across API key formats (older AIzaSy...
	// keys and newer AQ.* project-scoped keys both work via header;
	// the query-param path returned a 401 "expected OAuth2 token" for
	// the newer key format during Layer 7 development).
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call gemini api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read gemini response: %w", err)
	}

	var parsed generateContentResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decode gemini response: %w (body: %s)", err, string(respBody))
	}

	if parsed.Error != nil {
		return "", fmt.Errorf("gemini api error (code %d): %s", parsed.Error.Code, parsed.Error.Message)
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no candidates (body: %s)", string(respBody))
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

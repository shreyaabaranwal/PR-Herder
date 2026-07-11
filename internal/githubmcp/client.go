// Package githubmcp wraps GitHub's remote MCP server
// (https://api.githubcopilot.com/mcp/) — GitHub's official, hosted
// Model Context Protocol endpoint. All requests are JSON-RPC 2.0 POSTs.
package githubmcp

import (
	"time"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// CallTool invokes a named MCP tool and decodes the result into dest.
func (c *Client) CallTool(ctx context.Context, toolName string, args any, dest any) error {
	reqBody := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: toolCallParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal mcp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build mcp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call mcp server: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read mcp response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mcp server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// GitHub's MCP server uses the Streamable HTTP transport and replies
	// with an SSE envelope ("event: message\ndata: {...}") rather than a
	// bare JSON body. Strip the SSE framing before parsing JSON.
	jsonPayload := extractSSEData(respBody)

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(jsonPayload, &rpcResp); err != nil {
		return fmt.Errorf("decode mcp response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("mcp tool error (code %d): %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if dest != nil {
		if err := json.Unmarshal(rpcResp.Result, dest); err != nil {
			return fmt.Errorf("decode tool result: %w", err)
		}
	}

	return nil
}

// extractSSEData strips Server-Sent Events framing from a response body,
// returning just the JSON payload from the last "data:" line. If the
// body isn't SSE-framed, it's returned unchanged.
func extractSSEData(body []byte) []byte {
	lines := bytes.Split(body, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if bytes.HasPrefix(line, []byte("data: ")) {
			return bytes.TrimPrefix(line, []byte("data: "))
		}
	}
	return body
}

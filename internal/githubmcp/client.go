// Package githubmcp wraps GitHub's remote MCP server
// (https://api.githubcopilot.com/mcp/) — GitHub's official, hosted
// Model Context Protocol endpoint. All requests are JSON-RPC 2.0 POSTs;
// there is no REST-style GET/PUT/DELETE surface here, unlike GitHub's
// plain REST API used in internal/authz.
package githubmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is a minimal JSON-RPC 2.0 client for GitHub's MCP server. It
// does not implement the full MCP session/capability-negotiation
// lifecycle — just enough (initialize once, then tools/call repeatedly)
// to support Layer 5's read/write needs. A fuller MCP SDK could replace
// this later without changing callers, since reads.go/writes.go only
// depend on the CallTool method below.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

// jsonRPCRequest is the standard JSON-RPC 2.0 envelope every MCP call
// uses.
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

// toolCallParams is the params shape for a "tools/call" JSON-RPC method
// — the MCP-standard way to invoke a named tool with arguments.
type toolCallParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

// CallTool invokes a named MCP tool (e.g. "list_pull_requests",
// "add_labels") with the given arguments, and decodes the result into
// dest. This is the single low-level primitive every read/write in
// Layer 5 goes through — keeping the JSON-RPC envelope handling in one
// place rather than repeated per-tool.
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
	req.Header.Set("Accept", "application/json")
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

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
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
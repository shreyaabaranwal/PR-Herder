package githubmcp

import "context"

// pullRequestReadArgs mirrors the real "pull_request_read" MCP tool's
// input schema (verified against the live server on 2026-07-06 — see
// docs/adr for MCP tool discovery notes). This tool is method-based：
// one tool, many "method" values (get, get_diff, get_files,
// get_check_runs, etc.) rather than one tool per action.
type pullRequestReadArgs struct {
	Method     string `json:"method"`
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pullNumber"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"perPage,omitempty"`
}

// PullRequestDetails is githubmcp's view of "pull_request_read" with
// method=get. Kept separate from domain.PullRequest — same
// anti-corruption-layer principle as ingest/extract.go.
type PullRequestDetails struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Body        string `json:"body"`
	AuthorLogin string `json:"user"`
	HeadSHA     string `json:"head_sha"`
	BaseBranch  string `json:"base_branch"`
	HeadBranch  string `json:"head_branch"`
}

// GetPullRequest fetches core PR details via pull_request_read(method=get).
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, prNumber int) (*PullRequestDetails, error) {
	args := pullRequestReadArgs{
		Method:     "get",
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
	}
	var result PullRequestDetails
	if err := c.CallTool(ctx, "pull_request_read", args, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ChangedFile is a single file entry from pull_request_read(method=get_files).
type ChangedFile struct {
	Filename  string `json:"filename"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// GetPullRequestFiles fetches per-file diff stats via
// pull_request_read(method=get_files) — this feeds triage.Engine.Triage
// (ClassifySize, MatchSensitivePaths) once ingest is updated to enrich
// domain.PullRequest with this data (a Layer 5 follow-up: webhook
// payloads alone don't carry per-file add/delete counts).
func (c *Client) GetPullRequestFiles(ctx context.Context, owner, repo string, prNumber int) ([]ChangedFile, error) {
	args := pullRequestReadArgs{
		Method:     "get_files",
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
		PerPage:    100,
	}
	var result struct {
		Files []ChangedFile `json:"files"`
	}
	if err := c.CallTool(ctx, "pull_request_read", args, &result); err != nil {
		return nil, err
	}
	return result.Files, nil
}

// CheckRun is a single CI check result, used by the flaky-CI classifier
// (internal/triage/flaky.go, Layer 6).
type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// GetCheckRuns fetches CI check runs via
// pull_request_read(method=get_check_runs) — note this is a method on
// the same "pull_request_read" tool, NOT a separate MCP tool as
// originally assumed before schema discovery.
func (c *Client) GetCheckRuns(ctx context.Context, owner, repo string, prNumber int) ([]CheckRun, error) {
	args := pullRequestReadArgs{
		Method:     "get_check_runs",
		Owner:      owner,
		Repo:       repo,
		PullNumber: prNumber,
	}
	var result struct {
		CheckRuns []CheckRun `json:"check_runs"`
	}
	if err := c.CallTool(ctx, "pull_request_read", args, &result); err != nil {
		return nil, err
	}
	return result.CheckRuns, nil
}
package githubmcp

import "context"


type pullRequestReadArgs struct {
	Method     string `json:"method"`
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pullNumber"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"perPage,omitempty"`
}


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

type ChangedFile struct {
	Filename  string `json:"filename"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

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

type CheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}


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
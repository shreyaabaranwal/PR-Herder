// Package triage implements PR Herder's deterministic classification
// rules (ADR 0001: deterministic rules run before any LLM call). Every
// function here must be a pure function over domain.PullRequest — no
// network calls, no LLM calls, no database reads. This is what makes
// triage fast, free, and unit-testable without mocks.
package triage

import "strings"

// SensitivePathRule maps a glob-style path prefix to the team that should
// review changes touching it. This is deliberately simple prefix
// matching, not full glob support — CODEOWNERS-equivalent behavior
// without pulling in a glob library for Layer 2. Layer 5 (GitHub MCP)
// can later read the repo's actual CODEOWNERS file and feed rules here.
type SensitivePathRule struct {
	PathPrefix string
	Team       string
	Label      string
}

// DefaultSensitivePathRules is a starting set; real deployments would
// load this from repo config (a future layer), not hardcode it.
var DefaultSensitivePathRules = []SensitivePathRule{
	{PathPrefix: "internal/auth/", Team: "@security-team", Label: "needs-security-review"},
	{PathPrefix: "internal/authz/", Team: "@security-team", Label: "needs-security-review"},
	{PathPrefix: ".github/workflows/", Team: "@platform-team", Label: "needs-platform-review"},
	{PathPrefix: "migrations/", Team: "@data-team", Label: "needs-data-review"},
}

// PathMatch describes which sensitive-path rule fired for a given PR, if
// any. A PR can only match the FIRST rule whose prefix it touches, in
// rule-declaration order — this keeps routing deterministic (one PR,
// one primary reviewer team) rather than fanning a PR out to every team
// whose path it happens to touch.
type PathMatch struct {
	Matched bool
	Team    string
	Label   string
	Path    string // which changed file triggered the match, for the Slack card
}

// MatchSensitivePaths checks a PR's changed files against the given rules
// and returns the first match. Returns Matched=false if no changed file
// touches any sensitive prefix.
func MatchSensitivePaths(changedFiles []string, rules []SensitivePathRule) PathMatch {
	for _, rule := range rules {
		for _, f := range changedFiles {
			if strings.HasPrefix(f, rule.PathPrefix) {
				return PathMatch{
					Matched: true,
					Team:    rule.Team,
					Label:   rule.Label,
					Path:    f,
				}
			}
		}
	}
	return PathMatch{Matched: false}
}
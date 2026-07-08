package triage

import "github.com/shreyaabaranwal/pr-herder/internal/domain"

// Result is the full triage output for a single PR — everything the
// Slack card (Layer 3) needs to render, and everything downstream
// actions (Layer 4/5) need to route correctly.
type Result struct {
	PR domain.PullRequest

	SizeCategory SizeCategory
	PathMatch    PathMatch
	Contributor  ContributorSignal

	// Ambiguous is true when deterministic rules couldn't confidently
	// classify this PR (e.g. large diff, no sensitive-path match). Only
	// PRs where this is true are eligible for the Layer 7 LLM summary —
	// see ADR 0001. Nothing here ever flips this to trigger an LLM call
	// on the majority of PRs.
	Ambiguous bool

	// Labels are the suggested GitHub labels this triage run produced.
	// "Suggested" — Layer 4/5 still gate anything that actually posts
	// back to GitHub behind maintainer confirmation for security-relevant
	// labels.
	Labels []string

	// FlakyChecks holds the flaky-classification verdict for any CI
	// checks on this PR that the caller looked up (Layer 6). This is a
	// map, not a single verdict, because a PR can have multiple check
	// runs (lint, test, build) each with independent flaky history.
	// Populated by the caller (a Layer 6 follow-up to ingest/worker.go)
	// after fetching check-run history -- Triage() itself stays a pure,
	// network-free function per ADR 0001, so this is never set inside
	// Triage() itself.
	FlakyChecks map[string]FlakyVerdict
}

// Engine runs the deterministic rule layers, in order, over a PR. It
// holds configuration (which sensitive-path rules apply) so tests can
// inject a small fixed rule set instead of depending on the package-level
// default.
type Engine struct {
	SensitivePathRules []SensitivePathRule
}

// NewEngine builds an Engine with the default sensitive-path rules. Tests
// that need deterministic, minimal rule sets should construct an Engine
// directly instead, e.g. Engine{SensitivePathRules: []SensitivePathRule{...}}.
func NewEngine() *Engine {
	return &Engine{SensitivePathRules: DefaultSensitivePathRules}
}

// Triage runs all Layer 2 deterministic checks over a single PR and
// returns a Result. This function must stay free of network/LLM calls —
// that boundary is the entire point of ADR 0001. If a future rule
// genuinely needs external data (e.g. real CODEOWNERS from GitHub), it
// should be fetched by the caller and passed in as data, not fetched
// here.
func (e *Engine) Triage(pr domain.PullRequest) Result {
	pathMatch := MatchSensitivePaths(pr.ChangedFiles, e.SensitivePathRules)
	size := ClassifySize(pr.Additions, pr.Deletions)
	contributor := ClassifyContributor(pr.Association)

	var labels []string
	if pathMatch.Matched {
		labels = append(labels, pathMatch.Label)
	}
	labels = append(labels, "size/"+string(size))

	// Ambiguous: a PR is confidently classified if it either matches a
	// sensitive path (clear routing) or is small enough that risk is low
	// regardless of path (S/M). Large, unrouted diffs are exactly the
	// case ADR 0001 calls out for LLM summary in Layer 7 — everything
	// else here stays deterministic and free.
	ambiguous := !pathMatch.Matched && (size == SizeL || size == SizeXL)

	return Result{
		PR:           pr,
		SizeCategory: size,
		PathMatch:    pathMatch,
		Contributor:  contributor,
		Ambiguous:    ambiguous,
		Labels:       labels,
	}
}
package triage

// CheckRunObservation is one historical record of a check's outcome —
// what the flaky classifier operates on. This is intentionally decoupled
// from store.CheckRunHistory (the Postgres row shape) and from
// githubmcp.CheckRun (the MCP API response shape) — the classifier only
// needs conclusion + commit, nothing about how that data was fetched or
// stored.
type CheckRunObservation struct {
	CommitSHA  string
	Conclusion string // "success", "failure", "neutral", "cancelled", etc.
}

// FlakyVerdict is the classifier's output for one check.
type FlakyVerdict struct {
	IsFlaky bool
	// FailThenPassRate is included even when not flaky, so callers
	// (Slack card rendering) can show "improving" trends without a
	// second query -- keeps the classifier's one pass over history
	// useful for more than a single yes/no answer.
	FailThenPassRate float64
	SampleSize       int
}

// minSampleSize is the minimum number of distinct commits a check needs
// history for before we're willing to call it flaky vs. just unlucky.
// Below this, "flaky" is a guess, not a signal -- ADR 0001's whole point
// is that deterministic classifications should be ones we can defend,
// not superstition from 2 data points.
const minSampleSize = 3

// flakyThreshold is the fail-then-pass rate above which a check is
// classified flaky. Chosen conservatively (over half the time a failure
// on this check got overturned by a later run on the same commit) so a
// genuinely broken check doesn't get miscategorized as "safe to ignore."
const flakyThreshold = 0.5

// ClassifyFlaky examines a check's history (all observations for one
// check_name, ordered oldest-to-newest is NOT required -- see grouping
// below) and determines whether it exhibits a fail-then-pass-on-retry
// pattern.
//
// Algorithm: group observations by commit SHA (a "retry" is multiple
// observations for the SAME commit -- a new commit is a fresh attempt,
// not a retry). For each commit with 2+ observations, check whether the
// first-seen result was a failure and a later one was success. The
// fail-then-pass rate is (commits showing that pattern) / (commits with
// any failure at all) -- commits that only ever passed aren't part of
// the "was this flaky" question.
func ClassifyFlaky(observations []CheckRunObservation) FlakyVerdict {
	// Group by commit, preserving first-seen order per commit -- this
	// assumes the caller provides observations in chronological order
	// (oldest first), which is how store.go's query is designed to
	// return them (ORDER BY observed_at ASC).
	type commitRun struct {
		results []string
	}
	byCommit := make(map[string]*commitRun)
	var commitOrder []string

	for _, obs := range observations {
		run, exists := byCommit[obs.CommitSHA]
		if !exists {
			run = &commitRun{}
			byCommit[obs.CommitSHA] = run
			commitOrder = append(commitOrder, obs.CommitSHA)
		}
		run.results = append(run.results, obs.Conclusion)
	}

	commitsWithFailure := 0
	commitsFailThenPass := 0

	for _, sha := range commitOrder {
		run := byCommit[sha]
		hadFailure := false
		hadFailureThenSuccess := false
		sawFailure := false

		for _, result := range run.results {
			if result == "failure" {
				hadFailure = true
				sawFailure = true
			} else if result == "success" && sawFailure {
				hadFailureThenSuccess = true
			}
		}

		if hadFailure {
			commitsWithFailure++
		}
		if hadFailureThenSuccess {
			commitsFailThenPass++
		}
	}

	sampleSize := len(commitOrder)

	if commitsWithFailure == 0 {
		// No failures observed at all -- nothing to classify as flaky
		// vs. not; this check has simply never failed in our history.
		return FlakyVerdict{IsFlaky: false, FailThenPassRate: 0, SampleSize: sampleSize}
	}

	rate := float64(commitsFailThenPass) / float64(commitsWithFailure)

	isFlaky := sampleSize >= minSampleSize && rate >= flakyThreshold

	return FlakyVerdict{
		IsFlaky:          isFlaky,
		FailThenPassRate: rate,
		SampleSize:       sampleSize,
	}
}

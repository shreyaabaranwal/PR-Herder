// Package ingest's worker is the async processor that turns stored
// webhook_events rows into triaged Slack cards. It is deliberately
// separate from the HTTP handler in webhook.go: the handler's only job
// is fast-ack (verify, store, return), so a slow or buggy triage run can
// never cause GitHub to see a timeout and start retry-storming us.
package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"
     
	"github.com/shreyaabaranwal/pr-herder/internal/githubmcp"
	"github.com/shreyaabaranwal/pr-herder/internal/domain"
	"github.com/shreyaabaranwal/pr-herder/internal/llm"
	"github.com/shreyaabaranwal/pr-herder/internal/slackui"
	"github.com/shreyaabaranwal/pr-herder/internal/store"
	"github.com/shreyaabaranwal/pr-herder/internal/triage"
)

// Worker polls webhook_events for unprocessed rows and triages them.
type Worker struct {
	store      *store.Store
	engine     *triage.Engine
	publisher  *slackui.Publisher
	github *githubmcp.Client
	summarizer *llm.Summarizer
	log        *slog.Logger
}

// summarizer may be nil -- if so, ambiguous PRs are published without a
// summary rather than the worker failing. This keeps Layer 7 optional:
// the pipeline (Layers 0-6) works fully even if no LLM backend is wired
// in main.go.
func NewWorker(
	s *store.Store,
	engine *triage.Engine,
	publisher *slackui.Publisher,
	github *githubmcp.Client,
	summarizer *llm.Summarizer,
	log *slog.Logger,
) *Worker{
	return &Worker{
	store: s,
	engine: engine,
	publisher: publisher,
	github: github,
	summarizer: summarizer,
	log: log,
}
}

// Run polls forever at the given interval until ctx is cancelled. Each
// tick processes up to a bounded batch (see processBatch) rather than
// draining the whole backlog in one tick -- keeps any single tick's
// duration predictable even if a large backlog builds up.
func (w *Worker) Run(ctx context.Context, pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker stopping")
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.log.Error("batch processing failed", "err", err)
			}
		}
	}
}

const batchSize = 10

func (w *Worker) processBatch(ctx context.Context) error {
	events, err := w.store.GetUnprocessedEvents(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("fetch unprocessed events: %w", err)
	}

	for _, ev := range events {
		if err := w.processOne(ctx, ev); err != nil {
			w.log.Error("event processing failed",
				"delivery_id", ev.DeliveryID, "err", err)
			if markErr := w.store.MarkEventFailed(ctx, ev.ID, err); markErr != nil {
				w.log.Error("failed to record processing error", "err", markErr)
			}
			continue // one bad event shouldn't block the rest of the batch
		}
		if err := w.store.MarkEventProcessed(ctx, ev.ID); err != nil {
			w.log.Error("failed to mark event processed", "delivery_id", ev.DeliveryID, "err", err)
		}
	}

	return nil
}

// processOne triages a single webhook event and, if it's PR-scoped,
// publishes a triage card to Slack. Non-PR events (or events missing a
// PR number) are treated as a no-op success -- there's nothing to
// triage, not an error.
func (w *Worker) processOne(ctx context.Context, ev store.UnprocessedEvent) error {
	start := time.Now()
	defer func() {
		w.log.Info("processOne finished", "delivery_id", ev.DeliveryID, "duration_ms", time.Since(start).Milliseconds())
	}()

	if ev.PRNumber == nil {
		w.log.Info("skipping non-PR event", "delivery_id", ev.DeliveryID, "event_type", ev.EventType)
		return nil
	}

	pr, ok := buildPullRequestFromPayload(ev)
if !ok {
	w.log.Warn(
		"could not extract PR data from payload, skipping",
		"delivery_id", ev.DeliveryID,
	)
	return nil
}

// Layer 7: enrich the PR with the real changed file list before
// deterministic triage and LLM summarization.
files, err := w.github.GetPullRequestFiles(
	ctx,
	pr.RepoOwner,
	pr.RepoName,
	pr.Number,
)

if err != nil {
	w.log.Warn(
		"failed to fetch changed files from GitHub MCP",
		"repo", pr.RepoOwner+"/"+pr.RepoName,
		"pr", pr.Number,
		"err", err,
	)
} else {
	changedFiles := make([]string, 0, len(files))
	for _, f := range files {
		changedFiles = append(changedFiles, f.Filename)
	}
	pr.ChangedFiles = changedFiles
	w.log.Info(
    "loaded changed files",
    "repo", pr.RepoOwner+"/"+pr.RepoName,
    "pr", pr.Number,
    "count", len(changedFiles),
    "files", changedFiles,
)
}

result := w.engine.Triage(pr)
	// Layer 7: only ambiguous PRs get an LLM summary (ADR 0001). A
	// summarizer failure here is logged, not fatal -- the Slack card
	// still gets published without a summary rather than losing the
	// whole triage result over an LLM hiccup.
	if result.Ambiguous && w.summarizer != nil {
		var summary string
		sumErr := withRetry(ctx, 3, func() error {
			s, err := w.summarizer.Summarize(ctx, llm.PRSummaryInput{
				Title:        pr.Title,
				Body:         pr.Body,
				Additions:    pr.Additions,
				Deletions:    pr.Deletions,
				ChangedFiles: pr.ChangedFiles,
			})
			if err != nil {
				return err
			}
			summary = s
			return nil
		})
		if sumErr != nil {
			w.log.Warn("llm summary failed after retries, publishing without it",
				"delivery_id", ev.DeliveryID, "err", sumErr)
		} else {
			result.Summary = summary
		}
	}

	if err := withRetry(ctx, 3, func() error {
		return w.publisher.PublishTriageCard(ctx, result)
	}); err != nil {
		return fmt.Errorf("publish triage card after retries: %w", err)
	}

	w.log.Info("triaged and published",
		"delivery_id", ev.DeliveryID, "repo", ev.RepoOwner+"/"+ev.RepoName, "pr", *ev.PRNumber)
	return nil
}

// buildPullRequestFromPayload translates a raw GitHub webhook payload
// into domain.PullRequest -- the anti-corruption-layer boundary this
// package owns (see extract.go's doc comment for the same principle
// applied to routing info). Only pull_request-shaped payloads are
// supported; other event types return ok=false.
func buildPullRequestFromPayload(ev store.UnprocessedEvent) (domain.PullRequest, bool) {
	prData, ok := ev.RawPayload["pull_request"].(map[string]any)
	if !ok {
		return domain.PullRequest{}, false
	}

	title, _ := prData["title"].(string)
	body, _ := prData["body"].(string)
	state, _ := prData["state"].(string)

	var authorLogin string
	var association domain.AuthorAssociation
	if user, ok := prData["user"].(map[string]any); ok {
		authorLogin, _ = user["login"].(string)
	}
	if assoc, ok := prData["author_association"].(string); ok {
		association = domain.AuthorAssociation(assoc)
	}

	var additions, deletions int
	if a, ok := prData["additions"].(float64); ok {
		additions = int(a)
	}
	if d, ok := prData["deletions"].(float64); ok {
		deletions = int(d)
	}

	var headSHA, baseBranch, headBranch string
	if head, ok := prData["head"].(map[string]any); ok {
		headSHA, _ = head["sha"].(string)
		headBranch, _ = head["ref"].(string)
	}
	if base, ok := prData["base"].(map[string]any); ok {
		baseBranch, _ = base["ref"].(string)
	}

	// changed_files is NOT present in the webhook payload itself --
	// GitHub's pull_request event doesn't include the file list. This is
	// a known gap: triage rules that depend on ChangedFiles (sensitive
	// path matching) won't fire correctly until a Layer 5 follow-up
	// fetches files via githubmcp.GetPullRequestFiles before calling
	// Triage(). Tracked here rather than silently producing wrong
	// results.
	return domain.PullRequest{
		RepoOwner:    ev.RepoOwner,
		RepoName:     ev.RepoName,
		Number:       *ev.PRNumber,
		Title:        title,
		Body:         body,
		AuthorLogin:  authorLogin,
		Association:  association,
		BaseBranch:   baseBranch,
		HeadBranch:   headBranch,
		HeadSHA:      headSHA,
		Additions:    additions,
		Deletions:    deletions,
		ChangedFiles: nil, // see comment above
		State:        state,
		LastEventID:  ev.DeliveryID,
	}, true
}

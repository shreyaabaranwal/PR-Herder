// Package metrics exposes Prometheus counters and histograms for PR
// Herder's key operations. A single package-level registry (rather than
// dependency-injecting a *Metrics struct everywhere) keeps this simple --
// metrics are a cross-cutting concern read by /metrics, not business
// logic that needs per-call configuration.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// WebhookEventsReceived counts every webhook delivery stored,
	// labeled by event_type (pull_request, ping, etc.) and whether it
	// was a duplicate (already-seen delivery_id).
	WebhookEventsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prherder_webhook_events_received_total",
			Help: "Total webhook events received, labeled by event_type and duplicate status.",
		},
		[]string{"event_type", "duplicate"},
	)

	// TriageDuration measures how long a single processOne call takes,
	// end to end (file fetch + triage + optional LLM summary + Slack
	// publish). Helps spot whether slowness is typical or an outlier.
	TriageDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "prherder_triage_duration_seconds",
			Help:    "Duration of a single PR triage-and-publish cycle.",
			Buckets: prometheus.DefBuckets, // 0.005s to 10s, good default spread
		},
	)

	// LLMRequestsTotal counts every LLM summarization attempt, labeled
	// by outcome (success/failure) -- a rising failure rate here is the
	// clearest signal that Ollama/Gemini is struggling.
	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prherder_llm_requests_total",
			Help: "Total LLM summary requests, labeled by outcome.",
		},
		[]string{"outcome"},
	)

	// SlackPublishTotal counts every Slack publish attempt (triage cards
	// and digest messages), labeled by outcome.
	SlackPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prherder_slack_publish_total",
			Help: "Total Slack publish attempts, labeled by outcome.",
		},
		[]string{"outcome"},
	)

	// WorkerErrorsTotal counts processOne failures that led to a retry
	// or dead-letter -- distinct from LLM/Slack failure counters above,
	// since this tracks the whole-event failure, not just one call
	// within it.
	WorkerErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "prherder_worker_errors_total",
			Help: "Total events that failed processing (led to retry or dead-letter).",
		},
	)

	// EventsDeadLettered counts events that exhausted MaxRetries and
	// were dead-lettered -- ideally stays at or near zero; a rising
	// count means something is systematically broken, not just flaky.
	EventsDeadLettered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "prherder_events_dead_lettered_total",
			Help: "Total events dead-lettered after exhausting max retries.",
		},
	)
)

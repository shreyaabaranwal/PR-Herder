
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	
	WebhookEventsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prherder_webhook_events_received_total",
			Help: "Total webhook events received, labeled by event_type and duplicate status.",
		},
		[]string{"event_type", "duplicate"},
	)

	
	TriageDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "prherder_triage_duration_seconds",
			Help:    "Duration of a single PR triage-and-publish cycle.",
			Buckets: prometheus.DefBuckets, // 0.005s to 10s, good default spread
		},
	)

	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prherder_llm_requests_total",
			Help: "Total LLM summary requests, labeled by outcome.",
		},
		[]string{"outcome"},
	)


	SlackPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prherder_slack_publish_total",
			Help: "Total Slack publish attempts, labeled by outcome.",
		},
		[]string{"outcome"},
	)


	WorkerErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "prherder_worker_errors_total",
			Help: "Total events that failed processing (led to retry or dead-letter).",
		},
	)


	EventsDeadLettered = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "prherder_events_dead_lettered_total",
			Help: "Total events dead-lettered after exhausting max retries.",
		},
	)
)

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// UnprocessedEvent is a webhook_events row still awaiting processing —
// includes the row's own id (unlike WebhookEvent, used only for inserts)
// so the worker can mark it processed after handling it.
type UnprocessedEvent struct {
	ID         int64
	DeliveryID string
	EventType  string
	Action     string
	RepoOwner  string
	RepoName   string
	PRNumber   *int
	RawPayload map[string]any
	RetryCount int
}

// MaxRetries caps how many times a single event is retried across
// separate worker poll cycles before being dead-lettered. This is a
// second, outer layer of retry on top of ingest.withRetry's in-process
// retries for individual LLM/Slack calls -- if an event still fails
// after MaxRetries full processOne attempts, something is likely wrong
// with the event itself (bad payload, a real bug), not just transient
// network noise.
const MaxRetries = 5

// GetUnprocessedEvents fetches events awaiting processing, oldest first
// (FIFO — so a backlog drains in delivery order, not reverse). Bounded
// by limit so one worker poll can't accidentally try to load an
// unbounded backlog into memory at once. Dead-lettered events are
// excluded -- they've exhausted retries and need manual inspection, not
// another automatic attempt.
func (s *Store) GetUnprocessedEvents(ctx context.Context, limit int) ([]UnprocessedEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repo_owner, repo_name, pr_number, raw_payload, retry_count
		FROM webhook_events
		WHERE processed_at IS NULL AND dead_lettered_at IS NULL
		ORDER BY received_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query unprocessed events: %w", err)
	}
	defer rows.Close()

	var events []UnprocessedEvent
	for rows.Next() {
		var ev UnprocessedEvent
		var rawPayload []byte
		if err := rows.Scan(&ev.ID, &ev.DeliveryID, &ev.EventType, &ev.Action,
			&ev.RepoOwner, &ev.RepoName, &ev.PRNumber, &rawPayload, &ev.RetryCount); err != nil {
			return nil, fmt.Errorf("scan unprocessed event: %w", err)
		}
		if err := json.Unmarshal(rawPayload, &ev.RawPayload); err != nil {
			return nil, fmt.Errorf("unmarshal raw_payload: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unprocessed events: %w", err)
	}

	return events, nil
}

// MarkEventProcessed sets processed_at, so GetUnprocessedEvents won't
// return this row again. Called after successful processing.
func (s *Store) MarkEventProcessed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processed_at = $1 WHERE id = $2
	`, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("mark event processed: %w", err)
	}
	return nil
}

// MarkEventFailed records a processing error and increments retry_count.
// Once retry_count reaches MaxRetries, the event is dead-lettered
// (dead_lettered_at set) so GetUnprocessedEvents stops returning it --
// a permanently-broken event would otherwise retry forever, every poll
// cycle, indefinitely.
func (s *Store) MarkEventFailed(ctx context.Context, id int64, processErr error) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events
		SET process_error = $1,
		    retry_count = retry_count + 1,
		    dead_lettered_at = CASE
		        WHEN retry_count + 1 >= $3 THEN $2
		        ELSE dead_lettered_at
		    END
		WHERE id = $4
	`, processErr.Error(), time.Now().UTC(), MaxRetries, id)
	if err != nil {
		return fmt.Errorf("mark event failed: %w", err)
	}
	return nil
}

// GetDeadLetteredEvents returns events that exhausted all retries -- for
// manual inspection (e.g. a future admin endpoint or CLI command), not
// automatic reprocessing.
func (s *Store) GetDeadLetteredEvents(ctx context.Context, limit int) ([]UnprocessedEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repo_owner, repo_name, pr_number, raw_payload, retry_count
		FROM webhook_events
		WHERE dead_lettered_at IS NOT NULL
		ORDER BY dead_lettered_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query dead-lettered events: %w", err)
	}
	defer rows.Close()

	var events []UnprocessedEvent
	for rows.Next() {
		var ev UnprocessedEvent
		var rawPayload []byte
		if err := rows.Scan(&ev.ID, &ev.DeliveryID, &ev.EventType, &ev.Action,
			&ev.RepoOwner, &ev.RepoName, &ev.PRNumber, &rawPayload, &ev.RetryCount); err != nil {
			return nil, fmt.Errorf("scan dead-lettered event: %w", err)
		}
		if err := json.Unmarshal(rawPayload, &ev.RawPayload); err != nil {
			return nil, fmt.Errorf("unmarshal raw_payload: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dead-lettered events: %w", err)
	}

	return events, nil
}

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
}

// GetUnprocessedEvents fetches events awaiting processing, oldest first
// (FIFO — so a backlog drains in delivery order, not reverse). Bounded
// by limit so one worker poll can't accidentally try to load an
// unbounded backlog into memory at once.
func (s *Store) GetUnprocessedEvents(ctx context.Context, limit int) ([]UnprocessedEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, action, repo_owner, repo_name, pr_number, raw_payload
		FROM webhook_events
		WHERE processed_at IS NULL
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
			&ev.RepoOwner, &ev.RepoName, &ev.PRNumber, &rawPayload); err != nil {
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

// MarkEventFailed records a processing error without setting
// processed_at — leaving the row eligible for retry on the next poll,
// while process_error gives visibility into why previous attempts
// failed (useful for a stuck event that keeps failing the same way).
func (s *Store) MarkEventFailed(ctx context.Context, id int64, processErr error) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET process_error = $1 WHERE id = $2
	`, processErr.Error(), id)
	if err != nil {
		return fmt.Errorf("mark event failed: %w", err)
	}
	return nil
}

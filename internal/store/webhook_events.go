package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)


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

const MaxRetries = 5


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


func (s *Store) MarkEventProcessed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processed_at = $1 WHERE id = $2
	`, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("mark event processed: %w", err)
	}
	return nil
}


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

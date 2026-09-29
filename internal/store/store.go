
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}


func (s *Store) InsertWebhookEvent(ctx context.Context, ev WebhookEvent) (inserted bool, err error) {
	payload, err := json.Marshal(ev.RawPayload)
	if err != nil {
		return false, fmt.Errorf("marshal payload: %w", err)
	}

	tag, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events
			(delivery_id, event_type, action, repo_owner, repo_name, pr_number, raw_payload, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (delivery_id) DO NOTHING
	`, ev.DeliveryID, ev.EventType, ev.Action, ev.RepoOwner, ev.RepoName, ev.PRNumber, payload, time.Now().UTC())
	if err != nil {
		return false, fmt.Errorf("insert webhook_event: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

type WebhookEvent struct {
	DeliveryID string
	EventType  string
	Action     string
	RepoOwner  string
	RepoName   string
	PRNumber   *int 
	RawPayload map[string]any
}

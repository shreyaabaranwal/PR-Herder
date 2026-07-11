// Package store wraps Postgres access. It is the only package allowed to
// hold a *pgxpool.Pool or write raw SQL — everything else calls methods
// here. Keeping SQL in one place means the idempotency and upsert logic
// (the two things that MUST be correct) live in exactly one reviewable spot.
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

// Ping checks the Postgres connection is alive -- used by /readyz so a
// k8s readiness probe can detect "process is up but DB is unreachable"
// and stop routing traffic here.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// InsertWebhookEvent records a raw webhook delivery. It relies on the
// UNIQUE constraint on delivery_id to make this idempotent: if GitHub
// redelivers the same event (which it does, on timeout or manual
// redelivery), the second insert is a no-op and we report "duplicate"
// rather than erroring or double-processing.
//
// This is deliberately a plain INSERT with ON CONFLICT DO NOTHING rather
// than an application-level "SELECT then INSERT" check — that pattern has
// a race window between the check and the insert under concurrent
// requests. Let Postgres's unique index be the single source of truth.
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

// WebhookEvent is the store's view of an inbound delivery — a thin
// pass-through struct. It is intentionally not domain.PullRequest: this
// table stores the raw envelope, not our interpreted PR model. The
// translation into domain.PullRequest happens in the ingest processor,
// not here.
type WebhookEvent struct {
	DeliveryID string
	EventType  string
	Action     string
	RepoOwner  string
	RepoName   string
	PRNumber   *int // nil for events that aren't PR-scoped
	RawPayload map[string]any
}
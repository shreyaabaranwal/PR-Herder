package ingest

import (
	"context"
	"time"
)

// withRetry runs fn up to maxAttempts times with exponential backoff
// (1s, 2s, 4s, ...) between attempts. Used for calls to external
// services (LLM, Slack) that can fail transiently -- a single network
// blip shouldn't lose a whole triage result. Does NOT retry on ctx
// cancellation -- if the process is shutting down, stop immediately
// rather than sleeping through a graceful shutdown window.
func withRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return lastErr
}

// Package httpmw holds small, dependency-free HTTP middleware shared
// across all of PR Herder's handlers.
package httpmw

import (
	"log/slog"
	"net/http"
)

// Recover wraps a handler so a panic inside it (malformed payload,
// nil-pointer on an unexpected field, etc.) logs and returns 500
// instead of crashing the whole process -- taking the webhook endpoint,
// worker, and scheduler down with it.
func Recover(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered in HTTP handler",
					"panic", rec, "path", r.URL.Path, "method", r.Method)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

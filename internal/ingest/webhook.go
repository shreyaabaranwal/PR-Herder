// Package ingest is the GitHub-facing edge of PR Herder. It is the ONLY
// package that knows GitHub's webhook wire format (headers, payload
// shape). Its job: verify the request is genuinely from GitHub, extract
// just enough routing info to store the event idempotently, and ACK fast.
//
// It deliberately does NOT run triage logic. That happens in a separate
// worker reading from webhook_events, so a slow/buggy triage rule can
// never cause GitHub to see a webhook timeout and start retry-storming us.
package ingest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/shreyaabaranwal/pr-herder/internal/store"
)

const (
	headerSignature256 = "X-Hub-Signature-256"
	headerEvent        = "X-GitHub-Event"
	headerDelivery     = "X-GitHub-Delivery"

	maxBodyBytes = 5 << 20 // 5MB
)

type Handler struct {
	webhookSecret []byte
	store         *store.Store
	log           *slog.Logger
}

func NewHandler(webhookSecret string, s *store.Store, log *slog.Logger) *Handler {
	return &Handler{
		webhookSecret: []byte(webhookSecret),
		store:         s,
		log:           log,
	}
}

// ServeHTTP implements the GitHub webhook endpoint: POST /github/webhook
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		h.log.Error("read webhook body failed", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(body) > maxBodyBytes {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	sig := r.Header.Get(headerSignature256)
	if !h.verifySignature(body, sig) {
		h.log.Warn("webhook signature verification failed", "remote", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get(headerEvent)
	deliveryID := r.Header.Get(headerDelivery)
	if eventType == "" || deliveryID == "" {
		http.Error(w, "missing required headers", http.StatusBadRequest)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		h.log.Error("invalid JSON payload", "delivery_id", deliveryID, "err", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	ev, ok := extractRoutingInfo(eventType, deliveryID, payload)
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	inserted, err := h.store.InsertWebhookEvent(r.Context(), ev)
	if err != nil {
		h.log.Error("failed to persist webhook event", "delivery_id", deliveryID, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !inserted {
		h.log.Info("duplicate delivery ignored", "delivery_id", deliveryID)
	} else {
		h.log.Info("webhook event stored",
			"delivery_id", deliveryID,
			"event_type", eventType,
			"action", ev.Action,
			"repo", fmt.Sprintf("%s/%s", ev.RepoOwner, ev.RepoName),
		)
	}

	w.WriteHeader(http.StatusAccepted)
}

// verifySignature checks X-Hub-Signature-256 using constant-time
// comparison via hmac.Equal — never == or bytes.Equal, which leak timing
// information an attacker could exploit to recover the signature byte by
// byte.
func (h *Handler) verifySignature(body []byte, sigHeader string) bool {
	const prefix = "sha256="
	if len(sigHeader) <= len(prefix) || sigHeader[:len(prefix)] != prefix {
		return false
	}
	expectedHex := sigHeader[len(prefix):]
	expected, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, h.webhookSecret)
	mac.Write(body)
	computed := mac.Sum(nil)

	return hmac.Equal(computed, expected)
}
package slackui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

type Authorizer interface {
	CanActOnRepo(ctx context.Context, slackUserID, repoOwner, repoName string) (allowed bool, reason string, err error)
}

type ActionExecutor interface {
	Execute(ctx context.Context, action, repoOwner, repoName string, prNumber int) error
}

type InteractionHandler struct {
	signingSecret string
	authorizer    Authorizer
	executor      ActionExecutor
	log           *slog.Logger
}

func NewInteractionHandler(signingSecret string, authorizer Authorizer, executor ActionExecutor, log *slog.Logger) *InteractionHandler {
	return &InteractionHandler{
		signingSecret: signingSecret,
		authorizer:    authorizer,
		executor:      executor,
		log:           log,
	}
}

type blockActionsPayload struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Actions []struct {
		ActionID string `json:"action_id"`
		Value    string `json:"value"`
	} `json:"actions"`
	ResponseURL string `json:"response_url"`
}

func (h *InteractionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	timestamp := r.Header.Get(headerSlackTimestamp)
	signature := r.Header.Get(headerSlackSignature)
	if !VerifySlackSignature(h.signingSecret, timestamp, signature, body) {
		h.log.Warn("slack interaction signature verification failed", "remote", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	var payload blockActionsPayload
	if err := json.Unmarshal([]byte(values.Get("payload")), &payload); err != nil {
		h.log.Error("failed to parse slack interaction payload", "err", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if len(payload.Actions) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	action := payload.Actions[0]

	repoOwner, repoName, prNumber, err := parseActionValue(action.Value)
	if err != nil {
		h.log.Error("failed to parse action value", "value", action.Value, "err", err)
		http.Error(w, "invalid action value", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	allowed, reason, err := h.authorizer.CanActOnRepo(ctx, payload.User.ID, repoOwner, repoName)
	if err != nil {
		h.log.Error("authorization check failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		h.log.Info("action denied", "slack_user", payload.User.ID, "repo", repoOwner+"/"+repoName, "reason", reason)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.executor.Execute(ctx, action.ActionID, repoOwner, repoName, prNumber); err != nil {
		h.log.Error("action execution failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.log.Info("action executed", "slack_user", payload.User.ID, "action", action.ActionID, "repo", repoOwner+"/"+repoName, "pr", prNumber)
	w.WriteHeader(http.StatusOK)
}

func parseActionValue(value string) (owner, repo string, prNumber int, err error) {
	var slashIdx, hashIdx = -1, -1
	for i, c := range value {
		if c == '/' && slashIdx == -1 {
			slashIdx = i
		}
		if c == '#' {
			hashIdx = i
		}
	}
	if slashIdx == -1 || hashIdx == -1 || hashIdx < slashIdx {
		return "", "", 0, fmt.Errorf("malformed action value: %q", value)
	}
	owner = value[:slashIdx]
	repo = value[slashIdx+1 : hashIdx]
	numStr := value[hashIdx+1:]
	prNumber, err = atoiSimple(numStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number in %q: %w", value, err)
	}
	return owner, repo, prNumber, nil
}

func atoiSimple(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

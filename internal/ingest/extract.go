package ingest

import "github.com/shreyaabaranwal/pr-herder/internal/store"

// eventsWeCareAbout gates which GitHub event types we bother storing.
// GitHub sends dozens of event types (star, fork, push, issues, ...) to
// any webhook subscribed broadly; we only want pull_request and
// pull_request-adjacent events (reviews, check runs) for the triage loop.
var eventsWeCareAbout = map[string]bool{
	"pull_request":        true,
	"pull_request_review": true,
	"check_run":           false, // Layer 6 will flip this on
	"check_suite":         false, // Layer 6 will flip this on
}

// extractRoutingInfo pulls just enough out of a GitHub payload to store
// and later route the event: repo, PR number, action. It does NOT build a
// domain.PullRequest — that happens in the ingest worker (Layer 1 next
// piece), which reads raw_payload back out of Postgres and does the full
// translation.
func extractRoutingInfo(eventType, deliveryID string, payload map[string]any) (store.WebhookEvent, bool) {
	if care, known := eventsWeCareAbout[eventType]; !known || !care {
		return store.WebhookEvent{}, false
	}

	repo, _ := payload["repository"].(map[string]any)
	owner := nestedString(repo, "owner", "login")
	name := stringField(repo, "name")
	if owner == "" || name == "" {
		return store.WebhookEvent{}, false
	}

	action := stringField(payload, "action")

	var prNumber *int
	if pr, ok := payload["pull_request"].(map[string]any); ok {
		if n, ok := numberField(pr, "number"); ok {
			prNumber = &n
		}
	} else if n, ok := numberField(payload, "number"); ok {
		prNumber = &n
	}

	return store.WebhookEvent{
		DeliveryID: deliveryID,
		EventType:  eventType,
		Action:     action,
		RepoOwner:  owner,
		RepoName:   name,
		PRNumber:   prNumber,
		RawPayload: payload,
	}, true
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func nestedString(m map[string]any, keys ...string) string {
	cur := m
	for i, k := range keys {
		if cur == nil {
			return ""
		}
		if i == len(keys)-1 {
			return stringField(cur, k)
		}
		next, _ := cur[k].(map[string]any)
		cur = next
	}
	return ""
}

// numberField handles the fact that encoding/json decodes all JSON numbers
// into float64 when the target is map[string]any (no schema to guide it).
func numberField(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	f, ok := m[key].(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}
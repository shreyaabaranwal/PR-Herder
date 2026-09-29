package ingest

import "github.com/shreyaabaranwal/pr-herder/internal/store"


var eventsWeCareAbout = map[string]bool{
	"pull_request":        true,
	"pull_request_review": true,
	"check_run":           false, 
	"check_suite":         false,
}


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
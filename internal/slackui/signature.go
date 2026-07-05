package slackui

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	headerSlackSignature = "X-Slack-Signature"
	headerSlackTimestamp = "X-Slack-Request-Timestamp"

	// Slack recommends rejecting requests older than 5 minutes to prevent
	// replay attacks — a captured valid request replayed later should not
	// still authorize an action. This mirrors the replay-protection
	// principle from docs/SECURITY.md invariant #2.
	maxRequestAge = 5 * time.Minute
)

// VerifySlackSignature checks an inbound Slack request against Slack's
// v0 signing scheme: HMAC-SHA256 over "v0:{timestamp}:{raw body}", keyed
// by the app's signing secret, encoded as "v0={hex}".
//
// This is deliberately a separate function from GitHub's webhook
// verification (internal/ingest/webhook.go) even though both use
// HMAC-SHA256 — the two providers sign different string constructions
// (GitHub signs the raw body; Slack signs a composed string including
// the timestamp), so sharing code would obscure that difference rather
// than simplify anything.
func VerifySlackSignature(signingSecret string, timestampHeader, signatureHeader string, body []byte) bool {
	if timestampHeader == "" || signatureHeader == "" {
		return false
	}

	ts, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return false
	}

	age := time.Since(time.Unix(ts, 0))
	if math.Abs(age.Seconds()) > maxRequestAge.Seconds() {
		// Too old (replay risk) or too far in the future (clock skew /
		// forged timestamp) — reject either way rather than guessing
		// which case it is.
		return false
	}

	baseString := fmt.Sprintf("v0:%s:%s", timestampHeader, string(body))

	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))
	computed := "v0=" + hex.EncodeToString(mac.Sum(nil))

	// hmac.Equal is not directly usable on the "v0=" prefixed strings
	// (different lengths would short-circuit unsafely in a plain ==
	// anyway) — compare the raw MAC bytes instead for a true
	// constant-time comparison, same principle as the GitHub webhook
	// verifier.
	expectedHex := strings.TrimPrefix(signatureHeader, "v0=")
	expectedBytes, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}

	return hmac.Equal(mac.Sum(nil), expectedBytes) && strings.HasPrefix(computed, "v0=")
}
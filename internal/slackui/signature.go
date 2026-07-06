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
	maxRequestAge         = 5 * time.Minute
)

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
		return false
	}

	baseString := fmt.Sprintf("v0:%s:%s", timestampHeader, string(body))

	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(baseString))

	expectedHex := strings.TrimPrefix(signatureHeader, "v0=")
	expectedBytes, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}

	return hmac.Equal(mac.Sum(nil), expectedBytes)
}

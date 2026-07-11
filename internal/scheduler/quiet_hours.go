// Package scheduler runs periodic, non-real-time jobs -- currently just
// the stale-PR digest. Deliberately separate from ingest.Worker: this
// reads live GitHub state on a timer, it doesn't react to webhook
// events, so it has no business sharing a poll loop with the real-time
// path.
package scheduler

import "time"

// IsQuietHours reports whether hour t falls inside the configured quiet
// window [startHour, endHour). Handles wraparound across midnight (e.g.
// start=22, end=8 means 22:00-23:59 AND 00:00-07:59 are quiet) since
// that's the common real-world case for "don't message me overnight."
//
// startHour == endHour is treated as "quiet hours disabled" rather than
// "quiet 24/7" -- a zero-width window is almost certainly a config
// mistake, and silently blocking every digest forever is a worse
// failure mode than occasionally sending one at an inconvenient hour.
func IsQuietHours(t time.Time, startHour, endHour int) bool {
	if startHour == endHour {
		return false
	}
	hour := t.Hour()
	if startHour < endHour {
		return hour >= startHour && hour < endHour
	}
	// wraparound case, e.g. 22-8
	return hour >= startHour || hour < endHour
}
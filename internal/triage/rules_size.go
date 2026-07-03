package triage

// SizeCategory buckets a PR by total lines changed. Thresholds are
// deliberately simple constants here rather than configurable — this is
// exactly the kind of rule a repo would want to tune later (Layer 5+
// config), but hardcoding it now keeps Layer 2 focused on proving the
// deterministic-rules architecture works end to end.
type SizeCategory string

const (
	SizeS  SizeCategory = "S"  // 0-9 lines
	SizeM  SizeCategory = "M"  // 10-99 lines
	SizeL  SizeCategory = "L"  // 100-499 lines
	SizeXL SizeCategory = "XL" // 500+ lines
)

// ClassifySize buckets a PR's total diff size (additions + deletions).
// Pure arithmetic — no reason this should ever need a network call, which
// is exactly why it belongs in Layer 2's deterministic tier rather than
// being punted to the LLM.
func ClassifySize(additions, deletions int) SizeCategory {
	total := additions + deletions
	switch {
	case total < 10:
		return SizeS
	case total < 100:
		return SizeM
	case total < 500:
		return SizeL
	default:
		return SizeXL
	}
}
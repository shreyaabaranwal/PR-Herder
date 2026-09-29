package triage

import "testing"

func TestClassifyFlaky(t *testing.T) {
	tests := []struct {
		name         string
		observations []CheckRunObservation
		wantFlaky    bool
	}{
		{
			name: "consistently passing check is not flaky",
			observations: []CheckRunObservation{
				{CommitSHA: "a1", Conclusion: "success"},
				{CommitSHA: "a2", Conclusion: "success"},
				{CommitSHA: "a3", Conclusion: "success"},
			},
			wantFlaky: false,
		},
		{
			name: "consistently failing check is not flaky -- it's just broken",
			observations: []CheckRunObservation{
				{CommitSHA: "b1", Conclusion: "failure"},
				{CommitSHA: "b2", Conclusion: "failure"},
				{CommitSHA: "b3", Conclusion: "failure"},
			},
			wantFlaky: false,
		},
		{
			name: "fail-then-pass on same commit (retry) marks flaky",
			observations: []CheckRunObservation{
				{CommitSHA: "c1", Conclusion: "failure"},
				{CommitSHA: "c1", Conclusion: "success"}, 
				{CommitSHA: "c2", Conclusion: "failure"},
				{CommitSHA: "c2", Conclusion: "success"}, 
				{CommitSHA: "c3", Conclusion: "success"},
			},
			wantFlaky: true,
		},
		{
			name: "below minimum sample size is never flaky, even with fail-then-pass",
			observations: []CheckRunObservation{
				{CommitSHA: "d1", Conclusion: "failure"},
				{CommitSHA: "d1", Conclusion: "success"},
			},
			wantFlaky: false, 
		},
		{
			name: "new commit after a failure is not a retry -- no fail-then-pass signal",
			observations: []CheckRunObservation{
				{CommitSHA: "e1", Conclusion: "failure"},
				{CommitSHA: "e2", Conclusion: "success"}, 
				{CommitSHA: "e3", Conclusion: "success"},
			},
			wantFlaky: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := ClassifyFlaky(tt.observations)
			if verdict.IsFlaky != tt.wantFlaky {
				t.Errorf("IsFlaky = %v, want %v (rate=%.2f, sample=%d)",
					verdict.IsFlaky, tt.wantFlaky, verdict.FailThenPassRate, verdict.SampleSize)
			}
		})
	}
}

func TestClassifyFlaky_EmptyHistory(t *testing.T) {
	verdict := ClassifyFlaky(nil)
	if verdict.IsFlaky {
		t.Error("empty history should never be classified flaky")
	}
	if verdict.SampleSize != 0 {
		t.Errorf("SampleSize = %d, want 0", verdict.SampleSize)
	}
}

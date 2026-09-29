package triage

import (
	"testing"

	"github.com/shreyaabaranwal/pr-herder/internal/domain"
)


func TestEngine_Triage(t *testing.T) {
	testRules := []SensitivePathRule{
		{PathPrefix: "internal/auth/", Team: "@security-team", Label: "needs-security-review"},
	}
	engine := &Engine{SensitivePathRules: testRules}

	tests := []struct {
		name string
		pr   domain.PullRequest

		wantSize        SizeCategory
		wantPathMatched bool
		wantAmbiguous   bool
		wantFirstTimer  bool
	}{
		{
			name: "small PR, no sensitive path, not ambiguous",
			pr: domain.PullRequest{
				Additions:    3,
				Deletions:    2,
				ChangedFiles: []string{"README.md"},
				Association:  domain.AssocMember,
			},
			wantSize:        SizeS,
			wantPathMatched: false,
			wantAmbiguous:   false,
			wantFirstTimer:  false,
		},
		{
			name: "touches sensitive path, routed regardless of size",
			pr: domain.PullRequest{
				Additions:    600,
				Deletions:    100,
				ChangedFiles: []string{"internal/auth/middleware.go"},
				Association:  domain.AssocContributor,
			},
			wantSize:        SizeXL,
			wantPathMatched: true,
			wantAmbiguous:   false, 
			wantFirstTimer:  false,
		},
		{
			name: "large diff, no sensitive path match, ambiguous",
			pr: domain.PullRequest{
				Additions:    400,
				Deletions:    200,
				ChangedFiles: []string{"internal/whatever/big_refactor.go"},
				Association:  domain.AssocMember,
			},
			wantSize:        SizeXL,
			wantPathMatched: false,
			wantAmbiguous:   true, 
			wantFirstTimer:  false,
		},
		{
			name: "first-time contributor flagged",
			pr: domain.PullRequest{
				Additions:    5,
				Deletions:    1,
				ChangedFiles: []string{"docs/example.md"},
				Association:  domain.AssocFirstTimer,
			},
			wantSize:        SizeS,
			wantPathMatched: false,
			wantAmbiguous:   false,
			wantFirstTimer:  true,
		},
		{
			name: "medium diff stays deterministic, not ambiguous",
			pr: domain.PullRequest{
				Additions:    40,
				Deletions:    10,
				ChangedFiles: []string{"internal/other/thing.go"},
				Association:  domain.AssocCollaborator,
			},
			wantSize:        SizeM,
			wantPathMatched: false,
			wantAmbiguous:   false, 	
			wantFirstTimer:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.Triage(tt.pr)

			if result.SizeCategory != tt.wantSize {
				t.Errorf("SizeCategory = %v, want %v", result.SizeCategory, tt.wantSize)
			}
			if result.PathMatch.Matched != tt.wantPathMatched {
				t.Errorf("PathMatch.Matched = %v, want %v", result.PathMatch.Matched, tt.wantPathMatched)
			}
			if result.Ambiguous != tt.wantAmbiguous {
				t.Errorf("Ambiguous = %v, want %v", result.Ambiguous, tt.wantAmbiguous)
			}
			if result.Contributor.IsFirstTimer != tt.wantFirstTimer {
				t.Errorf("Contributor.IsFirstTimer = %v, want %v", result.Contributor.IsFirstTimer, tt.wantFirstTimer)
			}
		})
	}
}


func TestClassifySize_Boundaries(t *testing.T) {
	tests := []struct {
		additions, deletions int
		want                 SizeCategory
	}{
		{0, 0, SizeS},
		{9, 0, SizeS},
		{10, 0, SizeM},
		{99, 0, SizeM},
		{100, 0, SizeL},
		{499, 0, SizeL},
		{500, 0, SizeXL},
		{250, 250, SizeXL}, 
	}

	for _, tt := range tests {
		got := ClassifySize(tt.additions, tt.deletions)
		if got != tt.want {
			t.Errorf("ClassifySize(%d, %d) = %v, want %v", tt.additions, tt.deletions, got, tt.want)
		}
	}
}
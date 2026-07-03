package triage

import "github.com/shreyaabaranwal/pr-herder/internal/domain"

// ContributorSignal captures what triage knows about the PR author,
// purely from GitHub's own author_association field — no scraping, no
// inference beyond what GitHub already publishes. See docs/SECURITY.md's
// least-privilege-data principle.
type ContributorSignal struct {
	IsFirstTimer     bool
	SuggestedWelcome bool
}

// ClassifyContributor flags PRs from first-time contributors so the Slack
// card can surface a suggested (never auto-posted) welcome message. The
// maintainer confirms before anything is posted back to GitHub — this
// function only produces a suggestion.
func ClassifyContributor(assoc domain.AuthorAssociation) ContributorSignal {
	isFirstTimer := assoc == domain.AssocFirstTimer || assoc == domain.AssocFirstTimeToRepo || assoc == domain.AssocNone

	return ContributorSignal{
		IsFirstTimer:     isFirstTimer,
		SuggestedWelcome: isFirstTimer,
	}
}
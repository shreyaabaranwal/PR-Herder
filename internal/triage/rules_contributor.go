package triage

import "github.com/shreyaabaranwal/pr-herder/internal/domain"


type ContributorSignal struct {
	IsFirstTimer     bool
	SuggestedWelcome bool
}


func ClassifyContributor(assoc domain.AuthorAssociation) ContributorSignal {
	isFirstTimer := assoc == domain.AssocFirstTimer || assoc == domain.AssocFirstTimeToRepo || assoc == domain.AssocNone

	return ContributorSignal{
		IsFirstTimer:     isFirstTimer,
		SuggestedWelcome: isFirstTimer,
	}
}
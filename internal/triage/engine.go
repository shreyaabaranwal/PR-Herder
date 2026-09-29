package triage

import "github.com/shreyaabaranwal/pr-herder/internal/domain"


type Result struct {
	PR domain.PullRequest

	SizeCategory SizeCategory
	PathMatch    PathMatch
	Contributor  ContributorSignal

	
	Ambiguous bool

	
	Labels []string


	FlakyChecks map[string]FlakyVerdict

	
	Summary string
}


type Engine struct {
	SensitivePathRules []SensitivePathRule
}


func NewEngine() *Engine {
	return &Engine{SensitivePathRules: DefaultSensitivePathRules}
}


func (e *Engine) Triage(pr domain.PullRequest) Result {
	pathMatch := MatchSensitivePaths(pr.ChangedFiles, e.SensitivePathRules)
	size := ClassifySize(pr.Additions, pr.Deletions)
	contributor := ClassifyContributor(pr.Association)

	var labels []string
	if pathMatch.Matched {
		labels = append(labels, pathMatch.Label)
	}
	labels = append(labels, "size/"+string(size))

	
	ambiguous := !pathMatch.Matched && (size == SizeL || size == SizeXL)

	return Result{
		PR:           pr,
		SizeCategory: size,
		PathMatch:    pathMatch,
		Contributor:  contributor,
		Ambiguous:    ambiguous,
		Labels:       labels,
	}
}
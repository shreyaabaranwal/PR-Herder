
package domain

import (
	"strconv"
	"time"
)


type AuthorAssociation string

const (
	AssocOwner           AuthorAssociation = "OWNER"
	AssocMember          AuthorAssociation = "MEMBER"
	AssocCollaborator    AuthorAssociation = "COLLABORATOR"
	AssocContributor     AuthorAssociation = "CONTRIBUTOR"
	AssocFirstTimer      AuthorAssociation = "FIRST_TIME_CONTRIBUTOR"
	AssocFirstTimeToRepo AuthorAssociation = "FIRST_TIMER"
	AssocNone            AuthorAssociation = "NONE"
)


type CIStatus string

const (
	CIPending CIStatus = "pending"
	CIPassing CIStatus = "passing"
	CIFailing CIStatus = "failing"
	CIFlaky   CIStatus = "flaky" 
)


type PullRequest struct {
	ID          int64 
	RepoOwner   string
	RepoName    string
	Number      int
	Title       string
	Body        string
	AuthorLogin string
	Association AuthorAssociation

	BaseBranch string
	HeadBranch string
	HeadSHA    string

	Additions    int
	Deletions    int
	ChangedFiles []string 

	CIStatus CIStatus

	State  string 
	Merged bool

	CreatedAt time.Time
	UpdatedAt time.Time


	LastEventID string
}


func (p PullRequest) Key() string {
	return p.RepoOwner + "/" + p.RepoName + "#" + strconv.Itoa(p.Number)
}

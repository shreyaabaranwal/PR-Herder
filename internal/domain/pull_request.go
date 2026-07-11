  // Package domain holds PR Herder's internal model of the world.
//
// Deliberately independent of GitHub's webhook JSON shape and of Slack's
// Block Kit shape. GitHub's payload has nullable fields, deprecated fields,
// and quirks that change across API versions — none of that should leak
// into triage logic. ingest/ is the only package allowed to know about
// GitHub's wire format; it translates into these types at the boundary.
package domain

import (
	"strconv"
	"time"
)

// AuthorAssociation mirrors GitHub's own enum for a PR author's relationship
// to the repo. We keep it as a distinct type (not a raw string) so the
// compiler catches typos like "FIRST_TIME_CONTRIBUTOR" vs "FIRST_TIME_CONTRIBUTER".
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

// CIStatus is our normalized view of "is the check suite happy."
// GitHub actually has separate "status" and "conclusion" fields with many
// values (queued, in_progress, success, failure, neutral, cancelled,
// timed_out, action_required, stale). We collapse that into what triage
// actually needs to decide on.
type CIStatus string

const (
	CIPending CIStatus = "pending"
	CIPassing CIStatus = "passing"
	CIFailing CIStatus = "failing"
	CIFlaky   CIStatus = "flaky" // set by the flaky-CI classifier (Layer 6), not by ingest
)

// PullRequest is PR Herder's canonical representation of a GitHub PR,
// assembled from webhook events. This is what gets persisted, triaged,
// and rendered — never the raw GitHub payload.
type PullRequest struct {
	ID          int64 // our internal surrogate key (from store), 0 until persisted
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
	ChangedFiles []string // paths only; used for CODEOWNERS + sensitive-path matching

	CIStatus CIStatus

	State  string // "open" | "closed"
	Merged bool

	CreatedAt time.Time
	UpdatedAt time.Time

	// LastEventID is the delivery ID of the most recent webhook event that
	// touched this PR. Used for idempotency + debugging ("which event
	// produced this state?"), not a business field.
	LastEventID string
}

// Key uniquely identifies a PR across the whole system: owner/repo#number.
// Used as the natural key for upserts (see store package) since GitHub's
// PR "id" field is not human-legible and node_id churns across API versions.
func (p PullRequest) Key() string {
	return p.RepoOwner + "/" + p.RepoName + "#" + strconv.Itoa(p.Number)
}

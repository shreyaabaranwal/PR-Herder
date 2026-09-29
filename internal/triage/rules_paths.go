
package triage

import "strings"

type SensitivePathRule struct {
	PathPrefix string
	Team       string
	Label      string
}


var DefaultSensitivePathRules = []SensitivePathRule{
	{PathPrefix: "internal/auth/", Team: "@security-team", Label: "needs-security-review"},
	{PathPrefix: "internal/authz/", Team: "@security-team", Label: "needs-security-review"},
	{PathPrefix: ".github/workflows/", Team: "@platform-team", Label: "needs-platform-review"},
	{PathPrefix: "migrations/", Team: "@data-team", Label: "needs-data-review"},
}


type PathMatch struct {
	Matched bool
	Team    string
	Label   string
	Path    string 
}


func MatchSensitivePaths(changedFiles []string, rules []SensitivePathRule) PathMatch {
	for _, rule := range rules {
		for _, f := range changedFiles {
			if strings.HasPrefix(f, rule.PathPrefix) {
				return PathMatch{
					Matched: true,
					Team:    rule.Team,
					Label:   rule.Label,
					Path:    f,
				}
			}
		}
	}
	return PathMatch{Matched: false}
}
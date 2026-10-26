package application

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

const (
	compareSourceBranchParameter = "sourceBranch"
	compareTargetBranchParameter = "targetBranch"
	compareSourceAliasParameter  = "to"
	compareTargetAliasParameter  = "from"
)

var (
	htmlHrefLinkPattern    = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
	plainURLLinkPattern    = regexp.MustCompile(`https?://[^\s"'<>]+`)
	comparePathPattern     = regexp.MustCompile(`(?i)/projects/([^/]+)/repos/([^/]+)/compare(?:/[a-z]+)?/?$`)
	pullRequestPathPattern = regexp.MustCompile(`(?i)/projects/([^/]+)/repos/([^/]+)/pull-requests/([0-9]+)(?:/[a-z]+)?/?$`)
)

type CompareCandidate struct {
	CompareURL    string
	Origin        string
	ProjectKey    string
	RepoSlug      string
	SourceBranch  string
	TargetBranch  string
	PullRequestID string
}

func parseCompareCandidate(rawLink string) (CompareCandidate, bool) {
	cleaned := html.UnescapeString(strings.TrimSpace(rawLink))
	cleaned = strings.TrimSuffix(cleaned, ".")
	parsed, err := url.Parse(cleaned)
	if err != nil || parsed.Host == "" {
		return CompareCandidate{}, false
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return CompareCandidate{}, false
	}

	origin := parsed.Scheme + "://" + parsed.Host
	if match := pullRequestPathPattern.FindStringSubmatch(parsed.Path); match != nil {
		return CompareCandidate{
			CompareURL:    parsed.String(),
			Origin:        origin,
			ProjectKey:    match[1],
			RepoSlug:      match[2],
			PullRequestID: match[3],
		}, true
	}

	if match := comparePathPattern.FindStringSubmatch(parsed.Path); match != nil {
		query := parsed.Query()
		sourceBranch := normalizeBranchRef(query.Get(compareSourceBranchParameter))
		targetBranch := normalizeBranchRef(query.Get(compareTargetBranchParameter))
		if sourceBranch == "" {
			sourceBranch = normalizeBranchRef(query.Get(compareSourceAliasParameter))
		}
		if targetBranch == "" {
			targetBranch = normalizeBranchRef(query.Get(compareTargetAliasParameter))
		}
		if sourceBranch == "" || targetBranch == "" {
			return CompareCandidate{}, false
		}

		return CompareCandidate{
			CompareURL:   parsed.String(),
			Origin:       origin,
			ProjectKey:   match[1],
			RepoSlug:     match[2],
			SourceBranch: sourceBranch,
			TargetBranch: targetBranch,
		}, true
	}

	return CompareCandidate{}, false
}

func linkCandidate(cellHTML string) (CompareCandidate, bool) {
	for _, match := range htmlHrefLinkPattern.FindAllStringSubmatch(cellHTML, -1) {
		if candidate, ok := parseCompareCandidate(match[1]); ok {
			return candidate, true
		}
	}
	for _, rawLink := range plainURLLinkPattern.FindAllString(cellHTML, -1) {
		if candidate, ok := parseCompareCandidate(rawLink); ok {
			return candidate, true
		}
	}
	return CompareCandidate{}, false
}

func normalizeBranchRef(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "refs/heads/")
	return trimmed
}

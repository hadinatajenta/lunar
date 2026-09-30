package application

import (
	"strings"
	"time"
)

const JIRA_CACHE_TTL = 60 * time.Second

func issuesCacheKey(userID string) string {
	return "issues|" + userID
}

func backlogCacheKey(userID string, requestedSquad string) string {
	return "backlog|" + userID + "|" + strings.TrimSpace(requestedSquad)
}

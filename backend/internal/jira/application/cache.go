package application

import (
	"context"
	"strings"
	"time"
)

const JIRA_CACHE_TTL = 60 * time.Second

type forceRefreshContextKey struct{}

func WithForceRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceRefreshContextKey{}, true)
}

func isForceRefresh(ctx context.Context) bool {
	force, ok := ctx.Value(forceRefreshContextKey{}).(bool)
	return ok && force
}

func issuesCacheKey(userID string) string {
	return "issues|" + userID
}

func backlogCacheKey(userID string, requestedSquad string) string {
	return "backlog|" + userID + "|" + strings.TrimSpace(requestedSquad)
}

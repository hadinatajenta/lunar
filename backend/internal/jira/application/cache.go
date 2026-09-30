package application

import (
	"context"
	"strings"
	"time"
)

const JIRA_CACHE_TTL = 30 * time.Second

const jiraCacheMaxEntries = 256

type jiraCacheEntry struct {
	value    any
	storedAt time.Time
}

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

func (s *JiraService) loadCacheEntry(key string) (any, bool) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	entry, exists := s.cacheEntries[key]
	if !exists {
		return nil, false
	}
	if time.Since(entry.storedAt) >= s.cacheTTL {
		delete(s.cacheEntries, key)
		return nil, false
	}
	return entry.value, true
}

func (s *JiraService) storeCacheEntry(key string, value any) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	now := time.Now()
	for entryKey, entry := range s.cacheEntries {
		if now.Sub(entry.storedAt) >= s.cacheTTL {
			delete(s.cacheEntries, entryKey)
		}
	}
	for len(s.cacheEntries) >= jiraCacheMaxEntries {
		s.evictOldestCacheEntry()
	}
	s.cacheEntries[key] = jiraCacheEntry{value: value, storedAt: now}
}

func (s *JiraService) evictOldestCacheEntry() {
	var oldestKey string
	var oldestAt time.Time
	isFirst := true
	for entryKey, entry := range s.cacheEntries {
		if isFirst || entry.storedAt.Before(oldestAt) {
			oldestKey = entryKey
			oldestAt = entry.storedAt
			isFirst = false
		}
	}
	delete(s.cacheEntries, oldestKey)
}

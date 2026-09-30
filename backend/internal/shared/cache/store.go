package cache

import (
	"sync"
	"time"
)

const maxEntries = 256

type entry struct {
	value    any
	storedAt time.Time
}

type Store struct {
	mutex   sync.Mutex
	entries map[string]entry
	ttl     time.Duration
}

func New(ttl time.Duration) *Store {
	return &Store{
		entries: make(map[string]entry),
		ttl:     ttl,
	}
}

func (s *Store) Get(key string) (any, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	stored, exists := s.entries[key]
	if !exists {
		return nil, false
	}
	if time.Since(stored.storedAt) >= s.ttl {
		delete(s.entries, key)
		return nil, false
	}
	return stored.value, true
}

func (s *Store) Set(key string, value any) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for entryKey, stored := range s.entries {
		if now.Sub(stored.storedAt) >= s.ttl {
			delete(s.entries, entryKey)
		}
	}
	for len(s.entries) >= maxEntries {
		s.evictOldest()
	}
	s.entries[key] = entry{value: value, storedAt: now}
}

func (s *Store) evictOldest() {
	var oldestKey string
	var oldestAt time.Time
	isFirst := true
	for entryKey, stored := range s.entries {
		if isFirst || stored.storedAt.Before(oldestAt) {
			oldestKey = entryKey
			oldestAt = stored.storedAt
			isFirst = false
		}
	}
	delete(s.entries, oldestKey)
}

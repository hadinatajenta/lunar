package cache

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStore_GetWithinTTLReturnsStoredValue(t *testing.T) {
	store := New(time.Minute)

	if _, exists := store.Get("missing"); exists {
		t.Error("expected miss for unknown key")
	}

	store.Set("documents|usr-1", "first-value")

	cached, exists := store.Get("documents|usr-1")
	if !exists {
		t.Fatal("expected hit within TTL")
	}
	if cached != "first-value" {
		t.Errorf("expected first-value, got %v", cached)
	}
}

func TestStore_GetAfterTTLExpiryMisses(t *testing.T) {
	store := New(20 * time.Millisecond)
	store.Set("documents|usr-1", "first-value")

	time.Sleep(60 * time.Millisecond)

	if _, exists := store.Get("documents|usr-1"); exists {
		t.Error("expected miss after TTL expiry")
	}

	store.Set("documents|usr-1", "second-value")
	cached, exists := store.Get("documents|usr-1")
	if !exists {
		t.Fatal("expected hit for value stored after expiry")
	}
	if cached != "second-value" {
		t.Errorf("expected second-value, got %v", cached)
	}
}

func TestStore_SetEvictsOldestEntryAtCap(t *testing.T) {
	store := New(time.Hour)

	store.Set("oldest", "first-value")
	time.Sleep(5 * time.Millisecond)
	for index := 0; index < maxEntries-1; index++ {
		store.Set("filler-"+strconv.Itoa(index), index)
	}

	if _, exists := store.Get("oldest"); !exists {
		t.Fatal("expected oldest entry present before cap breach")
	}

	store.Set("newest", "last-value")

	if _, exists := store.Get("oldest"); exists {
		t.Error("expected oldest entry to be evicted at cap")
	}
	if _, exists := store.Get("newest"); !exists {
		t.Error("expected newest entry to be stored at cap")
	}
	if _, exists := store.Get("filler-0"); !exists {
		t.Error("expected non-oldest filler entry to survive eviction")
	}
}

func TestStore_ConcurrentGetSet(t *testing.T) {
	store := New(time.Minute)
	store.Set("warmup", "warmup-value")

	const workerCount = 8
	const iterationCount = 10
	workerErrors := make(chan error, workerCount*iterationCount*2)
	var waitGroup sync.WaitGroup

	for worker := range workerCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for iteration := range iterationCount {
				key := "concurrent|" + strconv.Itoa(worker) + "|" + strconv.Itoa(iteration)
				store.Set(key, iteration)
				if _, exists := store.Get(key); !exists {
					workerErrors <- fmt.Errorf("expected concurrent key %s to be present", key)
				}
				if _, exists := store.Get("warmup"); !exists {
					workerErrors <- fmt.Errorf("expected warmup key to survive concurrent access")
				}
			}
		}()
	}

	waitGroup.Wait()
	close(workerErrors)

	for workerErr := range workerErrors {
		t.Error(workerErr)
	}
}

package infrastructure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lunar/backend/internal/utdocs/domain"
)

func TestRunStoreSavesAndLoadsWithRestrictivePermissions(t *testing.T) {
	store := NewRunStore(t.TempDir())
	payload := domain.RunPayload{RunID: "00000000-0000-4000-8000-000000000001", Repo: "micro-payment", Branch: "feat/x", BaseRef: "main"}

	if err := store.Save(payload, []byte{0x89, 0x50, 0x4e, 0x47}); err != nil {
		t.Fatalf("expected the run to be stored, got %v", err)
	}

	for _, name := range []string{payload.RunID + ".json", payload.RunID + ".png"} {
		info, err := os.Stat(filepath.Join(store.Directory(), name))
		if err != nil {
			t.Fatalf("expected %s to exist, got %v", name, err)
		}
		if info.Mode().Perm() != runStoreFileMode {
			t.Fatalf("expected mode %o for %s, got %o", runStoreFileMode, name, info.Mode().Perm())
		}
	}

	screenshot, err := store.LoadScreenshot(payload.RunID)
	if err != nil {
		t.Fatalf("expected the screenshot to load, got %v", err)
	}
	if len(screenshot) != 4 {
		t.Fatalf("expected the stored bytes, got %+v", screenshot)
	}
}

func TestRunStoreReportsMissingScreenshots(t *testing.T) {
	store := NewRunStore(t.TempDir())

	_, err := store.LoadScreenshot("00000000-0000-4000-8000-000000000001")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected a missing file error, got %v", err)
	}
}

func TestRunStoreRejectsUnsafeRunIDs(t *testing.T) {
	store := NewRunStore(t.TempDir())

	for _, runID := range []string{"", "../secret", "a/b", `a\b`, ".."} {
		if _, err := store.LoadScreenshot(runID); err == nil || errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %q to be rejected, got %v", runID, err)
		}
	}
}

func TestRunStoreWithoutADirectoryFails(t *testing.T) {
	store := NewRunStore("")

	if err := store.Save(domain.RunPayload{RunID: "run"}, nil); err == nil {
		t.Fatal("expected a save failure without a directory")
	}
	if _, err := store.LoadScreenshot("run"); err == nil {
		t.Fatal("expected a load failure without a directory")
	}
}

func TestRunStoreKeepsTheNewestTwenty(t *testing.T) {
	store := NewRunStore(t.TempDir())
	baseTime := time.Now().Add(-time.Hour)

	for index := 0; index < 25; index++ {
		runID := fmt.Sprintf("00000000-0000-4000-8000-%012d", index)
		jsonPath := filepath.Join(store.Directory(), runID+".json")
		pngPath := filepath.Join(store.Directory(), runID+".png")
		if err := os.WriteFile(jsonPath, []byte("{}"), runStoreFileMode); err != nil {
			t.Fatalf("cannot write the metadata fixture: %v", err)
		}
		if err := os.WriteFile(pngPath, []byte("png"), runStoreFileMode); err != nil {
			t.Fatalf("cannot write the screenshot fixture: %v", err)
		}
		modTime := baseTime.Add(time.Duration(index) * time.Minute)
		if err := os.Chtimes(jsonPath, modTime, modTime); err != nil {
			t.Fatalf("cannot set the metadata time: %v", err)
		}
		if err := os.Chtimes(pngPath, modTime, modTime); err != nil {
			t.Fatalf("cannot set the screenshot time: %v", err)
		}
	}

	if err := store.Save(domain.RunPayload{RunID: "00000000-0000-4000-8000-000000000025"}, []byte("png")); err != nil {
		t.Fatalf("expected the new run to be stored, got %v", err)
	}

	entries, err := os.ReadDir(store.Directory())
	if err != nil {
		t.Fatalf("cannot read the run store: %v", err)
	}
	if len(entries) != RunStoreLimit*2 {
		t.Fatalf("expected %d files, got %d", RunStoreLimit*2, len(entries))
	}

	for index := 0; index < 6; index++ {
		prunedID := fmt.Sprintf("00000000-0000-4000-8000-%012d", index)
		if _, statErr := os.Stat(filepath.Join(store.Directory(), prunedID+".json")); !os.IsNotExist(statErr) {
			t.Fatalf("expected the oldest run %s to be pruned", prunedID)
		}
		if _, statErr := os.Stat(filepath.Join(store.Directory(), prunedID+".png")); !os.IsNotExist(statErr) {
			t.Fatalf("expected the oldest screenshot %s to be pruned", prunedID)
		}
	}
}

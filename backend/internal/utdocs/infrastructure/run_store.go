package infrastructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lunar/backend/internal/utdocs/domain"
)

const (
	RunStoreLimit = 20

	runStoreDirectoryMode = 0o700
	runStoreFileMode      = 0o600
)

type RunStore struct {
	directory string
}

func NewRunStore(directory string) *RunStore {
	return &RunStore{directory: directory}
}

func (s *RunStore) Directory() string {
	return s.directory
}

func (s *RunStore) Save(payload domain.RunPayload, screenshot []byte) error {
	if strings.TrimSpace(s.directory) == "" {
		return errors.New("the run store directory is not configured")
	}
	if err := os.MkdirAll(s.directory, runStoreDirectoryMode); err != nil {
		return fmt.Errorf("create the run store: %w", err)
	}

	metadata, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode the run metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.directory, payload.RunID+".json"), metadata, runStoreFileMode); err != nil {
		return fmt.Errorf("write the run metadata: %w", err)
	}
	if len(screenshot) > 0 {
		if err := os.WriteFile(filepath.Join(s.directory, payload.RunID+".png"), screenshot, runStoreFileMode); err != nil {
			return fmt.Errorf("write the run screenshot: %w", err)
		}
	}

	return s.prune()
}

func (s *RunStore) LoadScreenshot(runID string) ([]byte, error) {
	if strings.TrimSpace(s.directory) == "" {
		return nil, errors.New("the run store directory is not configured")
	}
	if runID == "" || strings.ContainsAny(runID, `/\`) || strings.Contains(runID, "..") {
		return nil, errors.New("the run id is invalid")
	}

	contents, err := os.ReadFile(filepath.Join(s.directory, runID+".png"))
	if err != nil {
		return nil, fmt.Errorf("read the run screenshot: %w", err)
	}
	return contents, nil
}

func (s *RunStore) prune() error {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("read the run store: %w", err)
	}

	type storedRun struct {
		runID   string
		modTime time.Time
	}
	storedRuns := make([]storedRun, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("read the run metadata info: %w", infoErr)
		}
		storedRuns = append(storedRuns, storedRun{
			runID:   strings.TrimSuffix(entry.Name(), ".json"),
			modTime: info.ModTime(),
		})
	}

	sort.Slice(storedRuns, func(left int, right int) bool {
		if storedRuns[left].modTime.Equal(storedRuns[right].modTime) {
			return storedRuns[left].runID > storedRuns[right].runID
		}
		return storedRuns[left].modTime.After(storedRuns[right].modTime)
	})

	for index := RunStoreLimit; index < len(storedRuns); index++ {
		for _, suffix := range []string{".json", ".png"} {
			path := filepath.Join(s.directory, storedRuns[index].runID+suffix)
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return fmt.Errorf("prune the run store: %w", removeErr)
			}
		}
	}
	return nil
}

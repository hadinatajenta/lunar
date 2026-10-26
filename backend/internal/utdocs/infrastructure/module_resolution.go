package infrastructure

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const moduleResolutionFlag = "-mod=mod"

var moduleResolutionMarkers = []string{
	"updates to go.mod needed",
	"missing go.sum entry",
	"to add it:",
	"to update it:",
}

func needsModuleResolution(output string) bool {
	for _, marker := range moduleResolutionMarkers {
		if strings.Contains(output, marker) {
			return true
		}
	}
	return false
}

func withModuleResolution(arguments []string) []string {
	resolved := make([]string, 0, len(arguments)+1)
	for _, argument := range arguments {
		resolved = append(resolved, argument)
		if argument == "test" {
			resolved = append(resolved, moduleResolutionFlag)
		}
	}
	return resolved
}

type moduleFileSnapshot struct {
	path    string
	content []byte
	existed bool
}

func snapshotModuleFiles(repositoryPath string) func() error {
	names := []string{"go.mod", "go.sum"}
	snapshots := make([]moduleFileSnapshot, 0, len(names))
	for _, name := range names {
		fullPath := filepath.Join(repositoryPath, name)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			snapshots = append(snapshots, moduleFileSnapshot{path: fullPath})
			continue
		}
		snapshots = append(snapshots, moduleFileSnapshot{path: fullPath, content: content, existed: true})
	}

	return func() error {
		var firstError error
		for _, snapshot := range snapshots {
			if !snapshot.existed {
				if err := os.Remove(snapshot.path); err != nil && !errors.Is(err, os.ErrNotExist) {
					if firstError == nil {
						firstError = err
					}
				}
				continue
			}

			current, err := os.ReadFile(snapshot.path)
			if err == nil && bytes.Equal(current, snapshot.content) {
				continue
			}
			if err := os.WriteFile(snapshot.path, snapshot.content, 0o644); err != nil && firstError == nil {
				firstError = err
			}
		}
		return firstError
	}
}

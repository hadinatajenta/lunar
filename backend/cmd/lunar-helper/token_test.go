package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestLoadOrCreateTokenCreatesRestrictedFiles(t *testing.T) {
	directory := filepath.Join(t.TempDir(), helperDirectoryName)
	tokenPath := filepath.Join(directory, helperTokenFileName)

	token, err := loadOrCreateToken(tokenPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedLength := len(hex.EncodeToString(make([]byte, helperTokenBytes)))
	if !regexp.MustCompile(fmt.Sprintf("^[0-9a-f]{%d}$", expectedLength)).MatchString(token) {
		t.Fatalf("expected a %d character hexadecimal token, got %q", expectedLength, token)
	}

	fileInfo, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if fileInfo.Mode().Perm() != helperTokenFileMode {
		t.Fatalf("expected token file mode %o, got %o", helperTokenFileMode, fileInfo.Mode().Perm())
	}

	directoryInfo, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("stat helper directory: %v", err)
	}
	if directoryInfo.Mode().Perm() != helperDirectoryMode {
		t.Fatalf("expected helper directory mode %o, got %o", helperDirectoryMode, directoryInfo.Mode().Perm())
	}

	reloaded, err := loadOrCreateToken(tokenPath)
	if err != nil {
		t.Fatalf("expected no error on the second load, got %v", err)
	}
	if reloaded != token {
		t.Fatalf("expected the stored token %q, got %q", token, reloaded)
	}
}

func TestLoadOrCreateTokenTightensExistingPermissions(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), helperTokenFileName)
	if err := os.WriteFile(tokenPath, []byte("existing-token\n"), 0o644); err != nil {
		t.Fatalf("write existing token: %v", err)
	}

	token, err := loadOrCreateToken(tokenPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "existing-token" {
		t.Fatalf("expected the existing token, got %q", token)
	}

	fileInfo, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if fileInfo.Mode().Perm() != helperTokenFileMode {
		t.Fatalf("expected token file mode %o, got %o", helperTokenFileMode, fileInfo.Mode().Perm())
	}
}

func TestLoadOrCreateTokenRejectsEmptyFile(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), helperTokenFileName)
	if err := os.WriteFile(tokenPath, []byte("  \n"), 0o600); err != nil {
		t.Fatalf("write empty token: %v", err)
	}

	if _, err := loadOrCreateToken(tokenPath); err == nil {
		t.Fatal("expected an error for an empty token file")
	}
}

func TestLoadOrCreateTokenGeneratesDistinctTokens(t *testing.T) {
	firstPath := filepath.Join(t.TempDir(), helperTokenFileName)
	secondPath := filepath.Join(t.TempDir(), helperTokenFileName)

	firstToken, err := loadOrCreateToken(firstPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	secondToken, err := loadOrCreateToken(secondPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if firstToken == secondToken {
		t.Fatal("expected two independently generated tokens to differ")
	}
}

func TestHelperTokenPathUsesHomeDirectory(t *testing.T) {
	homeDirectory := t.TempDir()
	t.Setenv("HOME", homeDirectory)

	tokenPath, err := helperTokenPath()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPath := filepath.Join(homeDirectory, helperDirectoryName, helperTokenFileName)
	if tokenPath != expectedPath {
		t.Fatalf("expected %q, got %q", expectedPath, tokenPath)
	}
}

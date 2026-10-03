package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	helperDirectoryName = ".lunar"
	helperTokenFileName = "helper-token"
	helperDirectoryMode = 0o700
	helperTokenFileMode = 0o600
	helperTokenBytes    = 32
)

var errEmptyHelperToken = errors.New("helper token file is empty")

func helperTokenPath() (string, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(homeDirectory, helperDirectoryName, helperTokenFileName), nil
}

func loadOrCreateToken(tokenPath string) (string, error) {
	token, err := readToken(tokenPath)
	if err == nil {
		return token, nil
	}

	if !os.IsNotExist(err) {
		return "", err
	}
	return createToken(tokenPath)
}

func readToken(tokenPath string) (string, error) {
	contents, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(string(contents))
	if token == "" {
		return "", fmt.Errorf("%w: %s", errEmptyHelperToken, tokenPath)
	}

	if err := os.Chmod(tokenPath, helperTokenFileMode); err != nil {
		return "", fmt.Errorf("restrict token file permissions: %w", err)
	}
	return token, nil
}

func createToken(tokenPath string) (string, error) {
	directory := filepath.Dir(tokenPath)
	if err := os.MkdirAll(directory, helperDirectoryMode); err != nil {
		return "", fmt.Errorf("create helper directory: %w", err)
	}

	if err := os.Chmod(directory, helperDirectoryMode); err != nil {
		return "", fmt.Errorf("restrict helper directory permissions: %w", err)
	}

	buffer := make([]byte, helperTokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate helper token: %w", err)
	}
	token := hex.EncodeToString(buffer)

	if err := os.WriteFile(tokenPath, []byte(token), helperTokenFileMode); err != nil {
		return "", fmt.Errorf("write helper token: %w", err)
	}

	if err := os.Chmod(tokenPath, helperTokenFileMode); err != nil {
		return "", fmt.Errorf("restrict token file permissions: %w", err)
	}
	return token, nil
}

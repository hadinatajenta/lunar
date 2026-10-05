package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelperCheckoutSwitchesBranch(t *testing.T) {
	rootPath, repositoryPath := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := doRequest(t, http.MethodPost, helperGitURL(server, checkoutPath, rootPath), `{"branch":"main"}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, readBody(t, response))
	}

	checkout := decodeHelperBody[checkoutResponse](t, response)
	if checkout.Branch != "main" {
		t.Fatalf("expected the response branch main, got %q", checkout.Branch)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "main" {
		t.Fatalf("expected the repository to switch to main, got %q", currentBranch)
	}
}

func TestHelperCheckoutRefusesDirtyTreeWithConflict(t *testing.T) {
	rootPath, repositoryPath := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	writeHelperFile(t, filepath.Join(repositoryPath, "README.md"), "dirty\n")

	response := doRequest(t, http.MethodPost, helperGitURL(server, checkoutPath, rootPath), `{"branch":"main"}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", response.StatusCode, readBody(t, response))
	}

	message := decodeHelperError(t, response)
	if !strings.Contains(message, "uncommitted changes") {
		t.Fatalf("expected a typed dirty tree message, got %q", message)
	}
	if strings.Contains(message, rootPath) {
		t.Fatalf("expected no absolute filesystem path in the error, got %q", message)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "feature/second" {
		t.Fatalf("expected the refused checkout to stay on feature/second, got %q", currentBranch)
	}
}

func TestHelperCheckoutRejectsUnsafeBranchName(t *testing.T) {
	rootPath, _ := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	response := doRequest(t, http.MethodPost, helperGitURL(server, checkoutPath, rootPath), `{"branch":"-x"}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.StatusCode, readBody(t, response))
	}

	message := decodeHelperError(t, response)
	if !strings.Contains(message, "branch name is invalid") {
		t.Fatalf("expected a typed branch name message, got %q", message)
	}
}

func TestHelperCheckoutAllowsUntrackedOnlyChanges(t *testing.T) {
	rootPath, repositoryPath := newHelperScratchRepository(t)
	server := newHelperGitServer(t, []string{rootPath})

	untrackedPath := filepath.Join(repositoryPath, "untracked.txt")
	writeHelperFile(t, untrackedPath, "keep me\n")

	response := doRequest(t, http.MethodPost, helperGitURL(server, checkoutPath, rootPath), `{"branch":"main"}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for an untracked-only tree, got %d: %s", response.StatusCode, readBody(t, response))
	}

	checkout := decodeHelperBody[checkoutResponse](t, response)
	if checkout.Branch != "main" {
		t.Fatalf("expected the response branch main, got %q", checkout.Branch)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "main" {
		t.Fatalf("expected the repository to switch to main, got %q", currentBranch)
	}

	if _, err := os.Stat(untrackedPath); err != nil {
		t.Fatalf("expected the untracked file to survive the checkout: %v", err)
	}
}

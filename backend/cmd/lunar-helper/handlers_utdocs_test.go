package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	utdocsapplication "lunar/backend/internal/utdocs/application"
	utdocsdomain "lunar/backend/internal/utdocs/domain"
	utdocsinfrastructure "lunar/backend/internal/utdocs/infrastructure"
	"lunar/backend/internal/workspace/infrastructure"
)

var utRunIDExtractionPattern = regexp.MustCompile(`"run_id":"([0-9a-f-]{36})"`)

type stubUTDocsRunner struct {
	result    utdocsdomain.RunResult
	tests     []utdocsdomain.TestResult
	output    string
	err       error
	calls     int
	packages  []string
	functions []string
}

func (r *stubUTDocsRunner) Run(_ context.Context, _ string, packages []string, testFunctions []string) (utdocsdomain.RunResult, []utdocsdomain.TestResult, string, error) {
	r.calls++
	r.packages = append([]string{}, packages...)
	r.functions = append([]string{}, testFunctions...)
	if r.err != nil {
		return utdocsdomain.RunResult{}, nil, "", r.err
	}
	return r.result, r.tests, r.output, nil
}

func writeUTDocsFile(t *testing.T, fullPath string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("cannot create the fixture directory for %s: %v", fullPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the fixture file %s: %v", fullPath, err)
	}
}

func newUTDocsScratchRepository(t *testing.T) (string, string) {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "micro-payment")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/repo/dao.go"), "package repo\n\nfunc Find() {}\n")
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/repo/dao_test.go"), "package repo\n\nfunc TestFind(t *testing.T) {}\n")

	runGitTool(t, repositoryPath, "init", "-q", "-b", "development")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feat/CRMMS-123")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/repo/dao.go"), "package repo\n\nfunc Find() int { return 1 }\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "feature change")
	runGitTool(t, repositoryPath, "checkout", "-q", "development")

	return rootPath, repositoryPath
}

func newUTDocsBlockedScratchRepository(t *testing.T) (string, string) {
	t.Helper()
	requireGitTool(t)

	rootPath := t.TempDir()
	repositoryPath := filepath.Join(rootPath, "micro-payment")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/seed/seed.go"), "package seed\n\nfunc Seed() {}\n")
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/seed/seed_test.go"), "package seed\n\nfunc TestSeed(t *testing.T) {}\n")
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/seed/reset_test.go"), "package seed\n\nfunc TestReset(t *testing.T) {\n\tTRUNCATE TABLE users\n}\n")

	runGitTool(t, repositoryPath, "init", "-q", "-b", "development")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "initial commit")
	runGitTool(t, repositoryPath, "checkout", "-q", "-b", "feat/CRMMS-123")

	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/seed/seed.go"), "package seed\n\nfunc Seed() int { return 1 }\n")
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/seed/reset_test.go"), "package seed\n\nfunc TestResetTables(t *testing.T) {\n\tTRUNCATE TABLE users\n}\n")
	runGitTool(t, repositoryPath, "add", "-A")
	runGitTool(t, repositoryPath, "commit", "-q", "-m", "feature change")
	runGitTool(t, repositoryPath, "checkout", "-q", "development")

	return rootPath, repositoryPath
}

func newUTDocsTestHandler(t *testing.T, allowedRoots []string, runner utdocsapplication.TestRunner) *utdocsHandler {
	t.Helper()

	return &utdocsHandler{
		allowedRoots: allowedRoots,
		operator:     infrastructure.NewGitInspector(allowedRoots),
		restoreStash: utdocsinfrastructure.RestoreStash,
		logger:       discardLogger(),
		service: utdocsapplication.NewRunService(
			utdocsinfrastructure.NewScanner(),
			runner,
			utdocsinfrastructure.NewScreenshotRenderer(),
		),
		store: utdocsinfrastructure.NewRunStore(t.TempDir()),
	}
}

func newUTDocsRunBody(rootPath string, branch string, baseRef string) string {
	return fmt.Sprintf(
		`{"root":%q,"repo":"micro-payment","branch":%q,"base_ref":%q,"model":"DeepSeek-V4 Flash"}`,
		rootPath,
		branch,
		baseRef,
	)
}

func performUTDocsRequest(t *testing.T, mux *http.ServeMux, method string, target string, body string, withToken bool) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}

	request := httptest.NewRequest(method, target, bodyReader)
	if withToken {
		request.Header.Set(authorizationHeader, bearerScheme+" "+testHelperToken)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func TestHelperUTRunStreamsPhasesAndStoresTheScreenshot(t *testing.T) {
	rootPath, repositoryPath := newUTDocsScratchRepository(t)
	runner := &stubUTDocsRunner{
		result: utdocsdomain.RunResult{Status: utdocsdomain.RunStatusPassed, Total: 1, Passed: 1, DurationMs: 120},
		tests:  []utdocsdomain.TestResult{{Name: "TestFind", Package: "lunar/backend/internal/repo", File: "internal/repo/dao_test.go", Status: utdocsdomain.TestStatusPassed, DurationMs: 120}},
		output: "ok  \tlunar/backend/internal/repo\t0.120s\n",
	}
	handler := newUTDocsTestHandler(t, []string{rootPath}, runner)
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	recorder := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, newUTDocsRunBody(rootPath, "feat/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}

	stream := recorder.Body.String()
	for _, marker := range []string{
		"event: phase",
		`"phase":"scan"`,
		`"phase":"safety"`,
		`"phase":"run"`,
		`"phase":"screenshot"`,
		"event: output",
		"event: done",
		`"run_id":"`,
		`"screenshot_url":"/ut-runs/`,
		`"status":"passed"`,
		`"executed_packages":["./internal/repo"]`,
		`"classification":"safe"`,
		`"tests":[{"name":"TestFind"`,
		`"untested_functions":[{"name":"Find","file":"internal/repo/dao.go","package":"./internal/repo"}]`,
		`"stashed":false`,
		`"restored":true`,
	} {
		if !strings.Contains(stream, marker) {
			t.Fatalf("expected the stream to contain %s, received %s", marker, stream)
		}
	}
	if runner.calls != 1 || len(runner.packages) != 1 || runner.packages[0] != "./internal/repo" {
		t.Fatalf("expected the safe package to run, got %+v", runner.packages)
	}
	if len(runner.functions) != 1 || runner.functions[0] != "TestFind" {
		t.Fatalf("expected the changed function to run, got %+v", runner.functions)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected the original branch to be restored after the run, got %q", currentBranch)
	}

	entries, err := os.ReadDir(handler.store.Directory())
	if err != nil {
		t.Fatalf("cannot read the run store: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected the metadata and the screenshot, got %+v", entries)
	}
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			t.Fatalf("cannot read the stored file info: %v", infoErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("expected mode 600 for %s, got %o", entry.Name(), info.Mode().Perm())
		}
	}

	matches := utRunIDExtractionPattern.FindStringSubmatch(stream)
	if len(matches) != 2 {
		t.Fatalf("expected a run id in the stream, received %s", stream)
	}

	screenshot := performUTDocsRequest(t, mux, http.MethodGet, "/ut-runs/"+matches[1]+"/screenshot.png", "", true)
	if screenshot.Code != http.StatusOK {
		t.Fatalf("expected 200 for the screenshot, got %d with body %s", screenshot.Code, screenshot.Body.String())
	}
	if contentType := screenshot.Header().Get("Content-Type"); contentType != "image/png" {
		t.Fatalf("expected image/png, got %q", contentType)
	}
	if !bytes.HasPrefix(screenshot.Body.Bytes(), []byte{0x89, 0x50, 0x4e, 0x47}) {
		t.Fatal("expected the stored screenshot to be a PNG")
	}
}

func TestHelperUTRunBlocksPackagesWithDestructiveTests(t *testing.T) {
	rootPath, _ := newUTDocsBlockedScratchRepository(t)
	runner := &stubUTDocsRunner{}
	handler := newUTDocsTestHandler(t, []string{rootPath}, runner)
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	recorder := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, newUTDocsRunBody(rootPath, "feat/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	stream := recorder.Body.String()
	if runner.calls != 0 {
		t.Fatalf("expected the runner not to be called, got %d calls", runner.calls)
	}
	for _, marker := range []string{
		`"status":"blocked"`,
		"internal/seed/reset_test.go",
		"contains destructive or database markers: TRUNCATE",
		"does not run because the package contains a blocked test file",
	} {
		if !strings.Contains(stream, marker) {
			t.Fatalf("expected the stream to contain %s, received %s", marker, stream)
		}
	}
}

func TestHelperUTRunRefusesWithoutAllowedRoots(t *testing.T) {
	handler := newUTDocsTestHandler(t, nil, &stubUTDocsRunner{})
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	rootPath := t.TempDir()
	recorder := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, newUTDocsRunBody(rootPath, "feat/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d with body %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), messageWriteAllowlist) {
		t.Fatalf("expected the allowlist message, got %s", recorder.Body.String())
	}
}

func TestHelperUTRunStashesAndRestoresADirtyWorktree(t *testing.T) {
	rootPath, repositoryPath := newUTDocsScratchRepository(t)
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/repo/dao.go"), "package repo\n\nfunc Find() string { return \"dirty\" }\n")

	runner := &stubUTDocsRunner{result: utdocsdomain.RunResult{Status: utdocsdomain.RunStatusPassed, Total: 1, Passed: 1}}
	handler := newUTDocsTestHandler(t, []string{rootPath}, runner)
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	recorder := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, newUTDocsRunBody(rootPath, "feat/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	stream := recorder.Body.String()
	if !strings.Contains(stream, `"stashed":true`) || !strings.Contains(stream, `"restored":true`) {
		t.Fatalf("expected the stash and restore flags, got %s", stream)
	}
	if runner.calls != 1 {
		t.Fatalf("expected the run to execute after the stash, got %d calls", runner.calls)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected the original branch to be restored, got %q", currentBranch)
	}
	content, err := os.ReadFile(filepath.Join(repositoryPath, "internal/repo/dao.go"))
	if err != nil {
		t.Fatalf("cannot read the restored file: %v", err)
	}
	if !strings.Contains(string(content), "dirty") {
		t.Fatalf("expected the stashed change to be restored, got %q", string(content))
	}
}

func TestHelperUTRunReportsARestoreFailure(t *testing.T) {
	rootPath, repositoryPath := newUTDocsScratchRepository(t)
	writeUTDocsFile(t, filepath.Join(repositoryPath, "internal/repo/dao.go"), "package repo\n\nfunc Find() string { return \"dirty\" }\n")

	runner := &stubUTDocsRunner{result: utdocsdomain.RunResult{Status: utdocsdomain.RunStatusPassed, Total: 1, Passed: 1}}
	handler := newUTDocsTestHandler(t, []string{rootPath}, runner)
	handler.restoreStash = func(context.Context, string) error {
		return errors.New("the stash pop failed")
	}
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	recorder := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, newUTDocsRunBody(rootPath, "feat/CRMMS-123", "development"), true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	stream := recorder.Body.String()
	if !strings.Contains(stream, `"stashed":true`) || !strings.Contains(stream, `"restored":false`) {
		t.Fatalf("expected the stash and failed restore flags, got %s", stream)
	}
	if !strings.Contains(stream, messageUTRunRestoreFailed) {
		t.Fatalf("expected the restore failure event, got %s", stream)
	}

	currentBranch := strings.TrimSpace(runGitTool(t, repositoryPath, "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "development" {
		t.Fatalf("expected the original branch to be restored even on a stash failure, got %q", currentBranch)
	}
}

func TestHelperUTRunRejectsInvalidBodies(t *testing.T) {
	rootPath, _ := newUTDocsScratchRepository(t)
	handler := newUTDocsTestHandler(t, []string{rootPath}, &stubUTDocsRunner{})
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	testCases := []struct {
		name    string
		body    string
		message string
	}{
		{name: "malformed json", body: "{", message: messageUTRunInvalidBody},
		{name: "unknown field", body: `{"root":"/tmp","unknown":true}`, message: messageUTRunInvalidBody},
		{name: "missing fields", body: `{"root":"/tmp"}`, message: messageUTRunFieldsRequired},
		{name: "invalid branch", body: newUTDocsRunBody(rootPath, "-bad", "development"), message: messageBranchNameInvalid},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, testCase.body, true)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d with body %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), testCase.message) {
				t.Fatalf("expected %q, got %s", testCase.message, recorder.Body.String())
			}
		})
	}
}

func TestHelperUTRunRoutesRequireTheToken(t *testing.T) {
	rootPath, _ := newUTDocsScratchRepository(t)
	handler := newUTDocsTestHandler(t, []string{rootPath}, &stubUTDocsRunner{})
	mux := http.NewServeMux()
	registerUTDocsHandler(mux, testHelperToken, handler)

	unauthorizedRun := performUTDocsRequest(t, mux, http.MethodPost, utRunsPath, newUTDocsRunBody(rootPath, "feat/CRMMS-123", "development"), false)
	if unauthorizedRun.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without the token, got %d", unauthorizedRun.Code)
	}

	unauthorizedScreenshot := performUTDocsRequest(t, mux, http.MethodGet, "/ut-runs/00000000-0000-0000-0000-000000000000/screenshot.png", "", false)
	if unauthorizedScreenshot.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without the token, got %d", unauthorizedScreenshot.Code)
	}

	invalidID := performUTDocsRequest(t, mux, http.MethodGet, "/ut-runs/not-a-run-id/screenshot.png", "", true)
	if invalidID.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for an invalid run id, got %d with body %s", invalidID.Code, invalidID.Body.String())
	}
	if !strings.Contains(invalidID.Body.String(), messageUTRunScreenshotInvalidID) {
		t.Fatalf("expected the invalid id message, got %s", invalidID.Body.String())
	}

	missing := performUTDocsRequest(t, mux, http.MethodGet, "/ut-runs/00000000-0000-0000-0000-000000000000/screenshot.png", "", true)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a missing screenshot, got %d", missing.Code)
	}
}

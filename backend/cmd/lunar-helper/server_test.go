package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"lunar/backend/internal/workspace/domain"
	"lunar/backend/internal/workspace/infrastructure"
)

const testHelperToken = "test-helper-token-0123456789abcdef"

type stubGitReader struct {
	summaries   []domain.RepositorySummary
	details     []domain.RepositoryDetail
	listErr     error
	readErr     error
	validateErr error
}

func (s stubGitReader) ListRepositories(_ context.Context, _ string) ([]domain.RepositorySummary, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.summaries, nil
}

func (s stubGitReader) ReadRepositories(_ context.Context, _ string, _ []string) ([]domain.RepositoryDetail, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.details, nil
}

func (s stubGitReader) ValidateRoot(_ context.Context, _ string) error {
	return s.validateErr
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newStubHelperServer(t *testing.T, reader domain.GitReader, allowedOrigins []string) *httptest.Server {
	t.Helper()

	handler, stop := newHelperHandler(helperDependencies{
		reader:         reader,
		token:          testHelperToken,
		allowedOrigins: allowedOrigins,
		version:        "test-version",
		logger:         discardLogger(),
	})
	t.Cleanup(stop)

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func doRequest(t *testing.T, method string, requestURL string, body string, authorization string, origin string) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	request, err := http.NewRequest(method, requestURL, bodyReader)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if authorization != "" {
		request.Header.Set(authorizationHeader, authorization)
	}
	if origin != "" {
		request.Header.Set(originHeader, origin)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	})
	return response
}

func decodeBody(t *testing.T, response *http.Response) map[string]any {
	t.Helper()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	decoded := map[string]any{}
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("decode response body %q: %v", contents, err)
	}
	return decoded
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return string(contents)
}

func TestHelperHealthSucceedsWithoutToken(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{}, nil)

	response := doRequest(t, http.MethodGet, server.URL+healthPath, "", "", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from %s, got %d", healthPath, response.StatusCode)
	}

	decoded := decodeBody(t, response)
	if decoded["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", decoded["status"])
	}
	if decoded["version"] != "test-version" {
		t.Fatalf("expected version test-version, got %v", decoded["version"])
	}
}

func TestHelperTokenEndpointReturnsToken(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{}, nil)

	response := doRequest(t, http.MethodGet, server.URL+tokenPath, "", "", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from %s, got %d", tokenPath, response.StatusCode)
	}

	decoded := decodeBody(t, response)
	if decoded["token"] != testHelperToken {
		t.Fatalf("expected the helper token, got %v", decoded["token"])
	}
}

func TestHelperTokenEndpointSupportsBrowserPreflight(t *testing.T) {
	origins := []string{"http://localhost:5174"}
	server := newStubHelperServer(t, stubGitReader{}, origins)

	response := doRequest(t, http.MethodOptions, server.URL+tokenPath, "", "", "http://localhost:5174")
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight, got %d", response.StatusCode)
	}
	if got := response.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5174" {
		t.Fatalf("expected the allowed origin echoed back, got %q", got)
	}
}

func TestHelperTokenEndpointRejectsUnlistedOrigin(t *testing.T) {
	origins := []string{"http://localhost:5174"}
	server := newStubHelperServer(t, stubGitReader{}, origins)

	response := doRequest(t, http.MethodGet, server.URL+tokenPath, "", "", "http://evil.example")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for an unlisted origin, got %d", response.StatusCode)
	}
}

func TestHelperProtectedEndpointsRequireToken(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{}, nil)

	protectedRequests := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: repositoriesPath + "?root=/workspace"},
		{method: http.MethodGet, path: servicesPath + "?root=/workspace&names=everest"},
		{method: http.MethodPost, path: validatePathPath},
	}

	authorizationValues := []struct {
		name  string
		value string
	}{
		{name: "missing header", value: ""},
		{name: "wrong token equal length", value: bearerScheme + " " + strings.Repeat("f", len(testHelperToken))},
		{name: "wrong token shorter", value: bearerScheme + " short"},
		{name: "missing scheme", value: testHelperToken},
		{name: "wrong scheme", value: "Basic " + testHelperToken},
		{name: "empty bearer value", value: bearerScheme + " "},
		{name: "blank header", value: "   "},
	}

	for _, protectedRequest := range protectedRequests {
		for _, authorization := range authorizationValues {
			testName := protectedRequest.method + " " + protectedRequest.path + " " + authorization.name
			t.Run(testName, func(t *testing.T) {
				response := doRequest(t, protectedRequest.method, server.URL+protectedRequest.path, "{}", authorization.value, "")
				if response.StatusCode != http.StatusUnauthorized {
					t.Fatalf("expected 401, got %d", response.StatusCode)
				}

				decoded := decodeBody(t, response)
				if decoded["error"] != messageUnauthorized {
					t.Fatalf("expected unauthorized error, got %v", decoded["error"])
				}
			})
		}
	}
}

func TestHelperAcceptsCorrectToken(t *testing.T) {
	reader := stubGitReader{
		summaries: []domain.RepositorySummary{
			{Name: "everest", IsGit: true, Branch: "development", DirtyCount: 2, UpdatedRelative: "5 minutes ago"},
			{Name: "way4", IsGit: false, Error: "not a git repository"},
		},
		details: []domain.RepositoryDetail{
			{
				Name:            "everest",
				Branch:          "development",
				UpdatedRelative: "5 minutes ago",
				Commit:          domain.CommitInfo{Hash: "88b4ca3", Author: "Hadinata Jenta", RelativeTime: "5 minutes ago", Subject: "fix: guard nil session"},
			},
		},
	}
	server := newStubHelperServer(t, reader, nil)

	listingResponse := doRequest(t, http.MethodGet, server.URL+repositoriesPath+"?root=/workspace", "", bearerScheme+" "+testHelperToken, "")
	if listingResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", listingResponse.StatusCode)
	}
	repositoriesBody := readBody(t, listingResponse)
	if !strings.Contains(repositoriesBody, `"root_path":"/workspace"`) {
		t.Fatalf("expected the requested root in the response, got %s", repositoriesBody)
	}
	if !strings.Contains(repositoriesBody, `"is_git":true`) {
		t.Fatalf("expected a git repository in the response, got %s", repositoriesBody)
	}
	if !strings.Contains(repositoriesBody, `"error":"not a git repository"`) {
		t.Fatalf("expected the plain folder error in the response, got %s", repositoriesBody)
	}

	detailResponse := doRequest(t, http.MethodGet, server.URL+servicesPath+"?root=/workspace&names=everest,%20way4", "", bearerScheme+" "+testHelperToken, "")
	if detailResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", detailResponse.StatusCode)
	}
	servicesBody := readBody(t, detailResponse)
	if !strings.Contains(servicesBody, `"files":[]`) {
		t.Fatalf("expected an empty files array instead of null, got %s", servicesBody)
	}
	if !strings.Contains(servicesBody, `"hash":"88b4ca3"`) {
		t.Fatalf("expected the commit hash in the response, got %s", servicesBody)
	}

	validateResponse := doRequest(t, http.MethodPost, server.URL+validatePathPath, `{"root_path":"/workspace"}`, bearerScheme+" "+testHelperToken, "")
	if validateResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", validateResponse.StatusCode)
	}
	decoded := decodeBody(t, validateResponse)
	if decoded["valid"] != true {
		t.Fatalf("expected valid true, got %v", decoded["valid"])
	}
	if decoded["reason"] != "" {
		t.Fatalf("expected an empty reason, got %v", decoded["reason"])
	}
}

func TestHelperValidatePathReportsInvalidRoot(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{validateErr: domain.ErrRootNotFound}, nil)

	response := doRequest(t, http.MethodPost, server.URL+validatePathPath, `{"root_path":"/workspace/missing"}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	decoded := decodeBody(t, response)
	if decoded["valid"] != false {
		t.Fatalf("expected valid false, got %v", decoded["valid"])
	}
	if decoded["reason"] != messageRootMissing {
		t.Fatalf("expected %q, got %v", messageRootMissing, decoded["reason"])
	}
}

func TestHelperValidatePathRejectsMalformedBody(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{}, nil)

	cases := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "truncated json", body: "{"},
		{name: "unknown field", body: `{"path":"/workspace/secret"}`},
		{name: "wrong field type", body: `{"root_path":12}`},
		{name: "array body", body: `[]`},
		{name: "trailing content", body: `{"root_path":"/workspace"}{"root_path":"/other"}`},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := doRequest(t, http.MethodPost, server.URL+validatePathPath, testCase.body, bearerScheme+" "+testHelperToken, "")
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", response.StatusCode)
			}

			contents := readBody(t, response)
			if strings.Contains(contents, "secret") || strings.Contains(contents, "/workspace") {
				t.Fatalf("expected no request data in the error, got %s", contents)
			}
			if !strings.Contains(contents, messageInvalidRequestBody) {
				t.Fatalf("expected %q, got %s", messageInvalidRequestBody, contents)
			}
		})
	}
}

func TestHelperCORSAllowsConfiguredOrigin(t *testing.T) {
	allowedOrigin := "http://localhost:5174"
	server := newStubHelperServer(t, stubGitReader{}, []string{allowedOrigin})

	response := doRequest(t, http.MethodGet, server.URL+healthPath, "", "", allowedOrigin)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != allowedOrigin {
		t.Fatalf("expected the allowed origin header, got %q", response.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestHelperCORSRejectsUnlistedOrigin(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{}, []string{"http://localhost:5174"})

	response := doRequest(t, http.MethodGet, server.URL+healthPath, "", "", "http://evil.example")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.StatusCode)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no allow-origin header, got %q", response.Header.Get("Access-Control-Allow-Origin"))
	}

	decoded := decodeBody(t, response)
	if decoded["error"] != messageOriginNotAllowed {
		t.Fatalf("expected %q, got %v", messageOriginNotAllowed, decoded["error"])
	}
}

func TestHelperCORSDefaultsToLocalWebOrigins(t *testing.T) {
	origins := parseOriginList("")
	if len(origins) != len(defaultHelperOrigins) {
		t.Fatalf("expected %d default origins, got %v", len(defaultHelperOrigins), origins)
	}

	server := newStubHelperServer(t, stubGitReader{}, origins)

	for _, origin := range defaultHelperOrigins {
		response := doRequest(t, http.MethodGet, server.URL+healthPath, "", "", origin)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", origin, response.StatusCode)
		}
		if response.Header.Get("Access-Control-Allow-Origin") != origin {
			t.Fatalf("expected allow-origin %s, got %q", origin, response.Header.Get("Access-Control-Allow-Origin"))
		}
	}

	response := doRequest(t, http.MethodGet, server.URL+healthPath, "", "", "http://evil.example")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for an unlisted origin, got %d", response.StatusCode)
	}
}

func TestHelperCORSPreflightForValidatePath(t *testing.T) {
	allowedOrigin := "http://localhost:5174"
	server := newStubHelperServer(t, stubGitReader{}, []string{allowedOrigin})

	request, err := http.NewRequest(http.MethodOptions, server.URL+validatePathPath, nil)
	if err != nil {
		t.Fatalf("create preflight request: %v", err)
	}
	request.Header.Set(originHeader, allowedOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("perform preflight request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.StatusCode)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != allowedOrigin {
		t.Fatalf("expected allow-origin %s, got %q", allowedOrigin, response.Header.Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(response.Header.Get("Access-Control-Allow-Methods"), http.MethodPost) {
		t.Fatalf("expected POST in the allowed methods, got %q", response.Header.Get("Access-Control-Allow-Methods"))
	}
	if !strings.Contains(response.Header.Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("expected Authorization in the allowed headers, got %q", response.Header.Get("Access-Control-Allow-Headers"))
	}
	if !strings.Contains(response.Header.Get("Access-Control-Allow-Headers"), "Content-Type") {
		t.Fatalf("expected Content-Type in the allowed headers, got %q", response.Header.Get("Access-Control-Allow-Headers"))
	}
}

func TestHelperReportsMissingRootWithoutLeakingPaths(t *testing.T) {
	server := newStubHelperServer(t, stubGitReader{listErr: domain.ErrRootNotFound}, nil)

	secretRoot := "/private/tmp/lunar-secret-workspace"
	response := doRequest(t, http.MethodGet, server.URL+repositoriesPath+"?root="+url.QueryEscape(secretRoot), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", response.StatusCode)
	}

	contents := readBody(t, response)
	if strings.Contains(contents, "lunar-secret-workspace") || strings.Contains(contents, "/") {
		t.Fatalf("expected no path leak in the error, got %s", contents)
	}
	if !strings.Contains(contents, messageRootMissing) {
		t.Fatalf("expected %q, got %s", messageRootMissing, contents)
	}
}

func TestHelperReadsRealRepositoryThroughInspector(t *testing.T) {
	requireGitExecutable(t)

	root := t.TempDir()
	repositoryPath := filepath.Join(root, "everest")
	initializeGitRepository(t, repositoryPath)
	writeTestFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n")
	commitTestRepository(t, repositoryPath, "feat: initial commit")
	writeTestFile(t, filepath.Join(repositoryPath, "main.go"), "package main\n\nfunc main() {}\n")
	if err := os.MkdirAll(filepath.Join(root, "way4"), 0o755); err != nil {
		t.Fatalf("create plain directory: %v", err)
	}

	server := newStubHelperServer(t, infrastructure.NewGitInspector(nil), nil)

	response := doRequest(t, http.MethodGet, server.URL+repositoriesPath+"?root="+url.QueryEscape(root), "", bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	var listing repositoriesResponse
	if err := json.Unmarshal([]byte(readBody(t, response)), &listing); err != nil {
		t.Fatalf("decode repositories response: %v", err)
	}
	if listing.RootPath != root {
		t.Fatalf("expected root_path %q, got %q", root, listing.RootPath)
	}
	if len(listing.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %d", len(listing.Repositories))
	}

	if !listing.Repositories[0].IsGit || listing.Repositories[0].Branch != "main" {
		t.Fatalf("expected everest on branch main, got %+v", listing.Repositories[0])
	}
	if listing.Repositories[0].DirtyCount != 1 {
		t.Fatalf("expected 1 dirty file, got %d", listing.Repositories[0].DirtyCount)
	}
	if listing.Repositories[0].Error != "" {
		t.Fatalf("expected no error for everest, got %q", listing.Repositories[0].Error)
	}
	if listing.Repositories[1].IsGit {
		t.Fatalf("expected way4 to be a plain folder, got %+v", listing.Repositories[1])
	}
	if listing.Repositories[1].Error == "" {
		t.Fatal("expected an error for the plain folder")
	}

	detailResponse := doRequest(t, http.MethodGet, server.URL+servicesPath+"?root="+url.QueryEscape(root)+"&names=everest,way4", "", bearerScheme+" "+testHelperToken, "")
	if detailResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", detailResponse.StatusCode)
	}

	var services servicesResponse
	if err := json.Unmarshal([]byte(readBody(t, detailResponse)), &services); err != nil {
		t.Fatalf("decode services response: %v", err)
	}
	if len(services.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services.Services))
	}
	if services.Services[0].Name != "everest" || services.Services[0].Branch != "main" {
		t.Fatalf("expected everest detail, got %+v", services.Services[0])
	}
	if len(services.Services[0].Commit.Hash) != 7 {
		t.Fatalf("expected a 7 character commit hash, got %q", services.Services[0].Commit.Hash)
	}
	if services.Services[0].Files == nil {
		t.Fatal("expected an empty files array instead of null")
	}
	if services.Services[1].Error == "" {
		t.Fatal("expected an error for the plain folder detail")
	}
}

func TestHelperValidatePathWithRealInspector(t *testing.T) {
	server := newStubHelperServer(t, infrastructure.NewGitInspector(nil), nil)

	validRoot := t.TempDir()
	response := doRequest(t, http.MethodPost, server.URL+validatePathPath, `{"root_path":"`+validRoot+`"}`, bearerScheme+" "+testHelperToken, "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	decoded := decodeBody(t, response)
	if decoded["valid"] != true {
		t.Fatalf("expected valid true for %s, got %v", validRoot, decoded)
	}

	missingRoot := filepath.Join(validRoot, "missing")
	response = doRequest(t, http.MethodPost, server.URL+validatePathPath, `{"root_path":"`+missingRoot+`"}`, bearerScheme+" "+testHelperToken, "")
	decoded = decodeBody(t, response)
	if decoded["valid"] != false {
		t.Fatalf("expected valid false for %s, got %v", missingRoot, decoded)
	}
	reason, isString := decoded["reason"].(string)
	if !isString {
		t.Fatalf("expected a string reason, got %v", decoded["reason"])
	}
	if strings.Contains(reason, validRoot) {
		t.Fatalf("expected no path leak in the reason, got %q", reason)
	}
}

func TestResolveListenAddressForcesLoopback(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: defaultHelperAddr},
		{name: "loopback kept", input: "127.0.0.1:5199", expected: "127.0.0.1:5199"},
		{name: "wildcard forced", input: "0.0.0.0:5199", expected: "127.0.0.1:5199"},
		{name: "empty host forced", input: ":5199", expected: "127.0.0.1:5199"},
		{name: "localhost forced", input: "localhost:5199", expected: "127.0.0.1:5199"},
		{name: "private address forced", input: "192.168.1.20:5199", expected: "127.0.0.1:5199"},
		{name: "public address forced", input: "8.8.8.8:5199", expected: "127.0.0.1:5199"},
		{name: "bare port", input: "5199", expected: "127.0.0.1:5199"},
		{name: "invalid port", input: "127.0.0.1:not-a-port", expected: defaultHelperAddr},
		{name: "port out of range", input: "127.0.0.1:70000", expected: defaultHelperAddr},
		{name: "invalid address", input: "not-an-address", expected: defaultHelperAddr},
		{name: "ipv6 loopback kept", input: "[::1]:5199", expected: "[::1]:5199"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			address := resolveListenAddress(testCase.input)
			if address != testCase.expected {
				t.Fatalf("expected %q for %q, got %q", testCase.expected, testCase.input, address)
			}
		})
	}
}

func TestResolveListenAddressBindsLoopbackInterface(t *testing.T) {
	address := resolveListenAddress("0.0.0.0:0")
	if address != "127.0.0.1:0" {
		t.Fatalf("expected a forced loopback address, got %q", address)
	}

	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("listen on %s: %v", address, err)
	}
	defer listener.Close()

	tcpAddress, isTCP := listener.Addr().(*net.TCPAddr)
	if !isTCP {
		t.Fatalf("expected a TCP address, got %T", listener.Addr())
	}
	if !tcpAddress.IP.IsLoopback() {
		t.Fatalf("expected a loopback bind, got %s", tcpAddress.IP)
	}
}

func TestParseOriginListTrimsAndFallsBack(t *testing.T) {
	if origins := parseOriginList(" , "); len(origins) != len(defaultHelperOrigins) {
		t.Fatalf("expected the default origins for a blank value, got %v", origins)
	}

	origins := parseOriginList(" http://localhost:5174 , http://127.0.0.1:5174 ")
	if len(origins) != 2 || origins[0] != "http://localhost:5174" || origins[1] != "http://127.0.0.1:5174" {
		t.Fatalf("expected trimmed origins, got %v", origins)
	}
}

func TestStatusForRootError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "missing root", err: domain.ErrRootNotFound, expected: http.StatusNotFound},
		{name: "not a directory", err: domain.ErrRootNotDirectory, expected: http.StatusNotFound},
		{name: "wrapped missing root", err: errors.Join(domain.ErrRootNotFound), expected: http.StatusNotFound},
		{name: "sensitive root", err: domain.ErrRootNotAllowed, expected: http.StatusBadRequest},
		{name: "outside allowed roots", err: domain.ErrRootOutsideScope, expected: http.StatusBadRequest},
		{name: "invalid root", err: domain.ErrInvalidRoot, expected: http.StatusBadRequest},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if status := statusForRootError(testCase.err); status != testCase.expected {
				t.Fatalf("expected %d, got %d", testCase.expected, status)
			}
		})
	}
}

func requireGitExecutable(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git executable is not available: %v", err)
	}
}

func initializeGitRepository(t *testing.T, repositoryPath string) {
	t.Helper()

	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("create repository directory: %v", err)
	}
	runGitTestCommand(t, repositoryPath, "init", "-q", "-b", "main", ".")
}

func commitTestRepository(t *testing.T, repositoryPath string, message string) {
	t.Helper()

	runGitTestCommand(t, repositoryPath, "add", "-A")
	runGitTestCommand(t, repositoryPath, "commit", "-q", "-m", message)
}

func runGitTestCommand(t *testing.T, repositoryPath string, args ...string) string {
	t.Helper()

	command := exec.Command("git", append([]string{"--no-optional-locks", "-C", repositoryPath}, args...)...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Lunar Test",
		"GIT_AUTHOR_EMAIL=lunar@test.local",
		"GIT_COMMITTER_NAME=Lunar Test",
		"GIT_COMMITTER_EMAIL=lunar@test.local",
		"GIT_CONFIG_NOSYSTEM=1",
	)

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, output)
	}
	return string(output)
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file %s: %v", path, err)
	}
}

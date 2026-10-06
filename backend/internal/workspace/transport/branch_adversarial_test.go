package transport

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func adversarialCommitAt(t *testing.T, repositoryPath string, fixtureName string, message string, timestamp string) {
	t.Helper()

	writeGitFixtureFile(t, filepath.Join(repositoryPath, fixtureName), fixtureName)
	runGitTool(t, repositoryPath, "add", "-A")

	commandArguments := append([]string{"--no-optional-locks", "-C", repositoryPath}, "commit", "-q", "-m", message)
	command := exec.Command("git", commandArguments...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Lunar Test",
		"GIT_AUTHOR_EMAIL=lunar@test.local",
		"GIT_COMMITTER_NAME=Lunar Test",
		"GIT_COMMITTER_EMAIL=lunar@test.local",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_DATE="+timestamp,
		"GIT_COMMITTER_DATE="+timestamp,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit failed: %v: %s", err, output)
	}
}

func adversarialOrderedRepository(t *testing.T, rootPath string) string {
	t.Helper()

	remotePath := filepath.Join(t.TempDir(), "ordered-remote.git")
	if err := os.MkdirAll(remotePath, 0o755); err != nil {
		t.Fatalf("cannot create the ordered remote: %v", err)
	}
	runGitTool(t, remotePath, "init", "-q", "--bare", "-b", "main", ".")

	sourcePath := filepath.Join(t.TempDir(), "ordered-source")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("cannot create the ordered source: %v", err)
	}
	runGitTool(t, sourcePath, "init", "-q", "-b", "main", ".")
	adversarialCommitAt(t, sourcePath, "README.md", "main commit", "2024-01-01T00:00:00+00:00")

	agedBranches := []struct {
		name      string
		timestamp string
	}{
		{name: "master", timestamp: "2024-02-01T00:00:00+00:00"},
		{name: "develop", timestamp: "2024-03-01T00:00:00+00:00"},
		{name: "pre/development", timestamp: "2024-04-01T00:00:00+00:00"},
		{name: "bugfix/CRMMS-77911-hotfix", timestamp: "2024-05-01T00:00:00+00:00"},
		{name: "feature/CRMMS-77911-in-flight", timestamp: "2024-06-01T00:00:00+00:00"},
	}
	for _, aged := range agedBranches {
		runGitTool(t, sourcePath, "checkout", "-q", "-b", aged.name, "main")
		fixtureName := strings.ReplaceAll(aged.name, "/", "-") + ".txt"
		adversarialCommitAt(t, sourcePath, fixtureName, aged.name+" commit", aged.timestamp)
	}

	runGitTool(t, sourcePath, "checkout", "-q", "main")
	runGitTool(t, sourcePath, "remote", "add", "origin", remotePath)
	runGitTool(t, sourcePath, "push", "-q", "origin", "--all")

	repositoryPath := filepath.Join(rootPath, "ordered-repo")
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the ordered clone: %v", err)
	}
	runGitTool(t, repositoryPath, "clone", "-q", remotePath, ".")
	runGitTool(t, repositoryPath, "remote", "set-head", "origin", "main")

	for _, aged := range agedBranches {
		runGitTool(t, repositoryPath, "branch", aged.name, "origin/"+aged.name)
	}

	return repositoryPath
}

func adversarialRemoteOnlyDefaultRepository(t *testing.T, rootPath string) string {
	t.Helper()

	remotePath := filepath.Join(t.TempDir(), "remote-only-default.git")
	if err := os.MkdirAll(remotePath, 0o755); err != nil {
		t.Fatalf("cannot create the remote: %v", err)
	}
	runGitTool(t, remotePath, "init", "-q", "--bare", "-b", "main", ".")

	sourcePath := filepath.Join(t.TempDir(), "remote-only-source")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatalf("cannot create the source: %v", err)
	}
	runGitTool(t, sourcePath, "init", "-q", "-b", "main", ".")
	adversarialCommitAt(t, sourcePath, "README.md", "main commit", "2024-01-01T00:00:00+00:00")
	runGitTool(t, sourcePath, "checkout", "-q", "-b", "trunk", "main")
	adversarialCommitAt(t, sourcePath, "trunk.txt", "trunk commit", "2024-02-01T00:00:00+00:00")
	runGitTool(t, sourcePath, "checkout", "-q", "main")
	runGitTool(t, sourcePath, "remote", "add", "origin", remotePath)
	runGitTool(t, sourcePath, "push", "-q", "origin", "--all")

	repositoryPath := filepath.Join(rootPath, "remote-only-default-repo")
	if err := os.MkdirAll(repositoryPath, 0o755); err != nil {
		t.Fatalf("cannot create the clone: %v", err)
	}
	runGitTool(t, repositoryPath, "clone", "-q", remotePath, ".")
	runGitTool(t, repositoryPath, "remote", "set-head", "origin", "trunk")

	return repositoryPath
}

func adversarialDecodeOptions(t *testing.T, handler http.Handler, repoName string) baseBranchOptionsResponse {
	t.Helper()

	path := "/api/workspace/repos/" + repoName + "/base-branch-options"
	recorder := performRequest(handler, authorizedRequest(t, http.MethodGet, path, "", handlerTestUserA))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d: %s", path, recorder.Code, recorder.Body.String())
	}

	var response baseBranchOptionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("cannot decode the base branch options: %v", err)
	}

	return response
}

func TestAdversarialCreateBranchesRefusesAnEmptyOrNullTargetsArray(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	emptyTargets := postGitRequest(t, handler, branchCreatePath, createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName, Targets: []createBranchTargetRequest{}})
	if emptyTargets.Code != http.StatusBadRequest {
		t.Fatalf("an empty targets array must be a request level 400, got %d: %s", emptyTargets.Code, emptyTargets.Body.String())
	}

	nullTargets := postGitRequest(t, handler, branchCreatePath, createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName, Targets: nil})
	if nullTargets.Code != http.StatusBadRequest {
		t.Fatalf("a null targets array must be a request level 400, got %d: %s", nullTargets.Code, nullTargets.Body.String())
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("a refused request must not create a branch")
		}
	}
}

func TestAdversarialCreateBranchesAccountsForADuplicatedRepositoryTarget(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if countOutcomeTargets(response) != 2 {
		t.Fatalf("a duplicated repository must still be accounted for twice, got %+v", response)
	}
	if len(response.Created) != 1 || len(response.Skipped) != 1 || len(response.Failed) != 0 {
		t.Fatalf("expected the second attempt on the same repository to be skipped, got %+v", response)
	}
}

func TestAdversarialCreateBranchesSkipsATargetThatIsBothExistingAndDirty(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	runGitTool(t, repositoryPath, "branch", rememberedBranchName, "main")
	writeGitFixtureFile(t, filepath.Join(repositoryPath, "README.md"), "modified")

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "scratch", BaseBranch: "main"},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if countOutcomeTargets(response) != 1 {
		t.Fatalf("the target must appear exactly once, got %+v", response)
	}
	if len(response.Skipped) != 1 {
		t.Fatalf("an existing branch must be skipped even on a dirty tree, got %+v", response)
	}
	if len(response.Failed) != 0 || len(response.Created) != 0 {
		t.Fatalf("the target must not be both skipped and failed, got %+v", response)
	}
}

func TestAdversarialCreateBranchesRefusesAnEmptyRepositoryNameBeforeAnythingIsCreated(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchesRequest{
		Key:    "CRMMS-77911",
		Branch: rememberedBranchName,
		Targets: []createBranchTargetRequest{
			{Repo: "scratch", BaseBranch: "main"},
			{Repo: "", BaseBranch: "main"},
		},
	})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("an empty repository name must refuse the whole request with 400, got %d: %s", recorder.Code, recorder.Body.String())
	}

	for _, branchName := range scratchBranchNames(t, repositoryPath) {
		if branchName == rememberedBranchName {
			t.Fatal("a request refused at validation must not create the earlier target either")
		}
	}
}

func TestAdversarialBaseBranchOptionsExcludesNothingByName(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)
	adversarialOrderedRepository(t, rootPath)

	response := adversarialDecodeOptions(t, handler, "ordered-repo")

	foundNames := make(map[string]bool, len(response.Options))
	for _, option := range response.Options {
		foundNames[option.Name] = true
		if option.LastCommitRelative == nil {
			t.Fatalf("every option must carry its last commit, %q did not", option.Name)
		}
	}

	for _, expected := range []string{"main", "master", "develop", "pre/development", "bugfix/CRMMS-77911-hotfix", "feature/CRMMS-77911-in-flight"} {
		if !foundNames[expected] {
			t.Fatalf("rule 54 forbids filtering by name, %q is missing from %+v", expected, response.Options)
		}
	}
}

func TestAdversarialBaseBranchOptionsOrdersEveryGroupAsClaimed(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)
	adversarialOrderedRepository(t, rootPath)

	response := adversarialDecodeOptions(t, handler, "ordered-repo")
	if response.Remembered != nil {
		t.Fatalf("expected nothing remembered yet, got %q", *response.Remembered)
	}
	if response.BitbucketDefault == nil || *response.BitbucketDefault != "main" {
		t.Fatalf("expected the remote default main, got %+v", response.BitbucketDefault)
	}

	expectedOrder := []struct {
		name   string
		source string
	}{
		{name: "main", source: "bitbucket_default"},
		{name: "develop", source: "trusted"},
		{name: "master", source: "trusted"},
		{name: "feature/CRMMS-77911-in-flight", source: "other"},
		{name: "bugfix/CRMMS-77911-hotfix", source: "other"},
		{name: "pre/development", source: "other"},
	}

	if len(response.Options) != len(expectedOrder) {
		t.Fatalf("expected %d options, got %+v", len(expectedOrder), response.Options)
	}
	for index, expected := range expectedOrder {
		if response.Options[index].Name != expected.name {
			t.Fatalf("expected option %d to be %q, got %+v", index, expected.name, response.Options)
		}
		if response.Options[index].Source != expected.source {
			t.Fatalf("expected %q to be %q, got %q", expected.name, expected.source, response.Options[index].Source)
		}
	}
}

func TestAdversarialBaseBranchOptionsPutsTheRememberedChoiceFirst(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)
	adversarialOrderedRepository(t, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "ordered-repo", BaseBranch: "develop", RememberBase: true},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := adversarialDecodeOptions(t, handler, "ordered-repo")
	if response.Remembered == nil || *response.Remembered != "develop" {
		t.Fatalf("expected develop to be remembered, got %+v", response.Remembered)
	}
	if len(response.Options) == 0 || response.Options[0].Name != "develop" || response.Options[0].Source != "remembered" {
		t.Fatalf("expected the remembered choice first, got %+v", response.Options)
	}
}

func TestAdversarialBaseBranchOptionsReportsADefaultThatHasNoLocalBranch(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)
	adversarialRemoteOnlyDefaultRepository(t, rootPath)

	response := adversarialDecodeOptions(t, handler, "remote-only-default-repo")

	if response.BitbucketDefault == nil || *response.BitbucketDefault != "trunk" {
		t.Fatalf("expected the remote default trunk, got %+v", response.BitbucketDefault)
	}

	optionNames := make([]string, 0, len(response.Options))
	for _, option := range response.Options {
		optionNames = append(optionNames, option.Name)
		if option.LastCommitRelative == nil {
			t.Fatalf("every emitted option must carry its last commit, %q did not", option.Name)
		}
	}
	t.Logf("remote default trunk with no local branch produced options %v and other_branch_count %d", optionNames, response.OtherBranchCount)
}
func TestAdversarialRememberedBaseDoesNotSurviveAServiceRestart(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)
	adversarialOrderedRepository(t, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchTargets(
		createBranchTargetRequest{Repo: "ordered-repo", BaseBranch: "develop", RememberBase: true},
	))
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	beforeRestart := adversarialDecodeOptions(t, handler, "ordered-repo")
	if beforeRestart.Remembered == nil || *beforeRestart.Remembered != "develop" {
		t.Fatalf("expected develop to be remembered before the restart, got %+v", beforeRestart.Remembered)
	}

	restartedHandler, _, _, _ := setupBranchCreateHandler(t, nil, []string{rootPath})
	saveServerWorkspace(t, restartedHandler, rootPath)

	afterRestart := adversarialDecodeOptions(t, restartedHandler, "ordered-repo")
	if afterRestart.Remembered != nil {
		t.Fatalf("process memory cannot survive a restart, expected null, got %q", *afterRestart.Remembered)
	}
	t.Logf("characterisation: after a restart remembered is null and the first option is %q with source %q", afterRestart.Options[0].Name, afterRestart.Options[0].Source)
}

func TestAdversarialBaseBranchOptionsForANeverClonedRepository(t *testing.T) {
	handler, rootPath, _, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	response := adversarialDecodeOptions(t, handler, missingRepositoryName)

	if response.IsCloned {
		t.Fatal("a repository that was never cloned must report is_cloned false")
	}
	if response.BitbucketDefault != nil {
		t.Fatalf("the default comes from the local origin head, so a never cloned repository has none, got %q", *response.BitbucketDefault)
	}
	if response.Remembered != nil {
		t.Fatalf("expected no remembered base, got %q", *response.Remembered)
	}
	if len(response.Options) != 0 {
		t.Fatalf("expected no options without a clone, got %+v", response.Options)
	}
}

func TestAdversarialCreateBranchesErrorBodiesAreReadableSentences(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	testCases := []struct {
		name    string
		request createBranchesRequest
	}{
		{name: "an empty targets array", request: createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName, Targets: []createBranchTargetRequest{}}},
		{name: "a null targets array", request: createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName}},
		{name: "an empty key", request: createBranchesRequest{Branch: rememberedBranchName, Targets: []createBranchTargetRequest{{Repo: "scratch", BaseBranch: "main"}}}},
		{name: "a target without a base branch", request: createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName, Targets: []createBranchTargetRequest{{Repo: "scratch"}}}},
		{name: "an invalid branch name", request: createBranchesRequest{Key: "CRMMS-77911", Branch: "main; rm -rf /", Targets: []createBranchTargetRequest{{Repo: "scratch", BaseBranch: "main"}}}},
		{name: "an invalid repository name", request: createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName, Targets: []createBranchTargetRequest{{Repo: "../outside", BaseBranch: "main"}}}},
		{name: "an invalid base branch name", request: createBranchesRequest{Key: "CRMMS-77911", Branch: rememberedBranchName, Targets: []createBranchTargetRequest{{Repo: "scratch", BaseBranch: "main; rm -rf /"}}}},
	}

	for _, testCase := range testCases {
		recorder := postGitRequest(t, handler, branchCreatePath, testCase.request)
		body := recorder.Body.String()
		t.Logf("%s returned %d with body %s", testCase.name, recorder.Code, body)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s must be a request level 400, got %d: %s", testCase.name, recorder.Code, body)
		}

		var decoded map[string]string
		if err := json.Unmarshal([]byte(body), &decoded); err != nil {
			t.Fatalf("%s must return a JSON error object, got %s", testCase.name, body)
		}
		if strings.TrimSpace(decoded["error"]) == "" {
			t.Fatalf("%s must name a readable reason, got %s", testCase.name, body)
		}
		if strings.Contains(decoded["error"], "/var/") || strings.Contains(decoded["error"], "goroutine") {
			t.Fatalf("%s must not leak a path or a stack trace, got %q", testCase.name, decoded["error"])
		}
		for _, branchName := range scratchBranchNames(t, repositoryPath) {
			if branchName == rememberedBranchName {
				t.Fatalf("%s must not create a branch", testCase.name)
			}
		}
	}
}
func TestAdversarialCreateBranchesSkipsWhenTheBranchNameEqualsTheBaseBranch(t *testing.T) {
	handler, rootPath, repositoryPath, _ := setupBranchCreateHandler(t, nil, nil)
	saveServerWorkspace(t, handler, rootPath)

	recorder := postGitRequest(t, handler, branchCreatePath, createBranchesRequest{
		Key:    "CRMMS-77911",
		Branch: "main",
		Targets: []createBranchTargetRequest{
			{Repo: "scratch", BaseBranch: "main"},
		},
	})
	if recorder.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d: %s", recorder.Code, recorder.Body.String())
	}

	response := decodeCreateBranchesResponse(t, recorder)
	if countOutcomeTargets(response) != 1 {
		t.Fatalf("the target must appear exactly once, got %+v", response)
	}
	if len(response.Skipped) != 1 || len(response.Failed) != 0 || len(response.Created) != 0 {
		t.Fatalf("creating main from main is a duplicate and must be skipped, got %+v", response)
	}

	branches := scratchBranchNames(t, repositoryPath)
	mainCount := 0
	for _, branchName := range branches {
		if branchName == "main" {
			mainCount++
		}
	}
	if mainCount != 1 {
		t.Fatalf("main must still exist exactly once, got %v", branches)
	}
}

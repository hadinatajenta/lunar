package application

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

const (
	testUserID      = "user-a"
	testUserBID     = "user-b"
	testEncryptKey  = "0123456789abcdef0123456789abcdef"
	testHelperToken = "helper-secret-token-value"
)

type fakeWorkspaceRepository struct {
	workspaces map[string]*domain.Workspace
	getErr     error
	upsertErr  error
}

func newFakeWorkspaceRepository() *fakeWorkspaceRepository {
	return &fakeWorkspaceRepository{workspaces: make(map[string]*domain.Workspace)}
}

func (f *fakeWorkspaceRepository) GetWorkspace(ctx context.Context, userID string) (*domain.Workspace, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	workspace, exists := f.workspaces[userID]
	if !exists {
		return nil, sharedErrors.ErrNotFound
	}

	storedCopy := *workspace
	return &storedCopy, nil
}

func (f *fakeWorkspaceRepository) UpsertWorkspace(ctx context.Context, workspace *domain.Workspace) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}

	storedCopy := *workspace
	f.workspaces[workspace.UserID] = &storedCopy
	return nil
}

type fakeSelectionRepository struct {
	selections   map[string][]string
	replaceCalls int
	listErr      error
	replaceErr   error
}

func newFakeSelectionRepository() *fakeSelectionRepository {
	return &fakeSelectionRepository{selections: make(map[string][]string)}
}

func (f *fakeSelectionRepository) ListSelected(ctx context.Context, userID string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	stored := f.selections[userID]
	listCopy := make([]string, len(stored))
	copy(listCopy, stored)
	return listCopy, nil
}

func (f *fakeSelectionRepository) ReplaceSelections(ctx context.Context, userID string, repoNames []string) error {
	if f.replaceErr != nil {
		return f.replaceErr
	}

	f.replaceCalls++
	replacement := make([]string, len(repoNames))
	copy(replacement, repoNames)
	f.selections[userID] = replacement
	return nil
}

type fakeGitReader struct {
	summaries     []domain.RepositorySummary
	details       []domain.RepositoryDetail
	listErr       error
	readErr       error
	receivedRoot  string
	receivedNames []string
}

func (f *fakeGitReader) ListRepositories(ctx context.Context, rootPath string) ([]domain.RepositorySummary, error) {
	f.receivedRoot = rootPath
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.summaries, nil
}

func (f *fakeGitReader) ReadRepositories(ctx context.Context, rootPath string, names []string) ([]domain.RepositoryDetail, error) {
	f.receivedRoot = rootPath
	f.receivedNames = names
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.details, nil
}

func (f *fakeGitReader) ValidateRoot(ctx context.Context, rootPath string) error {
	return nil
}

func newTestWorkspaceService(workspaceRepo domain.WorkspaceRepository, selectionRepo domain.SelectionRepository, gitReader domain.GitReader, allowedRoots []string) *WorkspaceService {
	return NewWorkspaceService(workspaceRepo, selectionRepo, gitReader, allowedRoots, testEncryptKey)
}

func TestWorkspaceService_GetWorkspaceDefaultsToHelperWhenUnset(t *testing.T) {
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), newFakeSelectionRepository(), nil, nil)

	config, err := service.GetWorkspace(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("cannot load default workspace: %v", err)
	}

	if config.Source != domain.SourceHelper {
		t.Errorf("expected default source %q, got %q", domain.SourceHelper, config.Source)
	}
	if config.RootPath != "" || config.HelperURL != "" {
		t.Errorf("expected empty workspace fields, got %+v", config)
	}
	if config.HasHelperToken {
		t.Error("expected has_helper_token to be false for an unconfigured workspace")
	}
	if config.SelectedRepoNames == nil {
		t.Fatal("expected an empty selection slice instead of nil")
	}
	if len(config.SelectedRepoNames) != 0 {
		t.Errorf("expected no stored selections, got %v", config.SelectedRepoNames)
	}
}

func TestWorkspaceService_GetWorkspaceIncludesStoredSelections(t *testing.T) {
	selectionRepo := newFakeSelectionRepository()
	selectionRepo.selections[testUserID] = []string{"everest", "aurora"}
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), selectionRepo, nil, nil)

	config, err := service.GetWorkspace(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("cannot load workspace: %v", err)
	}

	if len(config.SelectedRepoNames) != 2 || config.SelectedRepoNames[0] != "everest" || config.SelectedRepoNames[1] != "aurora" {
		t.Fatalf("expected the stored selections, got %v", config.SelectedRepoNames)
	}
}

func TestWorkspaceService_SaveWorkspaceRejectsUnknownSource(t *testing.T) {
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), newFakeSelectionRepository(), nil, nil)

	_, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{Source: "cluster"})

	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "helper") || !strings.Contains(err.Error(), "server") {
		t.Errorf("expected the error to name both accepted sources, got %q", err.Error())
	}
}

func TestWorkspaceService_SaveWorkspaceServerModeResolvesRootPath(t *testing.T) {
	rootPath := t.TempDir()
	workspaceRepo := newFakeWorkspaceRepository()
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), nil, nil)

	config, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:   string(domain.SourceServer),
		RootPath: rootPath,
	})
	if err != nil {
		t.Fatalf("cannot save server workspace: %v", err)
	}

	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		t.Fatalf("cannot resolve expected root: %v", err)
	}
	if config.RootPath != resolvedRoot {
		t.Errorf("expected resolved root %q, got %q", resolvedRoot, config.RootPath)
	}
}

func TestWorkspaceService_SaveWorkspaceServerModeRejectsMissingRootWithoutLeakingPath(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing-repository-folder")
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), newFakeSelectionRepository(), nil, nil)

	_, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:   string(domain.SourceServer),
		RootPath: missingRoot,
	})

	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "workspace root does not exist") {
		t.Errorf("expected a clear root message, got %q", err.Error())
	}
	if strings.Contains(err.Error(), missingRoot) {
		t.Errorf("expected the absolute path to stay out of the response, got %q", err.Error())
	}
}

func TestWorkspaceService_SaveWorkspaceServerModeRejectsRootOutsideAllowlist(t *testing.T) {
	allowedRoot := t.TempDir()
	outsideRoot := t.TempDir()
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), newFakeSelectionRepository(), nil, []string{allowedRoot})

	_, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:   string(domain.SourceServer),
		RootPath: outsideRoot,
	})

	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "outside the allowed roots") {
		t.Errorf("expected an allowlist message, got %q", err.Error())
	}
}

func TestWorkspaceService_SaveWorkspaceHelperModeKeepsRootPathUntouched(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), nil, nil)

	config, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:    string(domain.SourceHelper),
		RootPath:  "  /Users/erendt/BRI  ",
		HelperURL: "http://127.0.0.1:5199/",
	})
	if err != nil {
		t.Fatalf("cannot save helper workspace: %v", err)
	}

	if config.RootPath != "/Users/erendt/BRI" {
		t.Errorf("expected the helper root to stay untouched, got %q", config.RootPath)
	}
	if config.HelperURL != "http://127.0.0.1:5199" {
		t.Errorf("expected a normalized helper url, got %q", config.HelperURL)
	}
}

func TestWorkspaceService_SaveWorkspaceRejectsInvalidHelperURL(t *testing.T) {
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), newFakeSelectionRepository(), nil, nil)

	for _, helperURL := range []string{"127.0.0.1:5199", "ftp://127.0.0.1:5199", "http://"} {
		_, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
			Source:    string(domain.SourceHelper),
			HelperURL: helperURL,
		})
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected %q to be rejected with ErrBadRequest, got %v", helperURL, err)
		}
	}
}

func TestWorkspaceService_SaveWorkspaceEncryptsHelperToken(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), nil, nil)

	config, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:      string(domain.SourceHelper),
		HelperToken: stringPointer(testHelperToken),
	})
	if err != nil {
		t.Fatalf("cannot save helper token: %v", err)
	}

	storedToken := workspaceRepo.workspaces[testUserID].HelperTokenEnc
	if storedToken == "" {
		t.Fatal("expected a stored ciphertext")
	}
	if storedToken == testHelperToken {
		t.Fatal("expected the helper token to be encrypted at rest, not stored as plaintext")
	}
	if strings.Contains(storedToken, testHelperToken) {
		t.Fatal("expected the ciphertext to not contain the plaintext token")
	}

	decryptedToken, err := crypto.DecryptSecret(storedToken, testEncryptKey)
	if err != nil {
		t.Fatalf("cannot decrypt the stored token: %v", err)
	}
	if decryptedToken != testHelperToken {
		t.Errorf("expected the ciphertext to round-trip, got %q", decryptedToken)
	}
	if !config.HasHelperToken {
		t.Error("expected has_helper_token to be true after saving a token")
	}
}

func TestWorkspaceService_SaveWorkspacePreservesTokenWhenOmittedAndClearsWhenEmpty(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:         testUserID,
		Source:         domain.SourceHelper,
		RootPath:       "/Users/erendt/BRI",
		HelperTokenEnc: "existing-ciphertext",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), nil, nil)

	preservedConfig, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:   string(domain.SourceHelper),
		RootPath: "/Users/erendt/BRI",
	})
	if err != nil {
		t.Fatalf("cannot save workspace without a token: %v", err)
	}
	if workspaceRepo.workspaces[testUserID].HelperTokenEnc != "existing-ciphertext" {
		t.Fatal("expected an omitted helper_token to preserve the stored ciphertext")
	}
	if !preservedConfig.HasHelperToken {
		t.Error("expected has_helper_token to stay true when the token is omitted")
	}
	if !workspaceRepo.workspaces[testUserID].CreatedAt.Truncate(time.Microsecond).Equal(createdAt) {
		t.Error("expected created_at to be preserved on update")
	}

	clearedConfig, err := service.SaveWorkspace(context.Background(), testUserID, SaveWorkspaceInput{
		Source:      string(domain.SourceHelper),
		RootPath:    "/Users/erendt/BRI",
		HelperToken: stringPointer(""),
	})
	if err != nil {
		t.Fatalf("cannot clear the helper token: %v", err)
	}
	if workspaceRepo.workspaces[testUserID].HelperTokenEnc != "" {
		t.Fatal("expected an explicit empty helper_token to clear the stored ciphertext")
	}
	if clearedConfig.HasHelperToken {
		t.Error("expected has_helper_token to be false after clearing")
	}
}

func TestWorkspaceService_ReplaceSelectionsRejectsInvalidNameAndKeepsExistingSelections(t *testing.T) {
	selectionRepo := newFakeSelectionRepository()
	selectionRepo.selections[testUserID] = []string{"everest"}
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), selectionRepo, nil, nil)

	for _, invalidName := range []string{"../etc", "sub/dir", "", ".", "..", `back\slash`} {
		_, err := service.ReplaceSelections(context.Background(), testUserID, []string{"everest", invalidName})

		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Fatalf("expected %q to reject the batch with ErrBadRequest, got %v", invalidName, err)
		}
		if selectionRepo.replaceCalls != 0 {
			t.Fatalf("expected the invalid batch %q to never reach the repository", invalidName)
		}
		if len(selectionRepo.selections[testUserID]) != 1 || selectionRepo.selections[testUserID][0] != "everest" {
			t.Fatalf("expected the stored selections to stay untouched after batch %q", invalidName)
		}
	}
}

func TestWorkspaceService_ReplaceSelectionsStoresValidatedUniqueNames(t *testing.T) {
	selectionRepo := newFakeSelectionRepository()
	service := newTestWorkspaceService(newFakeWorkspaceRepository(), selectionRepo, nil, nil)

	selectedCount, err := service.ReplaceSelections(context.Background(), testUserID, []string{"everest", "aurora", "everest", " way4 "})
	if err != nil {
		t.Fatalf("cannot replace selections: %v", err)
	}

	if selectedCount != 3 {
		t.Errorf("expected 3 stored selections, got %d", selectedCount)
	}
	storedSelections := selectionRepo.selections[testUserID]
	if len(storedSelections) != 3 || storedSelections[0] != "everest" || storedSelections[1] != "aurora" || storedSelections[2] != "way4" {
		t.Errorf("unexpected stored selections: %v", storedSelections)
	}
}

func TestWorkspaceService_ListRepositoriesRejectsHelperSourceWithClientSideMessage(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceHelper,
		RootPath: "/Users/erendt/BRI",
	}
	gitReader := &fakeGitReader{}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), gitReader, nil)

	repositoryList, err := service.ListRepositories(context.Background(), testUserID)

	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest for helper source, got %v", err)
	}
	if !strings.Contains(err.Error(), "client-side") || !strings.Contains(err.Error(), "local helper") {
		t.Errorf("expected a client-side explanation, got %q", err.Error())
	}
	if repositoryList != nil {
		t.Fatalf("expected no fabricated repository list, got %+v", repositoryList)
	}
	if gitReader.receivedRoot != "" {
		t.Errorf("expected the git reader to never be called, got root %q", gitReader.receivedRoot)
	}
}

func TestWorkspaceService_ListRepositoriesReportsUnavailableReaderInServerMode(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceServer,
		RootPath: "/Users/erendt/BRI",
	}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), nil, nil)

	_, err := service.ListRepositories(context.Background(), testUserID)

	if !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected ErrServerReaderUnavailable, got %v", err)
	}
}

func TestWorkspaceService_ListRepositoriesReturnsSummariesInServerMode(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceServer,
		RootPath: "/Users/erendt/BRI",
	}
	gitReader := &fakeGitReader{
		summaries: []domain.RepositorySummary{
			{Name: "everest", IsGit: true, Branch: "development", DirtyCount: 2},
			{Name: "way4", IsGit: false, Error: "not a git repository"},
		},
	}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), gitReader, nil)

	repositoryList, err := service.ListRepositories(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("cannot list repositories: %v", err)
	}

	if repositoryList.RootPath != "/Users/erendt/BRI" {
		t.Errorf("expected the workspace root, got %q", repositoryList.RootPath)
	}
	if len(repositoryList.Repositories) != 2 {
		t.Fatalf("expected 2 repositories, got %+v", repositoryList.Repositories)
	}
	if repositoryList.Repositories[0].Name != "everest" || repositoryList.Repositories[0].Branch != "development" {
		t.Errorf("unexpected first repository: %+v", repositoryList.Repositories[0])
	}
	if repositoryList.Repositories[1].IsGit {
		t.Error("expected the plain folder to stay is_git false")
	}
}

func TestWorkspaceService_ListRepositoriesReturnsEmptySliceWhenReaderFindsNothing(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceServer,
		RootPath: "/Users/erendt/BRI",
	}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), &fakeGitReader{}, nil)

	repositoryList, err := service.ListRepositories(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("cannot list repositories: %v", err)
	}
	if repositoryList.Repositories == nil {
		t.Fatal("expected an empty repository slice instead of nil")
	}
	if len(repositoryList.Repositories) != 0 {
		t.Fatalf("expected no repositories, got %+v", repositoryList.Repositories)
	}
}

func TestWorkspaceService_ListServicesReadsOnlySelectedRepositories(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceServer,
		RootPath: "/Users/erendt/BRI",
	}
	selectionRepo := newFakeSelectionRepository()
	selectionRepo.selections[testUserID] = []string{"aurora", "everest"}
	gitReader := &fakeGitReader{
		details: []domain.RepositoryDetail{
			{Name: "aurora", Branch: "master", Commit: domain.CommitInfo{Hash: "88b4ca3", Author: "Hadinata Jenta", RelativeTime: "37 minutes ago", Subject: "fix: guard nil session"}},
		},
	}
	service := newTestWorkspaceService(workspaceRepo, selectionRepo, gitReader, nil)

	serviceList, err := service.ListServices(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("cannot list services: %v", err)
	}

	if len(gitReader.receivedNames) != 2 || gitReader.receivedNames[0] != "aurora" || gitReader.receivedNames[1] != "everest" {
		t.Fatalf("expected the selected repositories to be read, got %v", gitReader.receivedNames)
	}
	if gitReader.receivedRoot != "/Users/erendt/BRI" {
		t.Errorf("expected the workspace root, got %q", gitReader.receivedRoot)
	}
	if len(serviceList.Services) != 1 || serviceList.Services[0].Commit.Hash != "88b4ca3" {
		t.Fatalf("unexpected services: %+v", serviceList.Services)
	}
}

func TestWorkspaceService_ListServicesSkipsReaderWhenNothingIsSelected(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceServer,
		RootPath: "/Users/erendt/BRI",
	}
	gitReader := &fakeGitReader{}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), gitReader, nil)

	serviceList, err := service.ListServices(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("cannot list services: %v", err)
	}

	if gitReader.receivedNames != nil {
		t.Errorf("expected the git reader to never be called, got %v", gitReader.receivedNames)
	}
	if serviceList.Services == nil || len(serviceList.Services) != 0 {
		t.Fatalf("expected an empty service slice, got %+v", serviceList.Services)
	}
}

func TestWorkspaceService_ListServicesSkipsSelectionsMissingFromReader(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceServer,
		RootPath: "/Users/erendt/BRI",
	}
	selectionRepo := newFakeSelectionRepository()
	selectionRepo.selections[testUserID] = []string{"everest", "missing-on-disk", "way4"}
	gitReader := &fakeGitReader{
		details: []domain.RepositoryDetail{{Name: "everest", Branch: "development"}},
	}
	service := newTestWorkspaceService(workspaceRepo, selectionRepo, gitReader, nil)

	serviceList, err := service.ListServices(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("expected a missing repository to not fail the response, got %v", err)
	}

	if len(serviceList.Services) != 1 || serviceList.Services[0].Name != "everest" {
		t.Fatalf("expected only the repository found on disk, got %+v", serviceList.Services)
	}
}

func TestWorkspaceService_ListServicesRejectsHelperSource(t *testing.T) {
	workspaceRepo := newFakeWorkspaceRepository()
	workspaceRepo.workspaces[testUserID] = &domain.Workspace{
		UserID:   testUserID,
		Source:   domain.SourceHelper,
		RootPath: "/Users/erendt/BRI",
	}
	service := newTestWorkspaceService(workspaceRepo, newFakeSelectionRepository(), &fakeGitReader{}, nil)

	_, err := service.ListServices(context.Background(), testUserID)

	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest for helper source, got %v", err)
	}
	if !strings.Contains(err.Error(), "client-side") {
		t.Errorf("expected a client-side explanation, got %q", err.Error())
	}
}

func stringPointer(value string) *string {
	return &value
}

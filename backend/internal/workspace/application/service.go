package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"lunar/backend/internal/shared/crypto"
	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

type WorkspaceService struct {
	workspaceRepo domain.WorkspaceRepository
	selectionRepo domain.SelectionRepository
	gitReader     domain.GitReader
	allowedRoots  []string
	encryptionKey string
}

func NewWorkspaceService(
	workspaceRepo domain.WorkspaceRepository,
	selectionRepo domain.SelectionRepository,
	gitReader domain.GitReader,
	allowedRoots []string,
	encryptionKey string,
) *WorkspaceService {
	return &WorkspaceService{
		workspaceRepo: workspaceRepo,
		selectionRepo: selectionRepo,
		gitReader:     gitReader,
		allowedRoots:  allowedRoots,
		encryptionKey: encryptionKey,
	}
}

func (s *WorkspaceService) GetWorkspace(ctx context.Context, userID string) (*WorkspaceConfig, error) {
	selectedRepoNames, err := s.selectionRepo.ListSelected(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load selections: %w", err)
	}

	workspace, err := s.workspaceRepo.GetWorkspace(ctx, userID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			return &WorkspaceConfig{Source: defaultWorkspaceSource, SelectedRepoNames: selectedRepoNames}, nil
		}
		return nil, fmt.Errorf("load workspace: %w", err)
	}

	return &WorkspaceConfig{
		Source:            workspace.Source,
		RootPath:          workspace.RootPath,
		HelperURL:         workspace.HelperURL,
		HasHelperToken:    strings.TrimSpace(workspace.HelperTokenEnc) != "",
		SelectedRepoNames: selectedRepoNames,
	}, nil
}

func (s *WorkspaceService) SaveWorkspace(ctx context.Context, userID string, input SaveWorkspaceInput) (*WorkspaceConfig, error) {
	source, err := parseWorkspaceSource(input.Source)
	if err != nil {
		return nil, err
	}

	rootPath := strings.TrimSpace(input.RootPath)
	if source == domain.SourceServer {
		resolvedRoot, resolveErr := domain.ResolveWorkspaceRoot(rootPath, s.allowedRoots)
		if resolveErr != nil {
			return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, describeRootError(resolveErr))
		}
		rootPath = resolvedRoot
	}

	helperURL := strings.TrimRight(strings.TrimSpace(input.HelperURL), "/")
	if err := validateHelperURL(helperURL); err != nil {
		return nil, err
	}

	helperTokenEnc, createdAt, err := s.resolveWorkspaceSecrets(ctx, userID, input.HelperToken)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	workspace := &domain.Workspace{
		UserID:         userID,
		Source:         source,
		RootPath:       rootPath,
		HelperURL:      helperURL,
		HelperTokenEnc: helperTokenEnc,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
	}
	if err := s.workspaceRepo.UpsertWorkspace(ctx, workspace); err != nil {
		return nil, fmt.Errorf("save workspace: %w", err)
	}

	return s.GetWorkspace(ctx, userID)
}

func (s *WorkspaceService) ListRepositories(ctx context.Context, userID string) (*RepositoryList, error) {
	workspace, err := s.requireServerWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}

	repositories, err := s.gitReader.ListRepositories(ctx, workspace.RootPath)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read repositories: %w", sharedErrors.ErrBadGateway, err)
	}
	if repositories == nil {
		repositories = []domain.RepositorySummary{}
	}

	return &RepositoryList{RootPath: workspace.RootPath, Repositories: repositories}, nil
}

func (s *WorkspaceService) ListServices(ctx context.Context, userID string) (*ServiceList, error) {
	workspace, err := s.requireServerWorkspace(ctx, userID)
	if err != nil {
		return nil, err
	}

	selectedRepoNames, err := s.selectionRepo.ListSelected(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load selections: %w", err)
	}

	services := []domain.RepositoryDetail{}
	if len(selectedRepoNames) > 0 {
		details, readErr := s.gitReader.ReadRepositories(ctx, workspace.RootPath, selectedRepoNames)
		if readErr != nil {
			return nil, fmt.Errorf("%w: cannot read repository details: %w", sharedErrors.ErrBadGateway, readErr)
		}
		services = filterSelectedDetails(details, selectedRepoNames)
	}

	return &ServiceList{RootPath: workspace.RootPath, Services: services}, nil
}

func (s *WorkspaceService) ReplaceSelections(ctx context.Context, userID string, repoNames []string) (int, error) {
	validatedRepoNames, err := validateSelectionBatch(repoNames)
	if err != nil {
		return 0, err
	}

	if err := s.selectionRepo.ReplaceSelections(ctx, userID, validatedRepoNames); err != nil {
		return 0, fmt.Errorf("replace selections: %w", err)
	}

	return len(validatedRepoNames), nil
}

func (s *WorkspaceService) resolveWorkspaceSecrets(ctx context.Context, userID string, rawHelperToken *string) (string, time.Time, error) {
	existing, err := s.workspaceRepo.GetWorkspace(ctx, userID)
	if err != nil && !errors.Is(err, sharedErrors.ErrNotFound) {
		return "", time.Time{}, fmt.Errorf("load workspace: %w", err)
	}

	createdAt := time.Now().UTC()
	helperTokenEnc := ""
	if existing != nil {
		createdAt = existing.CreatedAt
		helperTokenEnc = existing.HelperTokenEnc
	}

	if rawHelperToken == nil {
		return helperTokenEnc, createdAt, nil
	}

	trimmedToken := strings.TrimSpace(*rawHelperToken)
	if trimmedToken == "" {
		return "", createdAt, nil
	}

	encryptedToken, err := crypto.EncryptSecret(trimmedToken, s.encryptionKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encrypt helper token: %w", err)
	}

	return encryptedToken, createdAt, nil
}

func (s *WorkspaceService) requireServerWorkspace(ctx context.Context, userID string) (*domain.Workspace, error) {
	workspace, err := s.workspaceRepo.GetWorkspace(ctx, userID)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, clientSideSourceMessage)
		}
		return nil, fmt.Errorf("load workspace: %w", err)
	}

	if workspace.Source != domain.SourceServer {
		return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, clientSideSourceMessage)
	}
	if s.gitReader == nil {
		return nil, ErrServerReaderUnavailable
	}
	if strings.TrimSpace(workspace.RootPath) == "" {
		return nil, fmt.Errorf("%w: workspace root path is not configured", sharedErrors.ErrBadRequest)
	}

	return workspace, nil
}

func filterSelectedDetails(details []domain.RepositoryDetail, selectedRepoNames []string) []domain.RepositoryDetail {
	selectedNames := make(map[string]bool, len(selectedRepoNames))
	for _, repoName := range selectedRepoNames {
		selectedNames[repoName] = true
	}

	filteredDetails := make([]domain.RepositoryDetail, 0, len(details))
	for _, detail := range details {
		if selectedNames[detail.Name] {
			filteredDetails = append(filteredDetails, detail)
		}
	}

	return filteredDetails
}

package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	copilotapplication "lunar/backend/internal/copilot/application"
	"lunar/backend/internal/postman/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type RepositoryResolver interface {
	ResolveExportRepositoryPath(ctx context.Context, userID string, repoName string) (string, error)
}

type CredentialResolver interface {
	ResolveProviderCredentials(ctx context.Context, userID string, model string) (copilotapplication.ProviderCredentials, error)
}

type ExportRequest struct {
	RepositoryName string
	Model          string
}

type Exporter struct {
	repositories RepositoryResolver
	credentials  CredentialResolver
	pipeline     *Pipeline
}

func NewExporter(repositories RepositoryResolver, credentials CredentialResolver, pipeline *Pipeline) *Exporter {
	return &Exporter{repositories: repositories, credentials: credentials, pipeline: pipeline}
}

func (e *Exporter) Export(ctx context.Context, userID string, request ExportRequest, emit Emitter) (Result, error) {
	repositoryPath, err := e.repositories.ResolveExportRepositoryPath(ctx, userID, request.RepositoryName)
	if err != nil {
		return Result{}, fmt.Errorf("resolve export repository: %w", err)
	}

	credentials, err := e.resolveCredentials(ctx, userID, request.Model)
	if err != nil {
		return Result{}, err
	}

	return e.pipeline.Run(ctx, RunRequest{
		Service:     request.RepositoryName,
		RepoPath:    repositoryPath,
		Credentials: credentials,
	}, emit)
}

func (e *Exporter) resolveCredentials(ctx context.Context, userID string, model string) (domain.ProviderCredentials, error) {
	resolved, err := e.credentials.ResolveProviderCredentials(ctx, userID, model)
	if err != nil {
		if errors.Is(err, sharedErrors.ErrBadRequest) {
			return domain.ProviderCredentials{}, fmt.Errorf("%w: configure an API key for the selected model in Settings", domain.ErrProviderKeyMissing)
		}
		return domain.ProviderCredentials{}, fmt.Errorf("resolve provider credentials: %w", err)
	}

	if strings.TrimSpace(resolved.APIKey) == "" || strings.TrimSpace(resolved.BaseURL) == "" || strings.TrimSpace(resolved.Model) == "" {
		return domain.ProviderCredentials{}, fmt.Errorf("%w: the provider credentials are incomplete", domain.ErrProviderKeyMissing)
	}

	return domain.ProviderCredentials{
		Provider: resolved.Provider,
		APIKey:   resolved.APIKey,
		BaseURL:  resolved.BaseURL,
		Model:    resolved.Model,
	}, nil
}

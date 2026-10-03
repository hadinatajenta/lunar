package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	agentapplication "lunar/backend/internal/agent/application"
	agentinfrastructure "lunar/backend/internal/agent/infrastructure"
	agenttools "lunar/backend/internal/codeindex/application/tools"
	workspacedomain "lunar/backend/internal/workspace/domain"
	workspaceinfrastructure "lunar/backend/internal/workspace/infrastructure"
)

const providerKeyTimeout = 15 * time.Second

type providerCredentials struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

type helperRepositoryReader struct {
	workspaceRoot string
}

func newHelperRepositoryReader(workspaceRoot string) *helperRepositoryReader {
	return &helperRepositoryReader{workspaceRoot: strings.TrimSpace(workspaceRoot)}
}

func (r *helperRepositoryReader) WorkspaceRoot() string {
	return r.workspaceRoot
}

func (r *helperRepositoryReader) GetDiff(ctx context.Context, request agenttools.GitDiffRequest) (agenttools.GitDiffResult, error) {
	repositoryName, err := workspacedomain.ValidateRepoName(strings.TrimSpace(request.RepositoryName))
	if err != nil {
		return agenttools.GitDiffResult{}, fmt.Errorf("getDiff: %w", err)
	}
	if r.workspaceRoot == "" {
		return agenttools.GitDiffResult{}, fmt.Errorf("getDiff: the workspace root is not configured")
	}

	repositoryPath, err := workspacedomain.ResolveRepoPath(r.workspaceRoot, repositoryName)
	if err != nil {
		return agenttools.GitDiffResult{}, fmt.Errorf("getDiff: %w", err)
	}

	diff, isTruncated, err := workspaceinfrastructure.ReadRepositoryDiff(
		ctx,
		repositoryPath,
		request.BaseRef,
		request.IncludeUncommitted,
		request.FilePath,
	)
	if err != nil {
		return agenttools.GitDiffResult{}, fmt.Errorf("getDiff: %w", err)
	}

	return agenttools.GitDiffResult{
		RepositoryName: repositoryName,
		BaseRef:        strings.TrimSpace(request.BaseRef),
		Diff:           diff,
		IsTruncated:    isTruncated,
	}, nil
}

func fetchProviderCredentials(ctx context.Context, backendURL string, sessionToken string, model string) (providerCredentials, error) {
	endpoint := fmt.Sprintf("%s/api/copilot/provider-key?model=%s", backendURL, url.QueryEscape(strings.TrimSpace(model)))

	requestContext, cancel := context.WithTimeout(ctx, providerKeyTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return providerCredentials{}, fmt.Errorf("fetchProviderCredentials: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{Timeout: providerKeyTimeout}
	response, err := client.Do(request)
	if err != nil {
		return providerCredentials{}, fmt.Errorf("fetchProviderCredentials: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return providerCredentials{}, fmt.Errorf("fetchProviderCredentials: unexpected status %d", response.StatusCode)
	}

	var credentials providerCredentials
	if err := json.NewDecoder(response.Body).Decode(&credentials); err != nil {
		return providerCredentials{}, fmt.Errorf("fetchProviderCredentials: %w", err)
	}
	if strings.TrimSpace(credentials.APIKey) == "" || strings.TrimSpace(credentials.BaseURL) == "" || strings.TrimSpace(credentials.Model) == "" {
		return providerCredentials{}, fmt.Errorf("fetchProviderCredentials: the backend returned incomplete credentials")
	}
	return credentials, nil
}

func (s *helperServer) buildAgent(workspaceRoot string, credentials providerCredentials) *agentapplication.Agent {
	client := agentinfrastructure.NewChatClient(agentinfrastructure.ProviderConfig{
		Provider: credentials.Provider,
		APIKey:   credentials.APIKey,
		BaseURL:  credentials.BaseURL,
		Model:    credentials.Model,
	})

	reader := newHelperRepositoryReader(workspaceRoot)

	registry := agentapplication.NewMapToolRegistry()
	registry.RegisterAll(agenttools.NewTools(s.indexStore, reader)...)

	return agentapplication.NewAgent(client, registry)
}

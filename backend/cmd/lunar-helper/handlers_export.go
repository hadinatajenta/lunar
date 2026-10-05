package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	postmanapplication "lunar/backend/internal/postman/application"
	"lunar/backend/internal/postman/domain"
	postmaninfrastructure "lunar/backend/internal/postman/infrastructure"
	postmantransport "lunar/backend/internal/postman/transport"
	sharedHttp "lunar/backend/internal/shared/http"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	exportRequestTimeout = 5 * time.Minute
	exportWriteDeadline  = 10 * time.Minute

	messageExportInvalidBody            = "invalid export request"
	messageExportModelRequired          = "a model is required"
	messageExportCredentialsRequired    = "the export request must carry backend_url and session_token so the helper can resolve the provider key"
	messageExportProviderKeyMissing     = "no provider key is configured for the selected model"
	messageExportProviderTimeout        = "the provider did not respond in time"
	messageExportSessionRejected        = "the backend rejected the helper session; sign in again"
	messageExportCredentialsUnavailable = "cannot resolve the provider credentials from the backend"
	messageExportDeadlineFailed         = "the request deadline could not be extended for the export"
	messageExportStreamingUnsupported   = "streaming is not supported"
)

type exportRequest struct {
	Root         string `json:"root"`
	Name         string `json:"name"`
	Model        string `json:"model"`
	BackendURL   string `json:"backend_url"`
	SessionToken string `json:"session_token"`
}

func (s *helperServer) handleExport(w http.ResponseWriter, r *http.Request) {
	var request exportRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageExportInvalidBody)
		return
	}

	root := firstNonEmpty(rootQueryValue(r), strings.TrimSpace(request.Root))
	name := firstNonEmpty(nameQueryValue(r), strings.TrimSpace(request.Name))
	model := strings.TrimSpace(request.Model)
	backendURL := strings.TrimRight(strings.TrimSpace(request.BackendURL), "/")
	sessionToken := strings.TrimSpace(request.SessionToken)
	if model == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageExportModelRequired)
		return
	}
	if backendURL == "" || sessionToken == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageExportCredentialsRequired)
		return
	}

	repositoryPath, err := resolveExportRepositoryPath(root, name, s.allowedRoots)
	if err != nil {
		status, message := postmantransport.FailureResponse(err)
		sharedHttp.WriteError(w, status, message)
		return
	}

	credentials, err := fetchProviderCredentials(r.Context(), backendURL, sessionToken, model)
	if err != nil {
		s.logger.Warn("cannot fetch provider credentials for the export", "error", err)
		status, message := helperCredentialFailure(err)
		sharedHttp.WriteError(w, status, message)
		return
	}

	if err := sharedHttp.ExtendWriteDeadline(w, exportWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageExportDeadlineFailed)
		return
	}
	stream, err := postmantransport.NewStream(w)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageExportStreamingUnsupported)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), exportRequestTimeout)
	defer cancel()

	pipeline := postmanapplication.NewPipeline(
		postmaninfrastructure.NewSelector(),
		postmaninfrastructure.NewClassifier(),
		postmaninfrastructure.NewProviderClient(),
	)
	result, exportError := pipeline.Run(ctx, postmanapplication.RunRequest{
		Service:  name,
		RepoPath: repositoryPath,
		Credentials: domain.ProviderCredentials{
			Provider: credentials.Provider,
			APIKey:   credentials.APIKey,
			BaseURL:  credentials.BaseURL,
			Model:    credentials.Model,
		},
	}, stream.Stage)
	if exportError != nil {
		if !stream.IsStarted() {
			status, message := postmantransport.FailureResponse(exportError)
			sharedHttp.WriteError(w, status, message)
			return
		}
		if writeError := stream.Error(postmantransport.FailureMessage(exportError)); writeError != nil {
			s.logger.Warn("cannot report the export failure", "error", writeError)
		}
		return
	}

	if writeError := stream.Result(result); writeError != nil {
		s.logger.Warn("cannot write the export result", "error", writeError)
	}
}

func resolveExportRepositoryPath(root string, name string, allowedRoots []string) (string, error) {
	resolvedRoot, err := workspacedomain.ResolveWorkspaceRoot(root, allowedRoots)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}

	repositoryPath, err := workspacedomain.ResolveRepoPath(resolvedRoot, name)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	return repositoryPath, nil
}

func helperCredentialFailure(err error) (int, string) {
	var credentialError *providerCredentialError
	if errors.As(err, &credentialError) {
		switch credentialError.StatusCode {
		case http.StatusBadRequest, http.StatusConflict:
			return http.StatusConflict, messageExportProviderKeyMissing
		case http.StatusUnauthorized, http.StatusForbidden:
			return http.StatusUnauthorized, messageExportSessionRejected
		case http.StatusGatewayTimeout:
			return http.StatusGatewayTimeout, messageExportProviderTimeout
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, messageExportProviderTimeout
	}
	return http.StatusBadGateway, messageExportCredentialsUnavailable
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

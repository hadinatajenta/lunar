package application

import (
	"errors"

	"lunar/backend/internal/workspace/domain"
)

var ErrServerReaderUnavailable = errors.New("server-side repository reader is not configured")

const (
	defaultWorkspaceSource  domain.WorkspaceSource = domain.SourceHelper
	clientSideSourceMessage                        = "repository data is read client-side by the local helper; set source to server to read repositories on the backend"
)

type WorkspaceConfig struct {
	Source            domain.WorkspaceSource
	RootPath          string
	HelperURL         string
	HasHelperToken    bool
	SelectedRepoNames []string
}

type SaveWorkspaceInput struct {
	Source      string
	RootPath    string
	HelperURL   string
	HelperToken *string
}

type RepositoryList struct {
	RootPath     string
	Repositories []domain.RepositorySummary
}

type ServiceList struct {
	RootPath string
	Services []domain.RepositoryDetail
}

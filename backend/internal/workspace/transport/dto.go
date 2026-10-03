package transport

import (
	"lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
)

type saveWorkspaceRequest struct {
	Source      string  `json:"source"`
	RootPath    string  `json:"root_path"`
	HelperURL   string  `json:"helper_url"`
	HelperToken *string `json:"helper_token"`
}

type replaceSelectionsRequest struct {
	Repos []string `json:"repos"`
}

type workspaceResponse struct {
	Source         string   `json:"source"`
	RootPath       string   `json:"root_path"`
	HelperURL      string   `json:"helper_url"`
	HasHelperToken bool     `json:"has_helper_token"`
	Selected       []string `json:"selected"`
}

type repositorySummaryResponse struct {
	Name            string `json:"name"`
	IsGit           bool   `json:"is_git"`
	Branch          string `json:"branch"`
	DirtyCount      int    `json:"dirty_count"`
	UpdatedRelative string `json:"updated_relative"`
	Error           string `json:"error"`
}

type repositoriesResponse struct {
	RootPath     string                      `json:"root_path"`
	Repositories []repositorySummaryResponse `json:"repositories"`
}

type changedFileResponse struct {
	Status  string `json:"status"`
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
}

type commitInfoResponse struct {
	Hash         string `json:"hash"`
	Author       string `json:"author"`
	RelativeTime string `json:"relative_time"`
	Subject      string `json:"subject"`
}

type repositoryDetailResponse struct {
	Name            string                `json:"name"`
	Branch          string                `json:"branch"`
	UpdatedRelative string                `json:"updated_relative"`
	Commit          commitInfoResponse    `json:"commit"`
	Ahead           int                   `json:"ahead"`
	Behind          int                   `json:"behind"`
	Files           []changedFileResponse `json:"files"`
	FilesTruncated  bool                  `json:"files_truncated"`
	Error           string                `json:"error"`
}

type servicesResponse struct {
	RootPath string                     `json:"root_path"`
	Services []repositoryDetailResponse `json:"services"`
}

type selectionsResponse struct {
	SelectedCount int `json:"selected_count"`
}

func toWorkspaceResponse(config *application.WorkspaceConfig) workspaceResponse {
	selectedRepoNames := config.SelectedRepoNames
	if selectedRepoNames == nil {
		selectedRepoNames = []string{}
	}

	return workspaceResponse{
		Source:         string(config.Source),
		RootPath:       config.RootPath,
		HelperURL:      config.HelperURL,
		HasHelperToken: config.HasHelperToken,
		Selected:       selectedRepoNames,
	}
}

func toRepositoriesResponse(repositoryList *application.RepositoryList) repositoriesResponse {
	repositories := make([]repositorySummaryResponse, 0, len(repositoryList.Repositories))
	for _, summary := range repositoryList.Repositories {
		repositories = append(repositories, toRepositorySummaryResponse(summary))
	}

	return repositoriesResponse{
		RootPath:     repositoryList.RootPath,
		Repositories: repositories,
	}
}

func toRepositorySummaryResponse(summary domain.RepositorySummary) repositorySummaryResponse {
	return repositorySummaryResponse{
		Name:            summary.Name,
		IsGit:           summary.IsGit,
		Branch:          summary.Branch,
		DirtyCount:      summary.DirtyCount,
		UpdatedRelative: summary.UpdatedRelative,
		Error:           summary.Error,
	}
}

func toServicesResponse(serviceList *application.ServiceList) servicesResponse {
	services := make([]repositoryDetailResponse, 0, len(serviceList.Services))
	for _, detail := range serviceList.Services {
		services = append(services, toRepositoryDetailResponse(detail))
	}

	return servicesResponse{
		RootPath: serviceList.RootPath,
		Services: services,
	}
}

func toRepositoryDetailResponse(detail domain.RepositoryDetail) repositoryDetailResponse {
	files := make([]changedFileResponse, 0, len(detail.Files))
	for _, changedFile := range detail.Files {
		files = append(files, changedFileResponse{
			Status:  string(changedFile.Status),
			Path:    changedFile.Path,
			Added:   changedFile.Added,
			Deleted: changedFile.Deleted,
		})
	}

	return repositoryDetailResponse{
		Name:            detail.Name,
		Branch:          detail.Branch,
		UpdatedRelative: detail.UpdatedRelative,
		Commit: commitInfoResponse{
			Hash:         detail.Commit.Hash,
			Author:       detail.Commit.Author,
			RelativeTime: detail.Commit.RelativeTime,
			Subject:      detail.Commit.Subject,
		},
		Ahead:          detail.Ahead,
		Behind:         detail.Behind,
		Files:          files,
		FilesTruncated: detail.FilesTruncated,
		Error:          detail.Error,
	}
}

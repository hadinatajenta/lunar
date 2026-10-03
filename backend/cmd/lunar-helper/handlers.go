package main

import (
	"errors"
	"net/http"
	"strings"

	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/domain"
)

const (
	messageUnauthorized       = "unauthorized"
	messageOriginNotAllowed   = "origin is not allowed"
	messageInvalidRequestBody = "request body must be a JSON object with a root_path field"

	messageRootMissing      = "workspace root does not exist"
	messageRootNotDirectory = "workspace root is not a directory"
	messageRootNotPermitted = "workspace root is not permitted"
	messageRootOutsideScope = "workspace root is outside the allowed roots"
	messageRootInvalid      = "workspace root is invalid"

	messageRepositoryNameInvalid = "repository name is invalid"
	messageRepositoryOutsideRoot = "repository path escapes the workspace root"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type repositorySummaryResponse struct {
	Name            string `json:"name"`
	IsGit           bool   `json:"is_git"`
	Branch          string `json:"branch"`
	DirtyCount      int    `json:"dirty_count"`
	UpdatedRelative string `json:"updated_relative"`
	Error           string `json:"error"`
}

type changedFileResponse struct {
	Status  domain.ChangeStatus `json:"status"`
	Path    string              `json:"path"`
	Added   int                 `json:"added"`
	Deleted int                 `json:"deleted"`
}

type commitResponse struct {
	Hash         string `json:"hash"`
	Author       string `json:"author"`
	RelativeTime string `json:"relative_time"`
	Subject      string `json:"subject"`
}

type repositoryDetailResponse struct {
	Name            string                `json:"name"`
	Branch          string                `json:"branch"`
	UpdatedRelative string                `json:"updated_relative"`
	Commit          commitResponse        `json:"commit"`
	Ahead           int                   `json:"ahead"`
	Behind          int                   `json:"behind"`
	Files           []changedFileResponse `json:"files"`
	FilesTruncated  bool                  `json:"files_truncated"`
	Error           string                `json:"error"`
}

type repositoriesResponse struct {
	RootPath     string                      `json:"root_path"`
	Repositories []repositorySummaryResponse `json:"repositories"`
}

type servicesResponse struct {
	RootPath string                     `json:"root_path"`
	Services []repositoryDetailResponse `json:"services"`
}

type validatePathRequest struct {
	RootPath string `json:"root_path"`
}

type validatePathResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (s *helperServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	sharedHttp.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok", Version: s.version})
}

func (s *helperServer) handleToken(w http.ResponseWriter, _ *http.Request) {
	sharedHttp.WriteJSON(w, http.StatusOK, tokenResponse{Token: s.token})
}

func (s *helperServer) handleRepositories(w http.ResponseWriter, r *http.Request) {
	rootPath := r.URL.Query().Get(rootQueryParameter)

	summaries, err := s.reader.ListRepositories(r.Context(), rootPath)
	if err != nil {
		sharedHttp.WriteError(w, statusForRootError(err), rootErrorMessage(err))
		return
	}

	repositories := make([]repositorySummaryResponse, 0, len(summaries))
	for _, summary := range summaries {
		repositories = append(repositories, toRepositorySummaryResponse(summary))
	}

	sharedHttp.WriteJSON(w, http.StatusOK, repositoriesResponse{
		RootPath:     strings.TrimSpace(rootPath),
		Repositories: repositories,
	})
}

func (s *helperServer) handleServices(w http.ResponseWriter, r *http.Request) {
	rootPath := r.URL.Query().Get(rootQueryParameter)
	names := parseRepositoryNames(r.URL.Query().Get(namesQueryParameter))

	details, err := s.reader.ReadRepositories(r.Context(), rootPath, names)
	if err != nil {
		sharedHttp.WriteError(w, statusForRootError(err), rootErrorMessage(err))
		return
	}

	services := make([]repositoryDetailResponse, 0, len(details))
	for _, detail := range details {
		services = append(services, toRepositoryDetailResponse(detail))
	}

	sharedHttp.WriteJSON(w, http.StatusOK, servicesResponse{
		RootPath: strings.TrimSpace(rootPath),
		Services: services,
	})
}

func (s *helperServer) handleValidatePath(w http.ResponseWriter, r *http.Request) {
	var request validatePathRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteJSON(w, http.StatusBadRequest, validatePathResponse{Valid: false, Reason: messageInvalidRequestBody})
		return
	}

	if err := s.reader.ValidateRoot(r.Context(), request.RootPath); err != nil {
		sharedHttp.WriteJSON(w, http.StatusOK, validatePathResponse{Valid: false, Reason: rootErrorMessage(err)})
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, validatePathResponse{Valid: true, Reason: ""})
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

func toRepositoryDetailResponse(detail domain.RepositoryDetail) repositoryDetailResponse {
	files := make([]changedFileResponse, 0, len(detail.Files))
	for _, changedFile := range detail.Files {
		files = append(files, changedFileResponse{
			Status:  changedFile.Status,
			Path:    changedFile.Path,
			Added:   changedFile.Added,
			Deleted: changedFile.Deleted,
		})
	}

	return repositoryDetailResponse{
		Name:            detail.Name,
		Branch:          detail.Branch,
		UpdatedRelative: detail.UpdatedRelative,
		Commit: commitResponse{
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

func parseRepositoryNames(rawNames string) []string {
	names := make([]string, 0)
	for _, entry := range strings.Split(rawNames, ",") {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

func statusForRootError(err error) int {
	if errors.Is(err, domain.ErrRootNotFound) || errors.Is(err, domain.ErrRootNotDirectory) {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

func rootErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrRootNotFound):
		return messageRootMissing
	case errors.Is(err, domain.ErrRootNotDirectory):
		return messageRootNotDirectory
	case errors.Is(err, domain.ErrRootNotAllowed):
		return messageRootNotPermitted
	case errors.Is(err, domain.ErrRootOutsideScope):
		return messageRootOutsideScope
	case errors.Is(err, domain.ErrInvalidRepoName):
		return messageRepositoryNameInvalid
	case errors.Is(err, domain.ErrRepoOutsideRoot):
		return messageRepositoryOutsideRoot
	default:
		return messageRootInvalid
	}
}

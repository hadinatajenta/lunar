package main

import (
	"errors"
	"net/http"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	workspaceApplication "lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
	"lunar/backend/internal/workspace/infrastructure"
)

const (
	issueBranchesPath  = "/issues/{key}/branches"
	sprintBranchesPath = "/sprint/branches"
	keysQueryParameter = "keys"

	messageInvalidIssueKey          = "the issue key is invalid"
	messageWorkspaceRootForbidden   = "the workspace root is outside the allowed roots"
	messageWorkspaceRootUnavailable = "the workspace root is not available"
	messageGitScanFailed            = "the repository branches could not be read"
	messageGitScanTimedOut          = "the repository branch scan timed out"
)

type issueBranchServer struct {
	service *workspaceApplication.IssueBranchService
}

func registerIssueBranchRoutes(mux *http.ServeMux, dependencies helperDependencies) {
	service := workspaceApplication.NewIssueBranchService(infrastructure.NewGitInspector(dependencies.allowedRoots))
	server := &issueBranchServer{service: service}

	mux.Handle("GET "+issueBranchesPath, requireToken(dependencies.token, http.HandlerFunc(server.handleIssueBranches)))
	mux.Handle("GET "+sprintBranchesPath, requireToken(dependencies.token, http.HandlerFunc(server.handleSprintBranches)))
}

func (s *issueBranchServer) handleIssueBranches(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	branches, err := s.service.IssueBranchesAtRoot(r.Context(), rootQueryValue(r), r.PathValue("key"))
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, branches)
}

func (s *issueBranchServer) handleSprintBranches(w http.ResponseWriter, r *http.Request) {
	if !canExtendGitDeadline(w) {
		return
	}

	issueKeys := parseIssueKeys(r.URL.Query().Get(keysQueryParameter))

	branches, err := s.service.SprintBranchesAtRoot(r.Context(), rootQueryValue(r), issueKeys)
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, branches)
}

func parseIssueKeys(rawKeys string) []string {
	issueKeys := make([]string, 0)
	for _, entry := range strings.Split(rawKeys, ",") {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			issueKeys = append(issueKeys, trimmed)
		}
	}

	return issueKeys
}

func writeIssueBranchError(w http.ResponseWriter, err error) {
	status, message := issueBranchErrorResponse(err)
	sharedHttp.WriteError(w, status, message)
}

func issueBranchErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, sharedErrors.ErrBadRequest):
		return http.StatusBadRequest, messageInvalidIssueKey
	case errors.Is(err, domain.ErrRootOutsideScope):
		return http.StatusForbidden, messageWorkspaceRootForbidden
	case errors.Is(err, domain.ErrInvalidRoot),
		errors.Is(err, domain.ErrRootNotFound),
		errors.Is(err, domain.ErrRootNotDirectory),
		errors.Is(err, domain.ErrRootNotAllowed),
		errors.Is(err, domain.ErrRepoOutsideRoot),
		errors.Is(err, domain.ErrInvalidRepoName),
		errors.Is(err, domain.ErrNotGitRepository):
		return http.StatusBadRequest, messageWorkspaceRootUnavailable
	case errors.Is(err, domain.ErrGitTimeout):
		return http.StatusGatewayTimeout, messageGitScanTimedOut
	default:
		return http.StatusBadGateway, messageGitScanFailed
	}
}

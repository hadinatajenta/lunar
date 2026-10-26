package main

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	sharedHttp "lunar/backend/internal/shared/http"
	sopdomain "lunar/backend/internal/sop/domain"
	sopinfrastructure "lunar/backend/internal/sop/infrastructure"
	utdocsinfrastructure "lunar/backend/internal/utdocs/infrastructure"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	sopScanPath = "/sop/scan"

	sopScanRequestTimeout = 2 * time.Minute
	sopScanWriteDeadline  = 3 * time.Minute

	messageSOPScanInvalidBody       = "invalid SOP scan request"
	messageSOPScanFieldsRequired    = "root, repo, and branch are required"
	messageSOPScanBaseRefUnresolved = "the default base branch could not be resolved; choose a branch with an explicit base"
	messageSOPScanFailed            = "the SOP scan failed"
)

type sopScanRequest struct {
	Root    string `json:"root"`
	Repo    string `json:"repo"`
	Branch  string `json:"branch"`
	BaseRef string `json:"base_ref"`
}

type sopHandler struct {
	allowedRoots []string
	scanner      *sopinfrastructure.Scanner
	logger       *slog.Logger
}

func newSOPHandler(dependencies helperDependencies) *sopHandler {
	return &sopHandler{
		allowedRoots: dependencies.allowedRoots,
		scanner:      sopinfrastructure.NewScanner(),
		logger:       dependencies.logger,
	}
}

func registerSOPRoutes(mux *http.ServeMux, dependencies helperDependencies) {
	registerSOPHandler(mux, dependencies.token, newSOPHandler(dependencies))
}

func registerSOPHandler(mux *http.ServeMux, token string, handler *sopHandler) {
	mux.Handle("POST "+sopScanPath, requireToken(token, http.HandlerFunc(handler.handleScan)))
}

func (h *sopHandler) handleScan(w http.ResponseWriter, r *http.Request) {
	var request sopScanRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageSOPScanInvalidBody)
		return
	}

	root := strings.TrimSpace(request.Root)
	repo := strings.TrimSpace(request.Repo)
	branch := strings.TrimSpace(request.Branch)
	baseRef := strings.TrimSpace(request.BaseRef)
	if root == "" || repo == "" || branch == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageSOPScanFieldsRequired)
		return
	}
	if _, err := workspacedomain.ValidateBranchName(branch); err != nil {
		writeGitOperationError(w, err)
		return
	}
	if len(h.allowedRoots) == 0 {
		sharedHttp.WriteError(w, http.StatusForbidden, messageWriteAllowlist)
		return
	}

	resolvedRoot, err := workspacedomain.ResolveWorkspaceRoot(root, h.allowedRoots)
	if err != nil {
		writeGitOperationError(w, err)
		return
	}
	if _, err := workspacedomain.ValidateRepoName(repo); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, "repository name is invalid: "+repo)
		return
	}
	repositoryPath, err := workspacedomain.ResolveRepoPath(resolvedRoot, repo)
	if err != nil {
		writeGitOperationError(w, err)
		return
	}
	if h.scanner == nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageSOPScanFailed)
		return
	}

	if baseRef == "" {
		resolvedBaseRef, resolveErr := utdocsinfrastructure.DefaultBaseBranch(r.Context(), repositoryPath)
		if resolveErr != nil {
			sharedHttp.WriteError(w, http.StatusUnprocessableEntity, messageSOPScanBaseRefUnresolved)
			return
		}
		baseRef = resolvedBaseRef
	}
	if _, err := workspacedomain.ValidateBranchName(baseRef); err != nil {
		writeGitOperationError(w, err)
		return
	}

	if err := sharedHttp.ExtendWriteDeadline(w, sopScanWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageSOPScanFailed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sopScanRequestTimeout)
	defer cancel()

	scan, scanErr := h.scanner.Scan(ctx, sopdomain.ScanInput{
		RepositoryPath: repositoryPath,
		Repository:     repo,
		Branch:         branch,
		BaseRef:        baseRef,
	})
	if scanErr != nil {
		h.logger.Warn("cannot scan the SOP repository", "repository", repo, "branch", branch, "error", scanErr)
		scan = errorScan(repo, branch, baseRef, scanErr)
	}

	sharedHttp.WriteJSON(w, http.StatusOK, scan)
}

func errorScan(repo string, branch string, baseRef string, cause error) *sopdomain.RepositoryScan {
	return &sopdomain.RepositoryScan{
		Repository:    repo,
		Branch:        branch,
		BaseRef:       baseRef,
		DatabaseFiles: []sopdomain.DatabaseFile{},
		Tables:        []sopdomain.DatabaseTable{},
		Warnings:      []string{"the repository " + repo + " could not be scanned for branch " + branch + ": " + cause.Error()},
	}
}

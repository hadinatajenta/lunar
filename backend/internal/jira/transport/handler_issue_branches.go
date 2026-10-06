package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	workspaceApp "lunar/backend/internal/workspace/application"
	workspaceDomain "lunar/backend/internal/workspace/domain"
)

const (
	squadQueryParameter = "squad"
	boardQueryParameter = "board"

	activeSprintState = "active"

	unauthenticatedMessage    = "authenticated user is required"
	invalidIssueKeyMessage    = "the issue key is invalid"
	invalidBoardMessage       = "the board identifier is invalid"
	noWorkspaceMessage        = "no usable workspace is configured"
	workspacePathMessage      = "the workspace root is not available"
	workspaceForbiddenMessage = "the workspace root is outside the allowed roots"
	gitFailureMessage         = "the repository branches could not be read"
	gitTimeoutMessage         = "the repository branch scan timed out"
	upstreamMessage           = "Jira could not be reached"
	deadlineMessage           = "the request deadline could not be extended for the git operation"
)

type SprintIssueSource interface {
	GetBacklog(ctx context.Context, userID string, squad string) (*jiraDomain.JiraBacklogResponse, error)
}

type WorkspaceRootSource interface {
	WorkspaceRootFor(ctx context.Context, userID string) (string, error)
}

type IssueBranchHandler struct {
	branches       *workspaceApp.IssueBranchService
	workspaceRoots WorkspaceRootSource
	sprintIssues   SprintIssueSource
}

func NewIssueBranchHandler(branches *workspaceApp.IssueBranchService, workspaceRoots WorkspaceRootSource, sprintIssues SprintIssueSource) *IssueBranchHandler {
	return &IssueBranchHandler{
		branches:       branches,
		workspaceRoots: workspaceRoots,
		sprintIssues:   sprintIssues,
	}
}

func (h *IssueBranchHandler) GetIssueBranches(w http.ResponseWriter, r *http.Request) {
	if !extendIssueBranchDeadline(w) {
		return
	}

	userID := issueBranchUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	rootPath, err := h.workspaceRoots.WorkspaceRootFor(r.Context(), userID)
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	branches, err := h.branches.IssueBranchesAtRoot(r.Context(), rootPath, r.PathValue("key"))
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, branches)
}

func (h *IssueBranchHandler) GetSprintBranches(w http.ResponseWriter, r *http.Request) {
	if !extendIssueBranchDeadline(w) {
		return
	}

	userID := issueBranchUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	boardID, err := parseBoardID(r.URL.Query().Get(boardQueryParameter))
	if err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBoardMessage)
		return
	}

	rootPath, err := h.workspaceRoots.WorkspaceRootFor(r.Context(), userID)
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	backlog, err := h.sprintIssues.GetBacklog(r.Context(), userID, r.URL.Query().Get(squadQueryParameter))
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	branches, err := h.branches.SprintBranchesAtRoot(r.Context(), rootPath, selectSprintIssueKeys(backlog.Sprints, boardID))
	if err != nil {
		writeIssueBranchError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, branches)
}

func issueBranchUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}

	return user.UserID
}

func extendIssueBranchDeadline(w http.ResponseWriter) bool {
	if err := sharedHttp.ExtendWriteDeadline(w, workspaceDomain.GitWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, deadlineMessage)
		return false
	}

	return true
}

func parseBoardID(rawBoardID string) (int, error) {
	trimmed := strings.TrimSpace(rawBoardID)
	if trimmed == "" {
		return 0, nil
	}

	boardID, err := strconv.Atoi(trimmed)
	if err != nil || boardID < 0 {
		return 0, fmt.Errorf("board identifier %q is not a board number", trimmed)
	}

	return boardID, nil
}

func selectSprintIssueKeys(sprints []jiraDomain.JiraSprint, boardID int) []string {
	sprint, found := selectSprint(sprints, boardID)
	if !found {
		return []string{}
	}

	keys := make([]string, 0, len(sprint.Issues))
	seen := make(map[string]struct{}, len(sprint.Issues))
	for _, issue := range sprint.Issues {
		key := strings.ToUpper(strings.TrimSpace(issue.Key))
		if key == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}

		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func selectSprint(sprints []jiraDomain.JiraSprint, boardID int) (jiraDomain.JiraSprint, bool) {
	candidates := make([]jiraDomain.JiraSprint, 0, len(sprints))
	for _, sprint := range sprints {
		if boardID > 0 && sprint.OriginBoardID != boardID {
			continue
		}

		candidates = append(candidates, sprint)
	}

	for _, sprint := range candidates {
		if strings.EqualFold(sprint.State, activeSprintState) {
			return sprint, true
		}
	}

	if len(candidates) > 0 {
		return candidates[0], true
	}

	return jiraDomain.JiraSprint{}, false
}

func writeIssueBranchError(w http.ResponseWriter, err error) {
	status, message := issueBranchErrorResponse(err)
	sharedHttp.WriteError(w, status, message)
}

func issueBranchErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, workspaceApp.ErrWorkspaceNotUsable):
		return http.StatusUnprocessableEntity, noWorkspaceMessage
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		return http.StatusUnauthorized, unauthenticatedMessage
	case errors.Is(err, sharedErrors.ErrBadRequest):
		return http.StatusBadRequest, invalidIssueKeyMessage
	case errors.Is(err, workspaceDomain.ErrRootOutsideScope):
		return http.StatusForbidden, workspaceForbiddenMessage
	case errors.Is(err, workspaceDomain.ErrGitTimeout):
		return http.StatusGatewayTimeout, gitTimeoutMessage
	case errors.Is(err, workspaceDomain.ErrInvalidRoot),
		errors.Is(err, workspaceDomain.ErrRootNotFound),
		errors.Is(err, workspaceDomain.ErrRootNotDirectory),
		errors.Is(err, workspaceDomain.ErrRootNotAllowed),
		errors.Is(err, workspaceDomain.ErrRepoOutsideRoot),
		errors.Is(err, workspaceDomain.ErrInvalidRepoName),
		errors.Is(err, workspaceDomain.ErrNotGitRepository):
		return http.StatusBadRequest, workspacePathMessage
	case errors.Is(err, sharedErrors.ErrBadGateway):
		return http.StatusBadGateway, upstreamMessage
	default:
		return http.StatusBadGateway, gitFailureMessage
	}
}

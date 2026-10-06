package transport

import (
	"errors"
	"net/http"

	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
)

const (
	repoPathParameter   = "repo"
	allowedRootsMessage = "the workspace root is outside LUNAR_WORKSPACE_ALLOWED_ROOTS"
)

type baseBranchOptionResponse struct {
	Name               string  `json:"name"`
	LastCommitRelative *string `json:"last_commit_relative"`
	Source             string  `json:"source"`
}

type baseBranchOptionsResponse struct {
	Repo             string                     `json:"repo"`
	IsCloned         bool                       `json:"is_cloned"`
	Remembered       *string                    `json:"remembered"`
	BitbucketDefault *string                    `json:"bitbucket_default"`
	Options          []baseBranchOptionResponse `json:"options"`
	OtherBranchCount int                        `json:"other_branch_count"`
}

type createBranchTargetRequest struct {
	Repo         string `json:"repo"`
	BaseBranch   string `json:"base_branch"`
	RememberBase bool   `json:"remember_base"`
}

type createBranchesRequest struct {
	Key     string                      `json:"key"`
	Branch  string                      `json:"branch"`
	Targets []createBranchTargetRequest `json:"targets"`
}

type createdBranchResponse struct {
	Repo       string `json:"repo"`
	BaseBranch string `json:"base_branch"`
	HeadCommit string `json:"head_commit"`
}

type failedBranchResponse struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

type skippedBranchResponse struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

type createBranchesResponse struct {
	Key     string                  `json:"key"`
	Branch  string                  `json:"branch"`
	Created []createdBranchResponse `json:"created"`
	Failed  []failedBranchResponse  `json:"failed"`
	Skipped []skippedBranchResponse `json:"skipped"`
}

func (h *WorkspaceHandler) ListBaseBranchOptions(w http.ResponseWriter, r *http.Request) {
	if !h.extendGitWriteDeadline(w) {
		return
	}

	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	options, err := h.service.BaseBranchOptions(r.Context(), userID, r.PathValue(repoPathParameter))
	if err != nil {
		h.writeBranchError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, toBaseBranchOptionsResponse(*options))
}

func (h *WorkspaceHandler) CreateBranches(w http.ResponseWriter, r *http.Request) {
	if !h.extendGitWriteDeadline(w) {
		return
	}

	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	var request createBranchesRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidBodyMessage)
		return
	}

	result, err := h.service.CreateBranches(r.Context(), userID, toBranchCreationRequest(request))
	if err != nil {
		h.writeBranchError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusMultiStatus, toCreateBranchesResponse(*result))
}

func (h *WorkspaceHandler) writeBranchError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrRootOutsideScope) {
		sharedHttp.WriteError(w, http.StatusForbidden, allowedRootsMessage)
		return
	}

	h.writeError(w, err)
}

func toBranchCreationRequest(request createBranchesRequest) application.BranchCreationRequest {
	targets := make([]application.CreateBranchTarget, 0, len(request.Targets))
	for _, target := range request.Targets {
		targets = append(targets, application.CreateBranchTarget{
			Repo:         target.Repo,
			BaseBranch:   target.BaseBranch,
			RememberBase: target.RememberBase,
		})
	}

	return application.BranchCreationRequest{
		Key:     request.Key,
		Branch:  request.Branch,
		Targets: targets,
	}
}

func toBaseBranchOptionsResponse(options domain.BaseBranchOptions) baseBranchOptionsResponse {
	optionResponses := make([]baseBranchOptionResponse, 0, len(options.Options))
	for _, option := range options.Options {
		optionResponses = append(optionResponses, baseBranchOptionResponse{
			Name:               option.Name,
			LastCommitRelative: optionalString(option.LastCommitRelative),
			Source:             string(option.Source),
		})
	}

	return baseBranchOptionsResponse{
		Repo:             options.Repo,
		IsCloned:         options.IsCloned,
		Remembered:       optionalString(options.Remembered),
		BitbucketDefault: optionalString(options.BitbucketDefault),
		Options:          optionResponses,
		OtherBranchCount: options.OtherBranchCount,
	}
}

func toCreateBranchesResponse(result application.BranchCreationResult) createBranchesResponse {
	created := make([]createdBranchResponse, 0, len(result.Created))
	for _, branch := range result.Created {
		created = append(created, createdBranchResponse{
			Repo:       branch.Repo,
			BaseBranch: branch.BaseBranch,
			HeadCommit: branch.HeadCommit,
		})
	}

	failed := make([]failedBranchResponse, 0, len(result.Failed))
	for _, branch := range result.Failed {
		failed = append(failed, failedBranchResponse{Repo: branch.Repo, Reason: branch.Reason})
	}

	skipped := make([]skippedBranchResponse, 0, len(result.Skipped))
	for _, branch := range result.Skipped {
		skipped = append(skipped, skippedBranchResponse{Repo: branch.Repo, Reason: branch.Reason})
	}

	return createBranchesResponse{
		Key:     result.Key,
		Branch:  result.Branch,
		Created: created,
		Failed:  failed,
		Skipped: skipped,
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

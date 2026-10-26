package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	sharedHttp "lunar/backend/internal/shared/http"
	utdocsapplication "lunar/backend/internal/utdocs/application"
	utdocsdomain "lunar/backend/internal/utdocs/domain"
	utdocsinfrastructure "lunar/backend/internal/utdocs/infrastructure"
	utdocstransport "lunar/backend/internal/utdocs/transport"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	utRunsPath          = "/ut-runs"
	utRunScreenshotPath = "/ut-runs/{id}/screenshot.png"

	utRunRequestTimeout = 10 * time.Minute
	utRunWriteDeadline  = 11 * time.Minute

	utRunStoreDirectory = "ut-runs"

	messageUTRunInvalidBody         = "invalid unit test run request"
	messageUTRunFieldsRequired      = "root, repo, and branch are required"
	messageUTRunBaseRefUnresolved   = "the default base branch could not be resolved; choose a branch with an explicit base"
	messageUTRunDeadlineFailed      = "the request deadline could not be extended for the unit test run"
	messageUTRunStreamingFailed     = "streaming is not supported"
	messageUTRunFailed              = "the unit test run failed"
	messageUTRunStoreFailed         = "the unit test run report could not be stored"
	messageUTRunScreenshotNotFound  = "no screenshot exists for that run"
	messageUTRunScreenshotInvalidID = "the run id is invalid"
	messageUTRunRestoreFailed       = "the run finished but the repository branch and stashed changes could not be fully restored"
)

var utRunIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type utRunRequest struct {
	Root    string `json:"root"`
	Repo    string `json:"repo"`
	Branch  string `json:"branch"`
	BaseRef string `json:"base_ref"`
	Model   string `json:"model"`
}

type utdocsHandler struct {
	allowedRoots []string
	operator     workspacedomain.GitOperator
	restoreStash func(ctx context.Context, repositoryPath string) error
	logger       *slog.Logger
	service      *utdocsapplication.RunService
	store        *utdocsinfrastructure.RunStore
}

func newUTDocsHandler(dependencies helperDependencies) *utdocsHandler {
	return &utdocsHandler{
		allowedRoots: dependencies.allowedRoots,
		operator:     dependencies.operator,
		restoreStash: utdocsinfrastructure.RestoreStash,
		logger:       dependencies.logger,
		service: utdocsapplication.NewRunService(
			utdocsinfrastructure.NewScanner(),
			utdocsinfrastructure.NewRunner(),
			utdocsinfrastructure.NewScreenshotRenderer(),
		),
		store: utdocsinfrastructure.NewRunStore(defaultUTRunStoreDir()),
	}
}

func registerUTDocsRoutes(mux *http.ServeMux, dependencies helperDependencies) {
	registerUTDocsHandler(mux, dependencies.token, newUTDocsHandler(dependencies))
}

func registerUTDocsHandler(mux *http.ServeMux, token string, handler *utdocsHandler) {
	mux.Handle("POST "+utRunsPath, requireToken(token, http.HandlerFunc(handler.handleRun)))
	mux.Handle("GET "+utRunScreenshotPath, requireToken(token, http.HandlerFunc(handler.handleScreenshot)))
}

func defaultUTRunStoreDir() string {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDirectory, helperDirectoryName, utRunStoreDirectory)
}

func (h *utdocsHandler) handleRun(w http.ResponseWriter, r *http.Request) {
	var request utRunRequest
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageUTRunInvalidBody)
		return
	}

	root := strings.TrimSpace(request.Root)
	repo := strings.TrimSpace(request.Repo)
	branch := strings.TrimSpace(request.Branch)
	baseRef := strings.TrimSpace(request.BaseRef)
	if root == "" || repo == "" || branch == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageUTRunFieldsRequired)
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
	if h.operator == nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageUTRunFailed)
		return
	}

	if baseRef == "" {
		resolvedBaseRef, resolveErr := utdocsinfrastructure.DefaultBaseBranch(r.Context(), repositoryPath)
		if resolveErr != nil {
			sharedHttp.WriteError(w, http.StatusUnprocessableEntity, messageUTRunBaseRefUnresolved)
			return
		}
		baseRef = resolvedBaseRef
	}
	if _, err := workspacedomain.ValidateBranchName(baseRef); err != nil {
		writeGitOperationError(w, err)
		return
	}

	currentBranch, err := utdocsinfrastructure.CurrentBranch(r.Context(), repositoryPath)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusConflict, messageGitOperationFailed)
		return
	}

	if err := sharedHttp.ExtendWriteDeadline(w, utRunWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageUTRunDeadlineFailed)
		return
	}
	stream, err := utdocstransport.NewEventStream(w)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageUTRunStreamingFailed)
		return
	}

	switched := currentBranch != branch
	stashed := false
	if switched {
		prepared, prepareErr := h.prepareBranch(r.Context(), resolvedRoot, repo, repositoryPath, branch)
		if prepareErr != nil {
			writeGitOperationError(w, prepareErr)
			return
		}
		stashed = prepared
	}

	ctx, cancel := context.WithTimeout(r.Context(), utRunRequestTimeout)
	defer cancel()

	observer := &utRunStreamObserver{stream: stream}
	outcome, runErr := h.service.Execute(ctx, utdocsapplication.RunInput{
		RepositoryPath: repositoryPath,
		Repo:           repo,
		Branch:         branch,
		BaseRef:        baseRef,
	}, observer)

	restored := !switched
	var restoreErr error
	if switched {
		restored, restoreErr = h.restoreBranch(r.Context(), resolvedRoot, repo, repositoryPath, currentBranch, stashed)
	}

	if observer.WriteError() != nil {
		h.logger.Warn("cannot stream the unit test run", "error", observer.WriteError())
		return
	}
	if runErr != nil {
		if writeError := stream.Error(utRunFailureMessage(runErr)); writeError != nil {
			h.logger.Warn("cannot report the unit test run failure", "error", writeError)
		}
		return
	}

	runID := uuid.NewString()
	outcome.Payload.RunID = runID
	outcome.Payload.Stashed = stashed
	outcome.Payload.Restored = restored
	outcome.Payload.ScreenshotURL = utRunScreenshotURL(runID)
	if storeError := h.store.Save(outcome.Payload, outcome.Screenshot); storeError != nil {
		h.logger.Warn("cannot store the unit test run report", "error", storeError)
		if writeError := stream.Error(messageUTRunStoreFailed); writeError != nil {
			h.logger.Warn("cannot report the unit test run failure", "error", writeError)
		}
		return
	}

	if writeError := stream.Done(outcome.Payload); writeError != nil {
		h.logger.Warn("cannot write the unit test run result", "error", writeError)
	}
	if restoreErr != nil {
		h.logger.Warn("cannot fully restore the repository after the unit test run", "error", restoreErr)
		if writeError := stream.Error(messageUTRunRestoreFailed); writeError != nil {
			h.logger.Warn("cannot report the unit test restore failure", "error", writeError)
		}
	}
}

func (h *utdocsHandler) prepareBranch(ctx context.Context, resolvedRoot string, repo string, repositoryPath string, branch string) (bool, error) {
	dirty, err := utdocsinfrastructure.IsWorktreeDirty(ctx, repositoryPath)
	if err != nil {
		return false, err
	}

	stashed := false
	if dirty {
		if _, stashErr := h.operator.StashRepository(ctx, resolvedRoot, repo); stashErr != nil {
			return false, stashErr
		}
		stashed = true
	}

	if _, checkoutErr := h.operator.CheckoutBranch(ctx, resolvedRoot, repo, branch); checkoutErr != nil {
		if stashed {
			if restoreErr := h.restoreStash(ctx, repositoryPath); restoreErr != nil {
				h.logger.Warn("cannot restore the stash after a failed checkout", "error", restoreErr)
			}
		}
		return false, checkoutErr
	}
	return stashed, nil
}

func (h *utdocsHandler) restoreBranch(ctx context.Context, resolvedRoot string, repo string, repositoryPath string, originalBranch string, stashed bool) (bool, error) {
	if _, err := h.operator.CheckoutBranch(ctx, resolvedRoot, repo, originalBranch); err != nil {
		return false, fmt.Errorf("check out the original branch %s: %w", originalBranch, err)
	}
	if !stashed {
		return true, nil
	}
	if err := h.restoreStash(ctx, repositoryPath); err != nil {
		return false, fmt.Errorf("restore the stashed changes: %w", err)
	}
	return true, nil
}

func (h *utdocsHandler) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(r.PathValue("id"))
	if !utRunIDPattern.MatchString(runID) {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageUTRunScreenshotInvalidID)
		return
	}

	contents, err := h.store.LoadScreenshot(runID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			sharedHttp.WriteError(w, http.StatusNotFound, messageUTRunScreenshotNotFound)
			return
		}
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageUTRunFailed)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	if _, writeError := w.Write(contents); writeError != nil {
		h.logger.Warn("cannot write the unit test run screenshot", "error", writeError)
	}
}

func utRunScreenshotURL(runID string) string {
	return strings.Replace(utRunScreenshotPath, "{id}", runID, 1)
}

func utRunFailureMessage(err error) string {
	switch {
	case errors.Is(err, utdocsdomain.ErrRunTimeout):
		return "the unit test run exceeded its ten minute deadline"
	case errors.Is(err, workspacedomain.ErrInvalidBranchName):
		return messageBranchNameInvalid
	case errors.Is(err, workspacedomain.ErrBranchNotFound):
		return messageBranchNotFound
	default:
		return messageUTRunFailed
	}
}

type utRunStreamObserver struct {
	stream   *utdocstransport.EventStream
	writeErr error
}

func (o *utRunStreamObserver) Phase(phase string) {
	if o.writeErr != nil {
		return
	}
	o.writeErr = o.stream.Phase(phase)
}

func (o *utRunStreamObserver) Output(chunk string) {
	if o.writeErr != nil {
		return
	}
	o.writeErr = o.stream.Output(chunk)
}

func (o *utRunStreamObserver) WriteError() error {
	return o.writeErr
}

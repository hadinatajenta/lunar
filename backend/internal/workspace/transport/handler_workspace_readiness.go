package transport

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	sharedAuth "lunar/backend/internal/shared/auth"
	sharedHttp "lunar/backend/internal/shared/http"
	"lunar/backend/internal/workspace/application"
	"lunar/backend/internal/workspace/domain"
)

const (
	readinessRootQueryParameter    = "root"
	readinessProjectQueryParameter = "project"

	invalidProjectMessage        = "the project key is invalid"
	readinessScanFailedMessage   = "the workspace repositories could not be read"
	readinessScanTimedOutMessage = "the workspace repository scan timed out"
	readinessHelperSourceMessage = "the workspace is read by the local helper"
)

var readinessProjectExpression = regexp.MustCompile(`(?i)^[A-Z][A-Z0-9]{1,9}$`)

type WorkspaceConfigSource interface {
	GetWorkspace(ctx context.Context, userID string) (*application.WorkspaceConfig, error)
}

type ReadinessCatalogueSource interface {
	Catalogue(ctx context.Context, userID string) (*application.RepositoryCatalogue, error)
}

type WorkspaceReadinessHandler struct {
	readiness *application.WorkspaceReadinessService
	workspace WorkspaceConfigSource
	catalogue ReadinessCatalogueSource
}

func NewWorkspaceReadinessHandler(readiness *application.WorkspaceReadinessService, workspace WorkspaceConfigSource, catalogue ReadinessCatalogueSource) *WorkspaceReadinessHandler {
	return &WorkspaceReadinessHandler{readiness: readiness, workspace: workspace, catalogue: catalogue}
}

func RegisterWorkspaceReadinessRoute(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler, workspace WorkspaceConfigSource, scanner application.RepositoryBranchScanner, allowedRoots []string, catalogue ReadinessCatalogueSource) {
	service := application.NewWorkspaceReadinessService(scanner, allowedRoots)
	handler := NewWorkspaceReadinessHandler(service, workspace, catalogue)
	mux.Handle("GET /api/workspace/readiness", authMiddleware(http.HandlerFunc(handler.GetReadiness)))
}

func (h *WorkspaceReadinessHandler) GetReadiness(w http.ResponseWriter, r *http.Request) {
	if !canExtendReadinessDeadline(w) {
		return
	}
	if h.workspace == nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, internalMessage)
		return
	}

	user, found := sharedAuth.FromContext(r.Context())
	if !found || user == nil || user.UserID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, unauthenticatedMessage)
		return
	}

	request, valid := readinessRequest(r)
	if !valid {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidProjectMessage)
		return
	}

	config, err := h.workspace.GetWorkspace(r.Context(), user.UserID)
	if err != nil {
		writeWorkspaceReadinessError(w, err)
		return
	}

	if config.Source != domain.SourceServer {
		if strings.TrimSpace(config.RootPath) == "" {
			h.writeReadiness(w, r, request)
			return
		}
		sharedHttp.WriteError(w, http.StatusUnprocessableEntity, readinessHelperSourceMessage)
		return
	}
	request.RootPath = config.RootPath

	if h.catalogue != nil {
		catalogue, err := h.catalogue.Catalogue(r.Context(), user.UserID)
		if err != nil {
			writeWorkspaceReadinessError(w, err)
			return
		}
		request.RemoteRepositoryCount = catalogue.RemoteRepositoryCount
		request.MissingRepositories = missingCatalogueRepositories(catalogue)
	}

	h.writeReadiness(w, r, request)
}

func (h *WorkspaceReadinessHandler) GetHelperReadiness(w http.ResponseWriter, r *http.Request) {
	if !canExtendReadinessDeadline(w) {
		return
	}

	request, valid := readinessRequest(r)
	if !valid {
		sharedHttp.WriteError(w, http.StatusBadRequest, invalidProjectMessage)
		return
	}
	request.RootPath = strings.TrimSpace(r.URL.Query().Get(readinessRootQueryParameter))

	h.writeReadiness(w, r, request)
}

func (h *WorkspaceReadinessHandler) writeReadiness(w http.ResponseWriter, r *http.Request, request application.WorkspaceReadinessRequest) {
	readiness, err := h.readiness.ReadinessFor(r.Context(), request)
	if err != nil {
		writeWorkspaceReadinessError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, readiness)
}

func readinessRequest(r *http.Request) (application.WorkspaceReadinessRequest, bool) {
	rawProject := strings.TrimSpace(r.URL.Query().Get(readinessProjectQueryParameter))
	if rawProject == "" {
		return application.WorkspaceReadinessRequest{}, true
	}
	if !readinessProjectExpression.MatchString(rawProject) {
		return application.WorkspaceReadinessRequest{}, false
	}

	return application.WorkspaceReadinessRequest{ProjectKey: strings.ToUpper(rawProject)}, true
}

func missingCatalogueRepositories(catalogue *application.RepositoryCatalogue) []string {
	missing := make([]string, 0, len(catalogue.Repositories))
	for _, repository := range catalogue.Repositories {
		if repository.IsCloned || strings.TrimSpace(repository.Repo) == "" {
			continue
		}
		missing = append(missing, repository.Repo)
	}

	return missing
}

func canExtendReadinessDeadline(w http.ResponseWriter) bool {
	if err := sharedHttp.ExtendWriteDeadline(w, domain.GitWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, internalMessage)
		return false
	}

	return true
}

func writeWorkspaceReadinessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, domain.ErrGitTimeout):
		sharedHttp.WriteError(w, http.StatusGatewayTimeout, readinessScanTimedOutMessage)
	case errors.Is(err, application.ErrServerReaderUnavailable):
		sharedHttp.WriteError(w, http.StatusInternalServerError, serverReaderMessage)
	default:
		sharedHttp.WriteError(w, http.StatusBadGateway, readinessScanFailedMessage)
	}
}

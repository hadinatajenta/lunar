package transport

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"lunar/backend/internal/postman/application"
	"lunar/backend/internal/postman/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

const (
	exportRequestTimeout = 5 * time.Minute
	exportWriteDeadline  = 10 * time.Minute

	messageUnauthenticated       = "authenticated user is required"
	messageInvalidBody           = "invalid request body"
	messageRootInvalid           = "the workspace root must be an absolute path"
	messageNameRequired          = "a repository name is required"
	messageModelRequired         = "a model is required"
	messageStreamingUnsupported  = "streaming is not supported"
	messageDeadlineFailed        = "the request deadline could not be extended for the export"
	messageProviderKeyMissing    = "no provider key is configured for the selected model"
	messageProviderTimeout       = "the provider did not respond in time"
	messageExportFailed          = "the export failed"
	messageRepositoryMissing     = "the repository does not exist"
	messageRepositoryNameInvalid = "the repository name is invalid"
	messageWorkspacePath         = "the workspace or repository path is not available"
	messageRepositoryOutsideRoot = "the repository is outside the workspace root"
)

type Exporter interface {
	Export(ctx context.Context, userID string, request application.ExportRequest, emit application.Emitter) (application.Result, error)
}

type ExportHandler struct {
	exporter Exporter
}

func NewExportHandler(exporter Exporter) *ExportHandler {
	return &ExportHandler{exporter: exporter}
}

type exportRequestBody struct {
	Root  string `json:"root"`
	Name  string `json:"name"`
	Model string `json:"model"`
}

func (h *ExportHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID := readUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, messageUnauthenticated)
		return
	}

	var request exportRequestBody
	if err := sharedHttp.ParseJSON(r, &request); err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageInvalidBody)
		return
	}
	if rootHint := strings.TrimSpace(request.Root); rootHint != "" && !filepath.IsAbs(rootHint) {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageRootInvalid)
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageNameRequired)
		return
	}
	model := strings.TrimSpace(request.Model)
	if model == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, messageModelRequired)
		return
	}

	if err := sharedHttp.ExtendWriteDeadline(w, exportWriteDeadline); err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageDeadlineFailed)
		return
	}
	stream, err := NewStream(w)
	if err != nil {
		sharedHttp.WriteError(w, http.StatusInternalServerError, messageStreamingUnsupported)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), exportRequestTimeout)
	defer cancel()

	result, exportError := h.exporter.Export(ctx, userID, application.ExportRequest{
		RepositoryName: name,
		Model:          model,
	}, stream.Stage)
	if exportError != nil {
		if !stream.IsStarted() {
			status, message := FailureResponse(exportError)
			sharedHttp.WriteError(w, status, message)
			return
		}
		if writeError := stream.Error(FailureMessage(exportError)); writeError != nil {
			log.Printf("postman export: cannot report the failure: %v", writeError)
		}
		return
	}

	if writeError := stream.Result(result); writeError != nil {
		log.Printf("postman export: cannot write the result: %v", writeError)
	}
}

func readUserID(r *http.Request) string {
	user, ok := sharedAuth.FromContext(r.Context())
	if !ok || user == nil {
		return ""
	}
	return user.UserID
}

func FailureResponse(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrProviderKeyMissing):
		return http.StatusConflict, messageProviderKeyMissing
	case errors.Is(err, domain.ErrProviderTimeout):
		return http.StatusGatewayTimeout, messageProviderTimeout
	case errors.Is(err, domain.ErrServiceNotExportable), errors.Is(err, domain.ErrCollectionInvalid):
		return http.StatusUnprocessableEntity, FailureMessage(err)
	case errors.Is(err, workspacedomain.ErrInvalidRepoName):
		return http.StatusBadRequest, messageRepositoryNameInvalid
	case errors.Is(err, workspacedomain.ErrRootNotFound):
		return http.StatusNotFound, messageRepositoryMissing
	case errors.Is(err, workspacedomain.ErrInvalidRoot),
		errors.Is(err, workspacedomain.ErrRootNotDirectory),
		errors.Is(err, workspacedomain.ErrRootNotAllowed),
		errors.Is(err, workspacedomain.ErrRootOutsideScope):
		return http.StatusBadRequest, messageWorkspacePath
	case errors.Is(err, workspacedomain.ErrRepoOutsideRoot):
		return http.StatusBadRequest, messageRepositoryOutsideRoot
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		return http.StatusUnauthorized, messageUnauthenticated
	case errors.Is(err, sharedErrors.ErrBadRequest):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusBadGateway, messageExportFailed
	}
}

func FailureMessage(err error) string {
	if errors.Is(err, domain.ErrServiceNotExportable) || errors.Is(err, domain.ErrCollectionInvalid) {
		return err.Error()
	}
	return messageExportFailed
}

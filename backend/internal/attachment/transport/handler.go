package transport

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"lunar/backend/internal/attachment/application"
	"lunar/backend/internal/attachment/domain"
	sharedAuth "lunar/backend/internal/shared/auth"
	sharedErrors "lunar/backend/internal/shared/errors"
	sharedHttp "lunar/backend/internal/shared/http"
)

const (
	uploadFieldName = "file"

	multipartRequiredMessage = "the request must be sent as multipart/form-data with a file field named file"
	oversizedFileMessage     = "the file exceeds the 10 MB limit"
	documentReadMessage      = "the attached document cannot be read"
	storeFailureMessage      = "the attachment cannot be stored"
	notFoundMessage          = "the attachment was not found"
)

var errFileTooLarge = errors.New(oversizedFileMessage)

type AttachmentHandler struct {
	service *application.Service
}

func NewAttachmentHandler(service *application.Service) *AttachmentHandler {
	return &AttachmentHandler{service: service}
}

type uploadResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Size    int64  `json:"size"`
	Preview string `json:"preview"`
}

type loadResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Preview     string `json:"preview"`
	Text        string `json:"text,omitempty"`
	DataURL     string `json:"data_url,omitempty"`
}

func (h *AttachmentHandler) getUserID(r *http.Request) string {
	userCtx, ok := sharedAuth.FromContext(r.Context())
	if !ok || userCtx == nil {
		return ""
	}
	return userCtx.UserID
}

func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, application.MaxRequestBodyBytes)

	multipartReader, err := r.MultipartReader()
	if err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, multipartRequiredMessage)
		return
	}

	fileName, content, err := readUploadedFile(multipartReader)
	if err != nil {
		writeUploadError(w, err)
		return
	}

	attachment, err := h.service.Upload(r.Context(), userID, fileName, content)
	if err != nil {
		writeUploadError(w, err)
		return
	}

	sharedHttp.WriteJSON(w, http.StatusCreated, uploadResponse{
		ID:      attachment.ID,
		Name:    attachment.Name,
		Kind:    string(attachment.Kind),
		Size:    attachment.Size,
		Preview: attachment.Preview,
	})
}

func (h *AttachmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	attachment, err := h.service.Load(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		switch {
		case errors.Is(err, sharedErrors.ErrNotFound):
			sharedHttp.WriteError(w, http.StatusNotFound, notFoundMessage)
		case errors.Is(err, sharedErrors.ErrUnauthorized):
			sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
		case errors.Is(err, sharedErrors.ErrBadRequest):
			sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			sharedHttp.WriteError(w, http.StatusInternalServerError, "the attachment cannot be loaded")
		}
		return
	}

	sharedHttp.WriteJSON(w, http.StatusOK, buildLoadResponse(attachment))
}

func buildLoadResponse(attachment *domain.Attachment) loadResponse {
	response := loadResponse{
		ID:          attachment.ID,
		Name:        attachment.Name,
		Kind:        string(attachment.Kind),
		ContentType: attachment.ContentType,
		Size:        attachment.Size,
		Preview:     attachment.Preview,
	}
	if attachment.Kind == domain.KindImage {
		response.DataURL = application.BuildImageDataURL(attachment.ContentType, attachment.Content)
		return response
	}
	response.Text = attachment.Text
	return response
}

func readUploadedFile(reader *multipart.Reader) (string, []byte, error) {
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return "", nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, multipartRequiredMessage)
		}
		if err != nil {
			return "", nil, err
		}

		if part.FormName() != uploadFieldName {
			if closeErr := part.Close(); closeErr != nil {
				return "", nil, fmt.Errorf("close unused multipart part: %w", closeErr)
			}
			continue
		}

		content, readErr := io.ReadAll(io.LimitReader(part, application.MaxFileBytes+1))
		closeErr := part.Close()
		if readErr != nil {
			return "", nil, readErr
		}
		if closeErr != nil {
			return "", nil, fmt.Errorf("close file part: %w", closeErr)
		}
		if int64(len(content)) > application.MaxFileBytes {
			return "", nil, errFileTooLarge
		}
		return strings.TrimSpace(part.FileName()), content, nil
	}
}

func writeUploadError(w http.ResponseWriter, err error) {
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.Is(err, errFileTooLarge), errors.As(err, &maxBytesError):
		sharedHttp.WriteError(w, http.StatusBadRequest, oversizedFileMessage)
	case errors.Is(err, application.ErrUnsupportedType):
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, application.ErrUnreadableDocument):
		sharedHttp.WriteError(w, http.StatusBadRequest, documentReadMessage)
	case errors.Is(err, sharedErrors.ErrBadRequest):
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, sharedErrors.ErrUnauthorized):
		sharedHttp.WriteError(w, http.StatusUnauthorized, "unauthorized")
	default:
		sharedHttp.WriteError(w, http.StatusInternalServerError, storeFailureMessage)
	}
}

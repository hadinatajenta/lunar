package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	agentinfrastructure "lunar/backend/internal/agent/infrastructure"
	attachmentapplication "lunar/backend/internal/attachment/application"
)

const (
	attachmentFetchTimeout = 30 * time.Second
	maxAgentAttachments    = 5
	attachmentKindImage    = "image"
	attachmentKindDocument = "document"

	imageUnsupportedMessageTemplate = "the selected model %q cannot read images: choose an image-capable model such as DeepSeek Flash"
)

func validateImageAttachments(
	requestedModel string,
	resolvedModel string,
	imageCount int,
	client *agentinfrastructure.ChatClient,
) error {
	if imageCount == 0 || client.SupportsImageInput(resolvedModel) {
		return nil
	}
	return fmt.Errorf(imageUnsupportedMessageTemplate, requestedModel)
}

type attachmentPayload struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Text    string `json:"text"`
	DataURL string `json:"data_url"`
}

func loadAttachmentContext(
	ctx context.Context,
	backendURL string,
	sessionToken string,
	attachmentIDs []string,
) (string, []string, error) {
	if len(attachmentIDs) == 0 {
		return "", nil, nil
	}
	if len(attachmentIDs) > maxAgentAttachments {
		return "", nil, fmt.Errorf("at most %d attachments are allowed per question", maxAgentAttachments)
	}

	documents := make([]attachmentapplication.DocumentText, 0, len(attachmentIDs))
	imageDataURLs := make([]string, 0, len(attachmentIDs))

	for _, rawID := range attachmentIDs {
		attachmentID := strings.TrimSpace(rawID)
		if err := attachmentapplication.ValidateAttachmentID(attachmentID); err != nil {
			return "", nil, fmt.Errorf("the attachment id %q is invalid", rawID)
		}

		payload, err := fetchAttachment(ctx, backendURL, sessionToken, attachmentID)
		if err != nil {
			return "", nil, err
		}

		switch payload.Kind {
		case attachmentKindDocument:
			documents = append(documents, attachmentapplication.DocumentText{Name: payload.Name, Text: payload.Text})
		case attachmentKindImage:
			if strings.TrimSpace(payload.DataURL) == "" {
				return "", nil, fmt.Errorf("the attachment %s has no image data", attachmentID)
			}
			imageDataURLs = append(imageDataURLs, payload.DataURL)
		default:
			return "", nil, fmt.Errorf("the attachment %s has an unknown kind %q", attachmentID, payload.Kind)
		}
	}

	return attachmentapplication.BuildAttachmentContext(documents), imageDataURLs, nil
}

func fetchAttachment(
	ctx context.Context,
	backendURL string,
	sessionToken string,
	attachmentID string,
) (attachmentPayload, error) {
	endpoint := backendURL + "/api/copilot/attachments/" + url.PathEscape(attachmentID)

	requestContext, cancel := context.WithTimeout(ctx, attachmentFetchTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if err != nil {
		return attachmentPayload{}, fmt.Errorf("fetchAttachment: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+sessionToken)

	client := &http.Client{Timeout: attachmentFetchTimeout}
	response, err := client.Do(request)
	if err != nil {
		return attachmentPayload{}, fmt.Errorf("fetchAttachment: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return attachmentPayload{}, fmt.Errorf("the attachment %s was not found", attachmentID)
	}
	if response.StatusCode != http.StatusOK {
		return attachmentPayload{}, fmt.Errorf("fetchAttachment: unexpected status %d", response.StatusCode)
	}

	var payload attachmentPayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return attachmentPayload{}, fmt.Errorf("fetchAttachment: %w", err)
	}
	return payload, nil
}

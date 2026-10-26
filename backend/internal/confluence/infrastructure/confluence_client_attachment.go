package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

const atlassianNoCheckHeader = "X-Atlassian-Token"

type confluenceAttachment struct {
	ID string `json:"id"`
}

type confluenceAttachmentResponse struct {
	Results []confluenceAttachment `json:"results"`
}

func (c *ConfluenceClient) UploadAttachment(ctx context.Context, pat string, pageID string, filename string, content []byte, mediaType string) (string, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return "", err
	}
	trimmedFilename := strings.TrimSpace(filename)
	if trimmedFilename == "" {
		return "", fmt.Errorf("%w: attachment filename is required", sharedErrors.ErrBadRequest)
	}
	if len(content) == 0 {
		return "", fmt.Errorf("%w: attachment content is required", sharedErrors.ErrBadRequest)
	}
	trimmedMediaType := strings.TrimSpace(mediaType)
	if trimmedMediaType == "" {
		return "", fmt.Errorf("%w: attachment media type is required", sharedErrors.ErrBadRequest)
	}

	createURL := c.baseURL + "/rest/api/content/" + url.PathEscape(trimmedID) + "/child/attachment"
	attachmentID, status, err := c.postAttachment(ctx, pat, createURL, trimmedFilename, content, trimmedMediaType)
	if err != nil {
		return "", err
	}
	if status == http.StatusOK || status == http.StatusCreated {
		if attachmentID == "" {
			return "", fmt.Errorf("confluence did not return an attachment id")
		}
		return attachmentID, nil
	}
	if status != http.StatusBadRequest && status != http.StatusConflict {
		return "", fmt.Errorf("confluence returned status %d while uploading the attachment", status)
	}

	existingID, err := c.findAttachmentID(ctx, pat, trimmedID, trimmedFilename)
	if err != nil {
		return "", err
	}
	if existingID == "" {
		return "", fmt.Errorf("confluence rejected the attachment upload with status %d", status)
	}

	replaceURL := createURL + "/" + url.PathEscape(existingID) + "/data"
	replacedID, replaceStatus, err := c.postAttachment(ctx, pat, replaceURL, trimmedFilename, content, trimmedMediaType)
	if err != nil {
		return "", err
	}
	if replaceStatus != http.StatusOK && replaceStatus != http.StatusCreated {
		return "", fmt.Errorf("confluence returned status %d while replacing the attachment", replaceStatus)
	}
	if replacedID != "" {
		return replacedID, nil
	}
	return existingID, nil
}

func (c *ConfluenceClient) findAttachmentID(ctx context.Context, pat string, pageID string, filename string) (string, error) {
	query := url.Values{"filename": []string{filename}}
	requestURL := c.baseURL + "/rest/api/content/" + url.PathEscape(pageID) + "/child/attachment?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, requestURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return "", err
	}

	var payload confluenceAttachmentResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to parse the Confluence attachment list: %w", err)
	}
	if len(payload.Results) == 0 {
		return "", nil
	}
	return payload.Results[0].ID, nil
}

func (c *ConfluenceClient) postAttachment(ctx context.Context, pat string, requestURL string, filename string, content []byte, mediaType string) (string, int, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	partHeader.Set("Content-Type", mediaType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return "", 0, fmt.Errorf("prepare the attachment upload: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return "", 0, fmt.Errorf("write the attachment content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", 0, fmt.Errorf("finalize the attachment upload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, &body)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")
	req.Header.Set(atlassianNoCheckHeader, "no-check")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("unable to reach Confluence API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", resp.StatusCode, fmt.Errorf("%w: invalid or expired Confluence PAT", sharedErrors.ErrUnauthorized)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 512))
		return "", resp.StatusCode, nil
	}

	var payload confluenceAttachmentResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&payload); err != nil {
		return "", resp.StatusCode, fmt.Errorf("failed to parse the Confluence attachment response: %w", err)
	}
	if len(payload.Results) == 0 {
		return "", resp.StatusCode, nil
	}
	return payload.Results[0].ID, resp.StatusCode, nil
}

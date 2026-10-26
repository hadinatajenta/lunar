package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
)

type confluenceCreatePageRequest struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Space struct {
		Key string `json:"key"`
	} `json:"space"`
	Ancestors []struct {
		ID string `json:"id"`
	} `json:"ancestors"`
	Body struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
}

func (c *ConfluenceClient) GetPageSpaceKey(ctx context.Context, pat string, pageID string) (string, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return "", err
	}

	query := url.Values{"expand": []string{"space"}}
	requestURL := c.baseURL + "/rest/api/content/" + url.PathEscape(trimmedID) + "?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, requestURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", pageNotFoundError(trimmedID)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return "", err
	}

	var content confluenceContent
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&content); err != nil {
		return "", fmt.Errorf("failed to parse the Confluence page space: %w", err)
	}

	spaceKey := strings.TrimSpace(content.Space.Key)
	if spaceKey == "" {
		return "", fmt.Errorf("%w: the Confluence page %s has no space key", sharedErrors.ErrBadGateway, trimmedID)
	}
	return spaceKey, nil
}

func (c *ConfluenceClient) GetPageVersion(ctx context.Context, pat string, pageID string) (int, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return 0, err
	}

	current, err := c.getPageVersion(ctx, pat, trimmedID)
	if err != nil {
		return 0, err
	}
	return current.Version.Number, nil
}

func (c *ConfluenceClient) CreatePage(ctx context.Context, pat string, spaceKey string, parentID string, title string, storage string) (string, int, string, error) {
	if _, err := validatePageTreeID(pat, parentID); err != nil {
		return "", 0, "", err
	}

	trimmedSpace := strings.TrimSpace(spaceKey)
	if trimmedSpace == "" {
		return "", 0, "", fmt.Errorf("%w: space key is required", sharedErrors.ErrBadRequest)
	}
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return "", 0, "", fmt.Errorf("%w: page title is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(storage) == "" {
		return "", 0, "", fmt.Errorf("%w: page storage is required", sharedErrors.ErrBadRequest)
	}

	create := confluenceCreatePageRequest{Type: "page", Title: trimmedTitle}
	create.Space.Key = trimmedSpace
	create.Ancestors = []struct {
		ID string `json:"id"`
	}{{ID: parentID}}
	create.Body.Storage.Value = storage
	create.Body.Storage.Representation = "storage"

	payload, err := json.Marshal(create)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to encode the Confluence page creation: %w", err)
	}

	requestURL := c.baseURL + "/rest/api/content"
	resp, err := c.doRequestWithBody(ctx, pat, http.MethodPost, requestURL, payload)
	if err != nil {
		return "", 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return "", 0, "", fmt.Errorf("%w: a Confluence page with the title %s already exists", sharedErrors.ErrConflict, trimmedTitle)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return "", 0, "", err
	}

	var created confluencePageVersionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&created); err != nil {
		return "", 0, "", fmt.Errorf("failed to parse the Confluence page creation response: %w", err)
	}
	if created.ID == "" {
		return "", 0, "", fmt.Errorf("confluence did not return the created page id")
	}
	return created.ID, created.Version.Number, created.Links.WebUI, nil
}

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

type confluencePageVersionResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Links confluenceLinks `json:"_links"`
}

type confluenceUpdatePageRequest struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value          string `json:"value"`
			Representation string `json:"representation"`
		} `json:"storage"`
	} `json:"body"`
}

func (c *ConfluenceClient) UpdatePageStorage(ctx context.Context, pat string, pageID string, storage string) (int, string, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return 0, "", err
	}
	if strings.TrimSpace(storage) == "" {
		return 0, "", fmt.Errorf("%w: storage value is required", sharedErrors.ErrBadRequest)
	}

	current, err := c.getPageVersion(ctx, pat, trimmedID)
	if err != nil {
		return 0, "", err
	}

	nextVersion := current.Version.Number + 1
	update := confluenceUpdatePageRequest{ID: trimmedID, Type: "page", Title: current.Title}
	update.Version.Number = nextVersion
	update.Body.Storage.Value = storage
	update.Body.Storage.Representation = "storage"

	payload, err := json.Marshal(update)
	if err != nil {
		return 0, "", fmt.Errorf("failed to encode the Confluence page update: %w", err)
	}

	requestURL := c.baseURL + "/rest/api/content/" + url.PathEscape(trimmedID)
	resp, err := c.doRequestWithBody(ctx, pat, http.MethodPut, requestURL, payload)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, "", pageNotFoundError(trimmedID)
	}
	if resp.StatusCode == http.StatusConflict {
		return 0, "", fmt.Errorf("%w: page %s was changed by someone else, retry the apply", sharedErrors.ErrConflict, trimmedID)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return 0, "", err
	}

	var updated confluencePageVersionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&updated); err != nil {
		return 0, "", fmt.Errorf("failed to parse the Confluence page update response: %w", err)
	}

	if updated.Version.Number > 0 {
		nextVersion = updated.Version.Number
	}
	webURL := updated.Links.WebUI
	if webURL == "" {
		webURL = current.Links.WebUI
	}
	return nextVersion, webURL, nil
}

func (c *ConfluenceClient) getPageVersion(ctx context.Context, pat string, pageID string) (confluencePageVersionResponse, error) {
	query := url.Values{"expand": []string{"version,space"}}
	requestURL := c.baseURL + "/rest/api/content/" + url.PathEscape(pageID) + "?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, requestURL)
	if err != nil {
		return confluencePageVersionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return confluencePageVersionResponse{}, pageNotFoundError(pageID)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return confluencePageVersionResponse{}, err
	}

	var current confluencePageVersionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&current); err != nil {
		return confluencePageVersionResponse{}, fmt.Errorf("failed to parse the Confluence page version: %w", err)
	}
	return current, nil
}

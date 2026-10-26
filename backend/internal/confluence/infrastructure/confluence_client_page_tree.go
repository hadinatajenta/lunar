package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"lunar/backend/internal/confluence/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	CONFLUENCE_PAGE_TREE_RESPONSE_BYTES = 8 << 20
	CONFLUENCE_CHILD_PAGE_LIMIT         = "100"
)

type confluenceAncestorsResponse struct {
	Ancestors []confluenceContent `json:"ancestors"`
}

func (c *ConfluenceClient) GetPageAncestors(ctx context.Context, pat string, pageID string) ([]domain.ConfluencePageRef, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return nil, err
	}

	query := url.Values{"expand": []string{"ancestors"}}
	requestURL := c.baseURL + "/rest/api/content/" + url.PathEscape(trimmedID) + "?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pageNotFoundError(trimmedID)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return nil, err
	}

	var payload confluenceAncestorsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to parse Confluence page ancestors: %w", err)
	}

	ancestors := make([]domain.ConfluencePageRef, 0, len(payload.Ancestors))
	for _, ancestor := range payload.Ancestors {
		ancestors = append(ancestors, domain.ConfluencePageRef{ID: ancestor.ID, Title: ancestor.Title})
	}
	return ancestors, nil
}

func (c *ConfluenceClient) GetChildPages(ctx context.Context, pat string, pageID string) ([]domain.ConfluencePageRef, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return nil, err
	}

	query := url.Values{"limit": []string{CONFLUENCE_CHILD_PAGE_LIMIT}}
	requestURL := c.baseURL + "/rest/api/content/" + url.PathEscape(trimmedID) + "/child/page?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, pageNotFoundError(trimmedID)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return nil, err
	}

	var payload confluenceSearchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, CONFLUENCE_PAGE_TREE_RESPONSE_BYTES)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to parse Confluence child pages: %w", err)
	}

	children := make([]domain.ConfluencePageRef, 0, len(payload.Results))
	for _, child := range payload.Results {
		children = append(children, domain.ConfluencePageRef{ID: child.ID, Title: child.Title})
	}
	return children, nil
}

func (c *ConfluenceClient) GetPageStorage(ctx context.Context, pat string, pageID string) (string, error) {
	trimmedID, err := validatePageTreeID(pat, pageID)
	if err != nil {
		return "", err
	}

	query := url.Values{"expand": []string{"body.storage"}}
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
		return "", fmt.Errorf("failed to parse Confluence page storage: %w", err)
	}
	return content.Body.Storage.Value, nil
}

func validatePageTreeID(pat string, pageID string) (string, error) {
	trimmedID := strings.TrimSpace(pageID)
	if trimmedID == "" {
		return "", fmt.Errorf("%w: page id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(pat) == "" {
		return "", fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}
	return trimmedID, nil
}

func pageNotFoundError(pageID string) error {
	return fmt.Errorf("%w: page %s not found", sharedErrors.ErrNotFound, pageID)
}

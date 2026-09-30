package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"lunar/backend/internal/confluence/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const (
	CONFLUENCE_DIAL_TIMEOUT            = 5 * time.Second
	CONFLUENCE_TLS_HANDSHAKE_TIMEOUT   = 5 * time.Second
	CONFLUENCE_RESPONSE_HEADER_TIMEOUT = 15 * time.Second
	CONFLUENCE_IDLE_CONN_TIMEOUT       = 30 * time.Second
	CONFLUENCE_CLIENT_TIMEOUT          = 20 * time.Second

	CONFLUENCE_CONTENT_WORKERS = 5
)

type confluenceLinks struct {
	Base  string `json:"base"`
	WebUI string `json:"webui"`
}

type confluenceLabel struct {
	Name string `json:"name"`
}

type confluenceUser struct {
	DisplayName string `json:"displayName"`
}

type confluenceSpace struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type confluenceVersion struct {
	When string         `json:"when"`
	By   confluenceUser `json:"by"`
}

type confluenceMetadata struct {
	Labels struct {
		Results []confluenceLabel `json:"results"`
	} `json:"labels"`
}

type confluenceBody struct {
	Storage struct {
		Value string `json:"value"`
	} `json:"storage"`
}

type confluenceContent struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Space    confluenceSpace    `json:"space"`
	Version  confluenceVersion  `json:"version"`
	Metadata confluenceMetadata `json:"metadata"`
	Body     confluenceBody     `json:"body"`
	Links    confluenceLinks    `json:"_links"`
}

type confluenceSearchResponse struct {
	Results []confluenceContent `json:"results"`
}

type ConfluenceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewConfluenceClient(baseURL string) *ConfluenceClient {
	cleanURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if cleanURL == "" {
		cleanURL = "https://confluence.bri.co.id"
	}

	return &ConfluenceClient{
		baseURL: cleanURL,
		httpClient: &http.Client{
			Timeout: CONFLUENCE_CLIENT_TIMEOUT,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: CONFLUENCE_DIAL_TIMEOUT,
				}).DialContext,
				TLSHandshakeTimeout:   CONFLUENCE_TLS_HANDSHAKE_TIMEOUT,
				ResponseHeaderTimeout: CONFLUENCE_RESPONSE_HEADER_TIMEOUT,
				IdleConnTimeout:       CONFLUENCE_IDLE_CONN_TIMEOUT,
				ForceAttemptHTTP2:     true,
			},
		},
	}
}

func (c *ConfluenceClient) ListDocuments(ctx context.Context, pat string) ([]domain.ConfluenceDocument, error) {
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	query := url.Values{
		"cql":    []string{"type=page"},
		"limit":  []string{"100"},
		"expand": []string{"version,space"},
	}
	searchURL := c.baseURL + "/rest/api/content/search?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ensureSuccessStatus(resp); err != nil {
		return nil, err
	}

	var searchResp confluenceSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse Confluence search response: %w", err)
	}

	documents := make([]domain.ConfluenceDocument, 0, len(searchResp.Results))
	for _, content := range searchResp.Results {
		documents = append(documents, buildDocumentFromContent(content, c.baseURL))
	}
	return documents, nil
}

func (c *ConfluenceClient) GetDocumentDetail(ctx context.Context, pat string, documentID string) (*domain.ConfluenceDocument, error) {
	trimmedID := strings.TrimSpace(documentID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: document id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	query := url.Values{"expand": []string{"body.storage,version,space,metadata.labels"}}
	detailURL := c.baseURL + "/rest/api/content/" + url.PathEscape(trimmedID) + "?" + query.Encode()

	resp, err := c.doRequest(ctx, pat, detailURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: document %s not found", sharedErrors.ErrNotFound, trimmedID)
	}
	if err := ensureSuccessStatus(resp); err != nil {
		return nil, err
	}

	var content confluenceContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to parse Confluence document detail: %w", err)
	}

	document := buildDocumentFromContent(content, c.baseURL)
	return &document, nil
}

func (c *ConfluenceClient) GetDocumentsByIDs(ctx context.Context, pat string, documentIDs []string) ([]domain.ConfluenceDocument, error) {
	if len(documentIDs) == 0 {
		return []domain.ConfluenceDocument{}, nil
	}
	if strings.TrimSpace(pat) == "" {
		return nil, fmt.Errorf("%w: Confluence PAT is not configured", sharedErrors.ErrUnauthorized)
	}

	type fetchResult struct {
		document domain.ConfluenceDocument
		err      error
		isFound  bool
	}

	results := make([]fetchResult, len(documentIDs))
	semaphore := make(chan struct{}, CONFLUENCE_CONTENT_WORKERS)
	var wg sync.WaitGroup

	for i, documentID := range documentIDs {
		wg.Add(1)
		go func(index int, docID string) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				results[index] = fetchResult{err: ctx.Err()}
				return
			}
			defer func() { <-semaphore }()

			if err := ctx.Err(); err != nil {
				results[index] = fetchResult{err: err}
				return
			}

			query := url.Values{"expand": []string{"version,space"}}
			fetchURL := c.baseURL + "/rest/api/content/" + url.PathEscape(docID) + "?" + query.Encode()
			resp, err := c.doRequest(ctx, pat, fetchURL)
			if err != nil {
				results[index] = fetchResult{err: err}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				results[index] = fetchResult{isFound: false}
				return
			}
			if err := ensureSuccessStatus(resp); err != nil {
				results[index] = fetchResult{err: err}
				return
			}

			var content confluenceContent
			if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
				results[index] = fetchResult{err: fmt.Errorf("parse document %s: %w", docID, err)}
				return
			}
			results[index] = fetchResult{
				document: buildDocumentFromContent(content, c.baseURL),
				isFound:  true,
			}
		}(i, documentID)
	}

	wg.Wait()

	documents := make([]domain.ConfluenceDocument, 0, len(documentIDs))
	var firstErr error
	for _, result := range results {
		if result.err != nil {
			if firstErr == nil {
				firstErr = result.err
			}
			continue
		}
		if result.isFound {
			documents = append(documents, result.document)
		}
	}
	return documents, firstErr
}

func (c *ConfluenceClient) doRequest(ctx context.Context, pat string, requestURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lunar-Workspace/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to reach Confluence API: %w", err)
	}
	return resp, nil
}

func ensureSuccessStatus(resp *http.Response) error {
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: invalid or expired Confluence PAT", sharedErrors.ErrUnauthorized)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("confluence returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

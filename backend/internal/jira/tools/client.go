package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	requestTimeout   = 30 * time.Second
	maxResponseBytes = 4 << 20
)

type BackendClient struct {
	baseURL      string
	sessionToken string
	httpClient   *http.Client
}

func NewBackendClient(baseURL string, sessionToken string) *BackendClient {
	return &BackendClient{
		baseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		sessionToken: strings.TrimSpace(sessionToken),
		httpClient:   &http.Client{Timeout: requestTimeout},
	}
}

func (c *BackendClient) getJSON(ctx context.Context, path string) ([]byte, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("the backend URL is not configured")
	}
	if c.sessionToken == "" {
		return nil, fmt.Errorf("the session token is not configured")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", path, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.sessionToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", path, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response for %s: %w", path, err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("response for %s exceeds the %d byte limit", path, maxResponseBytes)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request %s returned status %d: %s", path, response.StatusCode, backendErrorMessage(body))
	}
	return body, nil
}

func backendErrorMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "no response body"
	}
	return clipText(trimmed, 300)
}

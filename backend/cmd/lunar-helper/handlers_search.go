package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
	sharedHttp "lunar/backend/internal/shared/http"
	workspacedomain "lunar/backend/internal/workspace/domain"
)

var errInvalidSearchLimit = errors.New("limit must be a positive whole number")

const (
	searchPath         = "/search"
	defaultSearchLimit = 8
	maxSearchLimit     = 25
)

type searchResponse struct {
	Query   string                      `json:"query"`
	Results []codeindexdomain.CodeChunk `json:"results"`
}

func (s *helperServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if s.indexStore == nil {
		sharedHttp.WriteError(w, http.StatusServiceUnavailable, "the code index is not available")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		sharedHttp.WriteError(w, http.StatusBadRequest, "a search query is required")
		return
	}

	limit, err := parseSearchLimit(r.URL.Query().Get("limit"))
	if err != nil {
		sharedHttp.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	repositories := parseSearchRepositories(r.URL.Query().Get("repos"))

	results, searchError := s.indexStore.SearchChunksIn(r.Context(), query, repositories, limit)
	if searchError != nil {
		s.logger.Error("code search failed", "error", searchError)
		sharedHttp.WriteError(w, http.StatusInternalServerError, "code search failed")
		return
	}
	if results == nil {
		results = []codeindexdomain.CodeChunk{}
	}

	sharedHttp.WriteJSON(w, http.StatusOK, searchResponse{Query: query, Results: results})
}

func parseSearchLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultSearchLimit, nil
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, errInvalidSearchLimit
	}
	if parsed <= 0 {
		return 0, errInvalidSearchLimit
	}
	if parsed > maxSearchLimit {
		return maxSearchLimit, nil
	}
	return parsed, nil
}

func parseSearchRepositories(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	repositories := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, err := workspacedomain.ValidateRepoName(trimmed); err != nil {
			continue
		}
		repositories = append(repositories, trimmed)
	}

	if len(repositories) == 0 {
		return nil
	}
	return repositories
}

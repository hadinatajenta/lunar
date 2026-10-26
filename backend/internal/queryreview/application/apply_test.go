package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func sampleApplyRequest() domain.ApplyQueryReviewRequest {
	return domain.ApplyQueryReviewRequest{
		Repositories: []domain.ApplyRepository{
			{
				Repository: "  micro-payment  ",
				Queries: []domain.ApplyQuery{
					{No: 1, Feature: "  [micro-payment] Get user by PN  ", QueryStatement: "  explain select id from user_data where pn = ?  ", Index: "  pn  ", CreateIndex: "  CREATE INDEX idx_user_data_pn ON user_data (pn);  "},
				},
			},
		},
	}
}

func TestApplyQueryReviewTableReplacesTheTableAndReturnsTheWriteResult(t *testing.T) {
	provider := &stubDocumentProvider{
		storage:       `<p>Intro</p>`,
		updateVersion: 12,
		updateURL:     "https://confluence.example.com/display/DEV/Page",
	}
	service := NewQueryReviewService(provider, nil, nil, nil, nil)

	result, err := service.ApplyQueryReviewTable(context.Background(), "usr-1", " 123 ", sampleApplyRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PageID != "123" || result.Version != 12 {
		t.Fatalf("unexpected result identity %+v", result)
	}
	if result.URL != "https://confluence.example.com/display/DEV/Page" {
		t.Fatalf("unexpected result url %q", result.URL)
	}
	if provider.updatedPageID != "123" {
		t.Fatalf("expected the trimmed page id, got %q", provider.updatedPageID)
	}
	if !strings.HasPrefix(provider.updatedValue, `<p>Intro</p>`) {
		t.Fatalf("expected the existing storage to be preserved, got %q", provider.updatedValue)
	}
	if !strings.Contains(provider.updatedValue, captureExplainMarkerText) {
		t.Fatalf("expected the marker in the updated storage, got %q", provider.updatedValue)
	}
	if !strings.Contains(provider.updatedValue, `<tr><td>1</td><td>[micro-payment] Get user by PN</td><td>explain select id from user_data where pn = ?</td><td>pn<br/>CREATE INDEX idx_user_data_pn ON user_data (pn);</td><td></td></tr>`) {
		t.Fatalf("expected the trimmed query row, got %q", provider.updatedValue)
	}
}

func TestApplyQueryReviewTableValidatesTheRequest(t *testing.T) {
	provider := &stubDocumentProvider{storage: "<p>Intro</p>"}
	service := NewQueryReviewService(provider, nil, nil, nil, nil)
	ctx := context.Background()

	cases := map[string]domain.ApplyQueryReviewRequest{
		"no repositories":       {},
		"too many repositories": {Repositories: manyApplyRepositories(maxCaptureRepositories + 1)},
		"too many queries": {
			Repositories: []domain.ApplyRepository{{Repository: "micro-payment", Queries: manyApplyQueries(maxCaptureQueriesPerRepository + 1)}},
		},
		"oversized statement": {
			Repositories: []domain.ApplyRepository{{Repository: "micro-payment", Queries: []domain.ApplyQuery{{Feature: "x", QueryStatement: strings.Repeat("a", maxCaptureStatementLength+1)}}}},
		},
		"oversized create index": {
			Repositories: []domain.ApplyRepository{{Repository: "micro-payment", Queries: []domain.ApplyQuery{{Feature: "x", QueryStatement: "explain select 1", CreateIndex: strings.Repeat("a", maxCaptureCreateIndexLength+1)}}}},
		},
	}

	for name, request := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := service.ApplyQueryReviewTable(ctx, "usr-1", "123", request)
			if !errors.Is(err, sharedErrors.ErrBadRequest) {
				t.Fatalf("expected ErrBadRequest, got %v", err)
			}
		})
	}

	if provider.updatedValue != "" {
		t.Fatalf("expected no write for an invalid request, got %q", provider.updatedValue)
	}
}

func TestApplyQueryReviewTableWritesTheFallbackRowWhenNoQueriesExist(t *testing.T) {
	provider := &stubDocumentProvider{storage: "<p>Intro</p>"}
	service := NewQueryReviewService(provider, nil, nil, nil, nil)

	result, err := service.ApplyQueryReviewTable(context.Background(), "usr-1", "123", domain.ApplyQueryReviewRequest{
		Repositories: []domain.ApplyRepository{{Repository: "micro-payment"}},
	})
	if err != nil {
		t.Fatalf("expected the apply to succeed, got %v", err)
	}
	if result == nil {
		t.Fatal("expected an apply result")
	}
	if !strings.Contains(provider.updatedValue, noQueryChangesMessage) {
		t.Fatalf("expected the fallback message in the written storage, got %q", provider.updatedValue)
	}
}

func TestApplyQueryReviewTableRequiresAnAuthenticatedUser(t *testing.T) {
	service := NewQueryReviewService(&stubDocumentProvider{storage: "<p>x</p>"}, nil, nil, nil, nil)

	_, err := service.ApplyQueryReviewTable(context.Background(), "   ", "123", sampleApplyRequest())
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestApplyQueryReviewTableRequiresADocumentProvider(t *testing.T) {
	service := NewQueryReviewService(nil, nil, nil, nil, nil)

	_, err := service.ApplyQueryReviewTable(context.Background(), "usr-1", "123", sampleApplyRequest())
	if !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected ErrBadGateway, got %v", err)
	}
}

func TestApplyQueryReviewTablePropagatesReadAndWriteFailures(t *testing.T) {
	readFailure := NewQueryReviewService(&stubDocumentProvider{storageErr: sharedErrors.ErrNotFound}, nil, nil, nil, nil)
	if _, err := readFailure.ApplyQueryReviewTable(context.Background(), "usr-1", "123", sampleApplyRequest()); !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected the read failure to propagate, got %v", err)
	}

	writeFailure := NewQueryReviewService(&stubDocumentProvider{storage: "<p>x</p>", updateErr: errors.New("confluence is unreachable")}, nil, nil, nil, nil)
	_, err := writeFailure.ApplyQueryReviewTable(context.Background(), "usr-1", "123", sampleApplyRequest())
	if err == nil || errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a generic write failure, got %v", err)
	}
}

func manyApplyRepositories(count int) []domain.ApplyRepository {
	repositories := make([]domain.ApplyRepository, 0, count)
	for index := 0; index < count; index++ {
		repositories = append(repositories, domain.ApplyRepository{
			Repository: "repo",
			Queries:    []domain.ApplyQuery{{QueryStatement: "explain select 1"}},
		})
	}
	return repositories
}

func manyApplyQueries(count int) []domain.ApplyQuery {
	queries := make([]domain.ApplyQuery, 0, count)
	for index := 0; index < count; index++ {
		queries = append(queries, domain.ApplyQuery{QueryStatement: "explain select 1"})
	}
	return queries
}

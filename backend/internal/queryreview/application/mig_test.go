package application

import (
	"context"
	"errors"
	"testing"

	confluenceDomain "lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/queryreview/domain"
)

type stubDocumentProvider struct {
	document      *confluenceDomain.ConfluenceDocument
	documentErr   error
	ancestors     []confluenceDomain.ConfluencePageRef
	ancestorsErr  error
	children      []confluenceDomain.ConfluencePageRef
	childrenErr   error
	storage       string
	storageErr    error
	documentCalls int
	storagePageID string
	updateVersion int
	updateURL     string
	updateErr     error
	updatedPageID string
	updatedValue  string
}

func (s *stubDocumentProvider) GetDocumentDetail(ctx context.Context, userID string, documentID string) (*confluenceDomain.ConfluenceDocument, error) {
	s.documentCalls++
	if s.documentErr != nil {
		return nil, s.documentErr
	}
	return s.document, nil
}

func (s *stubDocumentProvider) GetPageAncestors(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	if s.ancestorsErr != nil {
		return nil, s.ancestorsErr
	}
	return s.ancestors, nil
}

func (s *stubDocumentProvider) GetChildPages(ctx context.Context, userID string, pageID string) ([]confluenceDomain.ConfluencePageRef, error) {
	if s.childrenErr != nil {
		return nil, s.childrenErr
	}
	return s.children, nil
}

func (s *stubDocumentProvider) GetPageStorage(ctx context.Context, userID string, pageID string) (string, error) {
	s.storagePageID = pageID
	if s.storageErr != nil {
		return "", s.storageErr
	}
	return s.storage, nil
}

func (s *stubDocumentProvider) UpdatePageStorage(ctx context.Context, userID string, pageID string, storage string) (int, string, error) {
	s.updatedPageID = pageID
	s.updatedValue = storage
	if s.updateErr != nil {
		return 0, "", s.updateErr
	}
	return s.updateVersion, s.updateURL, nil
}

const migStorageWithOneRepository = `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td></td><td></td><td>` + microCompareURL + `</td></tr>
</tbody></table>`

func newMigResolverService(provider *stubDocumentProvider) *QueryReviewService {
	return &QueryReviewService{documents: provider}
}

func newMigDocument(id string, title string) *confluenceDomain.ConfluenceDocument {
	return &confluenceDomain.ConfluenceDocument{ID: id, Title: title}
}

func TestResolveMigPicksTheImmediateParentAndTheMigPage(t *testing.T) {
	provider := &stubDocumentProvider{
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "1", Title: "Root"}, {ID: "2", Title: "Parent"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "9", Title: "Design"}, {ID: "10", Title: "CRMMS-1-MIG"}},
		storage:   migStorageWithOneRepository,
	}
	service := newMigResolverService(provider)

	resolution, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "CRMMS-1 Query Review"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution.pageID != "10" || resolution.pageTitle != "CRMMS-1-MIG" {
		t.Fatalf("expected the MIG page, got %+v", resolution)
	}
	if provider.storagePageID != "10" {
		t.Fatalf("expected the storage of the MIG page, got %q", provider.storagePageID)
	}
	if len(resolution.rows) != 1 || resolution.rows[0].repository != "micro-payment" {
		t.Fatalf("expected the parsed repository, got %+v", resolution.rows)
	}
}

func TestResolveMigRequiresTheIssueKeyInTheMigTitle(t *testing.T) {
	provider := &stubDocumentProvider{
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "2", Title: "Parent"}},
		children: []confluenceDomain.ConfluencePageRef{
			{ID: "10", Title: "CRMMS-1-MIG"},
			{ID: "11", Title: "CRMMS-76972-MIG"},
		},
		storage: migStorageWithOneRepository,
	}
	service := newMigResolverService(provider)

	resolution, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "CRMMS-76972 Query Review"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution.pageID != "11" {
		t.Fatalf("expected the page carrying the issue key, got %+v", resolution)
	}
}

func TestResolveMigReadsTheIssueKeyFromTheParentTitle(t *testing.T) {
	provider := &stubDocumentProvider{
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "2", Title: "CRMMS-76972 Parent"}},
		children: []confluenceDomain.ConfluencePageRef{
			{ID: "10", Title: "CRMMS-1-MIG"},
			{ID: "11", Title: "CRMMS-76972-MIG"},
		},
		storage: migStorageWithOneRepository,
	}
	service := newMigResolverService(provider)

	resolution, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "Query Review"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolution.pageID != "11" {
		t.Fatalf("expected the key matching page, got %+v", resolution)
	}
}

func TestResolveMigReportsNoMigPage(t *testing.T) {
	provider := &stubDocumentProvider{
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "2", Title: "Parent"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "9", Title: "Design"}},
	}
	service := newMigResolverService(provider)

	_, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "Query Review"))
	if !errors.Is(err, domain.ErrNoCompareLink) {
		t.Fatalf("expected the no compare link error, got %v", err)
	}
}

func TestResolveMigReportsNoParent(t *testing.T) {
	provider := &stubDocumentProvider{ancestors: nil}
	service := newMigResolverService(provider)

	_, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "Query Review"))
	if !errors.Is(err, domain.ErrNoCompareLink) {
		t.Fatalf("expected the no compare link error, got %v", err)
	}
}

func TestResolveMigReportsExcludedRepositories(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>fe-mms</td><td>feat/x</td><td>` + microCompareURL + `</td></tr>
<tr><td>micro-payment</td><td>feat/x</td><td>` + microCompareURL + `</td></tr>
</tbody></table>`
	provider := &stubDocumentProvider{
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "2", Title: "Parent"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "10", Title: "CRMMS-1-MIG"}},
		storage:   storage,
	}
	service := newMigResolverService(provider)

	resolution, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "CRMMS-1 Query Review"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resolution.rows) != 1 || resolution.rows[0].repository != "micro-payment" {
		t.Fatalf("expected only the kept repository, got %+v", resolution.rows)
	}
	if len(resolution.skipped) != 1 || resolution.skipped[0].Repository != "fe-mms" {
		t.Fatalf("expected the excluded repository to be reported, got %+v", resolution.skipped)
	}
}

func TestResolveMigRequiresAConfiguredProvider(t *testing.T) {
	service := &QueryReviewService{}

	_, err := service.resolveMig(context.Background(), "usr-1", newMigDocument("999", "Query Review"))
	if err == nil {
		t.Fatal("expected an error for a nil provider")
	}
}

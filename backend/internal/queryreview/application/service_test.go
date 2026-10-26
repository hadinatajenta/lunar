package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	authDomain "lunar/backend/internal/auth/domain"
	confluenceDomain "lunar/backend/internal/confluence/domain"
	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

const twoRepositoryStorage = `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>micro-payment</td><td>feat/CRMMS-1</td><td>` + microCompareURL + `</td></tr>
<tr><td>other-service</td><td>feat/CRMMS-1</td><td>` + otherCompareURL + `</td></tr>
</tbody></table>`

type stubPATAccess struct {
	secrets *authDomain.DecryptedSecrets
	err     error
}

func (s *stubPATAccess) GetDecryptedSecrets(ctx context.Context, userID string) (*authDomain.DecryptedSecrets, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.secrets, nil
}

type stubDiffSource struct {
	mu            sync.Mutex
	diff          string
	diffErr       error
	diffByRepo    map[string]string
	diffErrByRepo map[string]error
	branches      domain.PullRequestBranches
	branchesErr   error
	compareCalls  int
	branchesCalls int
	lastPAT       string
	lastProject   string
	lastRepo      string
	lastSource    string
	lastTarget    string
	lastPRID      string
}

func (s *stubDiffSource) GetCompareDiff(ctx context.Context, pat string, projectKey string, repoSlug string, sourceBranch string, targetBranch string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.compareCalls++
	s.lastPAT = pat
	s.lastProject = projectKey
	s.lastRepo = repoSlug
	s.lastSource = sourceBranch
	s.lastTarget = targetBranch
	if s.diffErrByRepo != nil {
		if err, exists := s.diffErrByRepo[repoSlug]; exists {
			return "", err
		}
	}
	if s.diffByRepo != nil {
		if diff, exists := s.diffByRepo[repoSlug]; exists {
			return diff, nil
		}
	}
	if s.diffErr != nil {
		return "", s.diffErr
	}
	return s.diff, nil
}

func (s *stubDiffSource) GetPullRequestBranches(ctx context.Context, pat string, projectKey string, repoSlug string, pullRequestID string) (domain.PullRequestBranches, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.branchesCalls++
	s.lastPAT = pat
	s.lastProject = projectKey
	s.lastRepo = repoSlug
	s.lastPRID = pullRequestID
	if s.branchesErr != nil {
		return domain.PullRequestBranches{}, s.branchesErr
	}
	return s.branches, nil
}

type stubExtractor struct {
	mu       sync.Mutex
	analysis domain.DiffAnalysis
	lastDiff string
}

func (s *stubExtractor) Extract(diffText string) domain.DiffAnalysis {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastDiff = diffText
	return s.analysis
}

type stubReviewer struct {
	mu         sync.Mutex
	extraction domain.QueryExtraction
	err        error
	calls      int
	lastUserID string
	lastModel  string
	lastFiles  []domain.PromptFile
}

func (s *stubReviewer) ReviewQueries(ctx context.Context, userID string, modelName string, files []domain.PromptFile) (domain.QueryExtraction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls++
	s.lastUserID = userID
	s.lastModel = modelName
	s.lastFiles = files
	if s.err != nil {
		return domain.QueryExtraction{}, s.err
	}
	return s.extraction, nil
}

func analysisWithOneFile() domain.DiffAnalysis {
	return domain.DiffAnalysis{
		Files: []domain.PromptFile{
			{
				Path:       "internal/repo/dao.go",
				Language:   "go",
				AddedLines: []domain.PromptLine{{Number: 11, Content: `db.Raw("select id from user_data")`}},
			},
		},
		Analyzed: []domain.AnalyzedFile{{Path: "internal/repo/dao.go", AddedLines: 1, Language: "go"}},
		Skipped:  []domain.SkippedFile{},
	}
}

func extractionWithOneQuery(feature string) domain.QueryExtraction {
	return domain.QueryExtraction{
		Queries:       []domain.QueryItem{{No: 1, Feature: feature, Table: "user_data", ActionType: "SELECT", QueryStatement: "select id from user_data", File: "internal/repo/dao.go", Line: 11}},
		Summary:       "One query was introduced.",
		PromptVersion: domain.PromptVersion,
	}
}

func migDocumentProvider(storage string) *stubDocumentProvider {
	return &stubDocumentProvider{
		document:  &confluenceDomain.ConfluenceDocument{ID: "123", Title: "CRMMS-1 Query Review"},
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "100", Title: "Parent"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "CRMMS-1-MIG"}},
		storage:   storage,
	}
}

func explicitSource() *domain.CompareSource {
	return &domain.CompareSource{
		CompareURL:   "https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fx&targetBranch=development",
		ProjectKey:   "NME",
		RepoSlug:     "micro-payment",
		SourceBranch: "feat/x",
		TargetBranch: "development",
	}
}

func decryptedPAT() *stubPATAccess {
	return &stubPATAccess{secrets: &authDomain.DecryptedSecrets{BitbucketPAT: "pat-1"}}
}

func TestGenerateQueryReviewUsesTheExplicitSourceAsASingleRepository(t *testing.T) {
	documents := &stubDocumentProvider{documentErr: errors.New("the document must not be fetched")}
	diffSource := &stubDiffSource{diff: "raw diff"}
	extractor := &stubExtractor{analysis: analysisWithOneFile()}
	reviewer := &stubReviewer{extraction: extractionWithOneQuery("Get user by PN")}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, reviewer)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{
		Model:  "model-a",
		Source: explicitSource(),
	})
	if err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}

	if documents.documentCalls != 0 {
		t.Fatalf("expected the document not to be fetched, got %d calls", documents.documentCalls)
	}
	if result.DocumentID != "123" || result.Model != "model-a" || result.PromptVersion != domain.PromptVersion {
		t.Fatalf("unexpected result identity %+v", result)
	}
	if result.MigPageID != "" || result.MigPageTitle != "" {
		t.Fatalf("expected no MIG page for an explicit source, got %q/%q", result.MigPageID, result.MigPageTitle)
	}
	if result.SkippedRepositories == nil || len(result.SkippedRepositories) != 0 {
		t.Fatalf("expected an empty skipped repositories array, got %+v", result.SkippedRepositories)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("expected exactly one repository, got %+v", result.Repositories)
	}

	review := result.Repositories[0]
	if review.Repository != "micro-payment" || review.ProjectKey != "NME" {
		t.Fatalf("unexpected repository identity %+v", review)
	}
	if review.SourceBranch != "feat/x" || review.TargetBranch != "development" {
		t.Fatalf("unexpected repository branches %+v", review)
	}
	if review.Status != domain.RepositoryStatusSuccess {
		t.Fatalf("expected a success status, got %q (error %q)", review.Status, review.Error)
	}
	if len(review.Queries) != 1 || review.Queries[0].Repository != "micro-payment" {
		t.Fatalf("expected the query to carry its repository, got %+v", review.Queries)
	}
	if review.Queries[0].Feature != "[micro-payment] Get user by PN" {
		t.Fatalf("expected the feature prefix, got %q", review.Queries[0].Feature)
	}
	if diffSource.lastPAT != "pat-1" || diffSource.lastProject != "NME" || diffSource.lastRepo != "micro-payment" {
		t.Fatalf("unexpected diff fetch %+v", diffSource)
	}
	if extractor.lastDiff != "raw diff" {
		t.Fatalf("expected the fetched diff to be extracted, got %q", extractor.lastDiff)
	}
}

func TestGenerateQueryReviewProcessesEveryMigRepository(t *testing.T) {
	documents := migDocumentProvider(twoRepositoryStorage)
	diffSource := &stubDiffSource{diffByRepo: map[string]string{
		"micro-payment": "diff micro",
		"other-service": "diff other",
	}}
	extractor := &stubExtractor{analysis: analysisWithOneFile()}
	reviewer := &stubReviewer{extraction: extractionWithOneQuery("Get user by PN")}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, reviewer)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}

	if result.MigPageID != "200" || result.MigPageTitle != "CRMMS-1-MIG" {
		t.Fatalf("expected the MIG page identity, got %q/%q", result.MigPageID, result.MigPageTitle)
	}
	if len(result.Repositories) != 2 {
		t.Fatalf("expected two repositories, got %+v", result.Repositories)
	}
	if result.Repositories[0].Repository != "micro-payment" || result.Repositories[1].Repository != "other-service" {
		t.Fatalf("expected the MIG table order, got %+v", result.Repositories)
	}
	for _, review := range result.Repositories {
		if review.Status != domain.RepositoryStatusSuccess {
			t.Fatalf("expected a success status for %s, got %q", review.Repository, review.Status)
		}
		if len(review.Queries) != 1 || review.Queries[0].Repository != review.Repository {
			t.Fatalf("expected the query to carry %s, got %+v", review.Repository, review.Queries)
		}
		if !strings.HasPrefix(review.Queries[0].Feature, "["+review.Repository+"] ") {
			t.Fatalf("expected the feature prefix for %s, got %q", review.Repository, review.Queries[0].Feature)
		}
	}
	if diffSource.compareCalls != 2 {
		t.Fatalf("expected one diff fetch per repository, got %d", diffSource.compareCalls)
	}
}

func TestGenerateQueryReviewReportsARepositoryFailureWithoutFailingOthers(t *testing.T) {
	documents := migDocumentProvider(twoRepositoryStorage)
	diffSource := &stubDiffSource{
		diffByRepo:    map[string]string{"micro-payment": "raw diff"},
		diffErrByRepo: map[string]error{"other-service": errors.New("bitbucket is unreachable")},
	}
	extractor := &stubExtractor{analysis: analysisWithOneFile()}
	reviewer := &stubReviewer{extraction: extractionWithOneQuery("Get user by PN")}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, reviewer)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if err != nil {
		t.Fatalf("expected the other repositories to survive, got %v", err)
	}

	if len(result.Repositories) != 2 {
		t.Fatalf("expected two repositories, got %+v", result.Repositories)
	}
	if result.Repositories[0].Status != domain.RepositoryStatusSuccess {
		t.Fatalf("expected the healthy repository to succeed, got %q", result.Repositories[0].Status)
	}
	failed := result.Repositories[1]
	if failed.Status != domain.RepositoryStatusError {
		t.Fatalf("expected an error status, got %q", failed.Status)
	}
	if !strings.Contains(failed.Error, "bitbucket is unreachable") {
		t.Fatalf("expected the failure message, got %q", failed.Error)
	}
	if failed.Queries == nil || len(failed.Queries) != 0 {
		t.Fatalf("expected empty queries on failure, got %+v", failed.Queries)
	}
}

func TestGenerateQueryReviewReportsExcludedRepositories(t *testing.T) {
	storage := `<table><tbody>
<tr><th>Repository</th><th>Branch</th><th>Compare</th></tr>
<tr><td>fe-mms</td><td>feat/x</td><td>` + microCompareURL + `</td></tr>
<tr><td>micro-payment</td><td>feat/x</td><td>` + microCompareURL + `</td></tr>
</tbody></table>`
	documents := migDocumentProvider(storage)
	diffSource := &stubDiffSource{diff: "raw diff"}
	extractor := &stubExtractor{analysis: analysisWithOneFile()}
	reviewer := &stubReviewer{extraction: extractionWithOneQuery("Get user by PN")}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, reviewer)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}

	if len(result.Repositories) != 1 || result.Repositories[0].Repository != "micro-payment" {
		t.Fatalf("expected only the kept repository, got %+v", result.Repositories)
	}
	if len(result.SkippedRepositories) != 1 || result.SkippedRepositories[0].Repository != "fe-mms" {
		t.Fatalf("expected the excluded repository to be reported, got %+v", result.SkippedRepositories)
	}
	if diffSource.compareCalls != 1 {
		t.Fatalf("expected the excluded repository never to be fetched, got %d calls", diffSource.compareCalls)
	}
}

func TestGenerateQueryReviewMarksNoQueriesWhenNoAddedLinesAreRelevant(t *testing.T) {
	documents := migDocumentProvider(migStorageWithOneRepository)
	diffSource := &stubDiffSource{diff: "raw diff"}
	extractor := &stubExtractor{analysis: domain.DiffAnalysis{
		Files:    []domain.PromptFile{},
		Analyzed: []domain.AnalyzedFile{},
		Skipped:  []domain.SkippedFile{{Path: "internal/repo/dao_test.go", Reason: "test file"}},
	}}
	reviewer := &stubReviewer{err: errors.New("the model must not be called")}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, reviewer)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}

	if reviewer.calls != 0 {
		t.Fatalf("expected no model call, got %d", reviewer.calls)
	}
	review := result.Repositories[0]
	if review.Status != domain.RepositoryStatusNoQueries {
		t.Fatalf("expected a no_queries status, got %q", review.Status)
	}
	if review.Queries == nil || len(review.Queries) != 0 {
		t.Fatalf("expected an empty non nil queries array, got %+v", review.Queries)
	}
	if review.AnalyzedFiles == nil || review.SkippedFiles == nil {
		t.Fatalf("expected non nil file arrays, got %+v", review)
	}
	if review.Summary != emptyAnalysisSummary {
		t.Fatalf("expected the empty analysis summary, got %q", review.Summary)
	}
	if len(review.SkippedFiles) != 1 {
		t.Fatalf("expected the skipped file to survive, got %+v", review.SkippedFiles)
	}
}

func TestGenerateQueryReviewRejectsADocumentWithoutAMigPage(t *testing.T) {
	documents := &stubDocumentProvider{
		document:  &confluenceDomain.ConfluenceDocument{ID: "123", Title: "Query Review"},
		ancestors: []confluenceDomain.ConfluencePageRef{{ID: "100", Title: "Parent"}},
		children:  []confluenceDomain.ConfluencePageRef{{ID: "200", Title: "Design"}},
	}
	diffSource := &stubDiffSource{}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, &stubExtractor{}, &stubReviewer{})

	_, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if !errors.Is(err, domain.ErrNoCompareLink) {
		t.Fatalf("expected the no compare link error, got %v", err)
	}
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected the error to be a bad request, got %v", err)
	}
	if diffSource.compareCalls != 0 {
		t.Fatalf("expected no diff fetch, got %d", diffSource.compareCalls)
	}
}

func TestGenerateQueryReviewUsesTheDefaultModelWhenNoneIsRequested(t *testing.T) {
	documents := migDocumentProvider(migStorageWithOneRepository)
	diffSource := &stubDiffSource{diff: "raw diff"}
	extractor := &stubExtractor{analysis: analysisWithOneFile()}
	reviewer := &stubReviewer{extraction: extractionWithOneQuery("Get user by PN")}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, reviewer)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}

	if reviewer.lastModel != DefaultModel || result.Model != DefaultModel {
		t.Fatalf("expected the default model %q, got %q/%q", DefaultModel, reviewer.lastModel, result.Model)
	}
}

func TestGenerateQueryReviewResolvesAPullRequestExplicitSource(t *testing.T) {
	service := NewQueryReviewService(
		nil,
		decryptedPAT(),
		&stubDiffSource{
			diff:     "raw diff",
			branches: domain.PullRequestBranches{SourceBranch: "feat/pr", TargetBranch: "development"},
		},
		&stubExtractor{analysis: analysisWithOneFile()},
		&stubReviewer{extraction: extractionWithOneQuery("Get user by PN")},
	)

	result, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{
		Source: &domain.CompareSource{CompareURL: "https://bitbucket.example.com/projects/NME/repos/micro-payment/pull-requests/421/overview"},
	})
	if err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}

	review := result.Repositories[0]
	if review.SourceBranch != "feat/pr" || review.TargetBranch != "development" {
		t.Fatalf("expected the pull request branches, got %+v", review)
	}
	expectedCompareURL := "https://bitbucket.example.com/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat%2Fpr&targetBranch=development"
	if review.CompareURL != expectedCompareURL {
		t.Fatalf("expected the synthesized compare URL %q, got %q", expectedCompareURL, review.CompareURL)
	}
}

func TestGenerateQueryReviewRequiresAConfiguredBitbucketPAT(t *testing.T) {
	service := NewQueryReviewService(
		&stubDocumentProvider{},
		&stubPATAccess{secrets: &authDomain.DecryptedSecrets{}},
		&stubDiffSource{},
		&stubExtractor{},
		&stubReviewer{},
	)

	_, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{Source: explicitSource()})
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestGenerateQueryReviewValidatesTheExplicitSource(t *testing.T) {
	service := NewQueryReviewService(nil, decryptedPAT(), &stubDiffSource{}, &stubExtractor{}, &stubReviewer{})

	_, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{
		Source: &domain.CompareSource{ProjectKey: "NME", RepoSlug: "micro-payment", TargetBranch: "development"},
	})
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error for the missing source branch, got %v", err)
	}
}

func TestGenerateQueryReviewRequiresAnAuthenticatedUser(t *testing.T) {
	service := NewQueryReviewService(nil, decryptedPAT(), &stubDiffSource{}, &stubExtractor{}, &stubReviewer{})

	_, err := service.GenerateQueryReview(context.Background(), "  ", "123", domain.QueryReviewRequest{Source: explicitSource()})
	if !errors.Is(err, sharedErrors.ErrUnauthorized) {
		t.Fatalf("expected an unauthorized error, got %v", err)
	}
}

func TestGenerateQueryReviewReportsAMissingDocument(t *testing.T) {
	service := NewQueryReviewService(&stubDocumentProvider{}, decryptedPAT(), &stubDiffSource{}, &stubExtractor{}, &stubReviewer{})

	_, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if !errors.Is(err, sharedErrors.ErrNotFound) {
		t.Fatalf("expected a not found error, got %v", err)
	}
}

func TestGenerateQueryReviewRequiresAConfiguredDocumentProvider(t *testing.T) {
	service := NewQueryReviewService(nil, decryptedPAT(), &stubDiffSource{}, &stubExtractor{}, &stubReviewer{})

	_, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{})
	if !errors.Is(err, sharedErrors.ErrBadGateway) {
		t.Fatalf("expected a bad gateway error, got %v", err)
	}
}

func TestGenerateQueryReviewReturnsTheFetchedDiffToTheExtractor(t *testing.T) {
	documents := migDocumentProvider(migStorageWithOneRepository)
	diffSource := &stubDiffSource{diff: "diff --git a/x b/x\n"}
	extractor := &stubExtractor{analysis: domain.DiffAnalysis{
		Files:    []domain.PromptFile{},
		Analyzed: []domain.AnalyzedFile{},
		Skipped:  []domain.SkippedFile{},
	}}
	service := NewQueryReviewService(documents, decryptedPAT(), diffSource, extractor, &stubReviewer{})

	if _, err := service.GenerateQueryReview(context.Background(), "usr-1", "123", domain.QueryReviewRequest{}); err != nil {
		t.Fatalf("cannot generate the query review: %v", err)
	}
	if extractor.lastDiff != diffSource.diff {
		t.Fatalf("expected the fetched diff, got %q", extractor.lastDiff)
	}
}

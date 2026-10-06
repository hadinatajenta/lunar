package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/workspace/domain"
)

func categorisedBranchScanner() *stubBranchScanner {
	return &stubBranchScanner{
		branchesByRepository: map[string][]domain.Branch{
			"alpha": {
				{Name: "feature/CRMMS-6397-Enhancement-Manage-User", IsCurrent: true},
				{Name: "feature/CRMMS-77745-Enhancement-Other"},
				{Name: "feature/CRMMS-77740-Manage-User"},
			},
			"beta": {
				{Name: "feature/CRMMS-77741-Manage-User"},
				{Name: "main"},
			},
		},
	}
}

func TestCategoryCountsAtRootCountsTheSegmentAfterTheKey(t *testing.T) {
	service := NewIssueBranchService(categorisedBranchScanner())

	counts, err := service.CategoryCountsAtRoot(context.Background(), "/workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counts) != 2 || counts["Manage"] != 2 || counts["Enhancement"] != 2 {
		t.Fatalf("counts = %v, want Manage twice and Enhancement twice", counts)
	}
}

func TestCategoryCountsAtRootIgnoresKeysThatDoNotLeadTheFinalSegment(t *testing.T) {
	scanner := &stubBranchScanner{branchesByRepository: map[string][]domain.Branch{
		"alpha": {
			{Name: "feature/generate-tid-CRMMS-6397"},
			{Name: "feature/enhancement-input-tid-max-30-CRMMS-12290"},
			{Name: "feature/CRMMS-6397-"},
		},
	}}
	service := NewIssueBranchService(scanner)

	counts, err := service.CategoryCountsAtRoot(context.Background(), "/workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(counts) != 0 {
		t.Fatalf("counts = %v, want none, because no branch carries a category after its leading key", counts)
	}
}

func TestCategoryCountsAtRootSharesTheCachedBranchScan(t *testing.T) {
	scanner := categorisedBranchScanner()
	service := NewIssueBranchService(scanner)

	if _, err := service.CategoryCountsAtRoot(context.Background(), "/workspace"); err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if _, err := service.CategoryCountsAtRoot(context.Background(), "/workspace"); err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	if scanner.calls != 1 {
		t.Fatalf("the category counts must reuse the cached branch scan, got %d scans", scanner.calls)
	}
}

func TestCategoryCountsAtRootPropagatesTheScanFailure(t *testing.T) {
	scanner := categorisedBranchScanner()
	scanner.err = errors.New("git is unavailable")
	service := NewIssueBranchService(scanner)

	_, err := service.CategoryCountsAtRoot(context.Background(), "/workspace")
	if err == nil || !strings.Contains(err.Error(), "git is unavailable") {
		t.Fatalf("the scan failure must be wrapped with its cause, got %v", err)
	}
}

func TestCategoryCountsAtRootRequiresAScanner(t *testing.T) {
	service := NewIssueBranchService(nil)

	if _, err := service.CategoryCountsAtRoot(context.Background(), "/workspace"); !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader, got %v", err)
	}
}

func TestIssueCategoryVocabularyResolvesTheRootAndCounts(t *testing.T) {
	vocabulary := NewIssueCategoryVocabulary(stubSuggestionRoots{rootPath: "/workspace"}, NewIssueBranchService(categorisedBranchScanner()))

	counts, err := vocabulary.ObserveCategories(context.Background(), "user-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if counts["Manage"] != 2 || counts["Enhancement"] != 2 {
		t.Fatalf("counts = %v, want Manage twice and Enhancement twice", counts)
	}
}

func TestIssueCategoryVocabularyReportsARootFailure(t *testing.T) {
	vocabulary := NewIssueCategoryVocabulary(stubSuggestionRoots{err: errors.New("no usable workspace is configured")}, NewIssueBranchService(categorisedBranchScanner()))

	_, err := vocabulary.ObserveCategories(context.Background(), "user-a")
	if err == nil || !strings.Contains(err.Error(), "no usable workspace is configured") {
		t.Fatalf("the root failure must be wrapped with its cause, got %v", err)
	}
}

func TestIssueCategoryVocabularyRequiresItsCollaborators(t *testing.T) {
	vocabulary := NewIssueCategoryVocabulary(stubSuggestionRoots{rootPath: "/workspace"}, nil)

	if _, err := vocabulary.ObserveCategories(context.Background(), "user-a"); !errors.Is(err, ErrServerReaderUnavailable) {
		t.Fatalf("expected an unavailable reader, got %v", err)
	}
}

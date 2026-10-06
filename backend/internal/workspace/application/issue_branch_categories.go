package application

import (
	"context"
	"fmt"
	"strings"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const branchCategorySeparator = "-"

type WorkspaceRootResolver interface {
	WorkspaceRootFor(ctx context.Context, userID string) (string, error)
}

type IssueCategoryVocabulary struct {
	roots    WorkspaceRootResolver
	branches *IssueBranchService
}

func NewIssueCategoryVocabulary(roots WorkspaceRootResolver, branches *IssueBranchService) *IssueCategoryVocabulary {
	return &IssueCategoryVocabulary{roots: roots, branches: branches}
}

func (v *IssueCategoryVocabulary) ObserveCategories(ctx context.Context, userID string) (map[string]int, error) {
	if v.roots == nil || v.branches == nil {
		return nil, fmt.Errorf("category vocabulary: %w", ErrServerReaderUnavailable)
	}

	rootPath, err := v.roots.WorkspaceRootFor(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve the workspace root for the branch categories: %w", err)
	}

	return v.branches.CategoryCountsAtRoot(ctx, rootPath)
}

func (s *IssueBranchService) CategoryCountsAtRoot(ctx context.Context, rootPath string) (map[string]int, error) {
	index, err := s.branchIndex(ctx, rootPath)
	if err != nil {
		return nil, fmt.Errorf("count the branch categories: %w", err)
	}

	return categoryCounts(index.allBranches), nil
}

func categoryCounts(branches []BranchMatch) map[string]int {
	counts := make(map[string]int)
	for _, branch := range branches {
		category, found := branchCategory(branch.Branch)
		if !found {
			continue
		}

		counts[category]++
	}

	return counts
}

func branchCategory(branchName string) (string, bool) {
	segment := branchName[strings.LastIndex(branchName, "/")+1:]
	keys := jiraDomain.IssueKeysIn(segment)
	if len(keys) == 0 || !strings.HasPrefix(strings.ToUpper(segment), keys[0]) {
		return "", false
	}

	remainder := strings.TrimPrefix(segment[len(keys[0]):], branchCategorySeparator)
	category, _, _ := strings.Cut(remainder, branchCategorySeparator)
	category = strings.TrimSpace(category)
	if category == "" {
		return "", false
	}

	return category, true
}

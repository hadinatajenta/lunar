package application

import (
	"context"
	"fmt"
	"time"

	jiraDomain "lunar/backend/internal/jira/domain"
)

const branchNameIssueSummary = "Sebagai Administrator, I want to melihat daftar user yang dikelompokkan berdasarkan Personal Number (PN), mengelola level/jabatan mereka dalam satu halaman detail, serta melihat riwayat perubahannya"

const branchNameModelAnswer = `{"category":"Enhancement","category_choice":"new","core":"Manage-User","evidence":"Halaman Utama Manage User (Grouping by PN)"}`

const branchNameFallbackCore = "Melihat-Daftar-User-Dikelompokkan-Berdasarkan"

type stubBranchNameIssues struct {
	issue    *jiraDomain.JiraIssue
	err      error
	requests []string
}

func (s *stubBranchNameIssues) GetIssueDetail(_ context.Context, userID string, issueKey string) (*jiraDomain.JiraIssue, error) {
	s.requests = append(s.requests, userID+"|"+issueKey)
	if s.err != nil {
		return nil, s.err
	}

	return s.issue, nil
}

type stubBranchNameModel struct {
	answers     []string
	err         error
	calls       int
	prompts     []string
	hasDeadline bool
	budget      time.Duration
}

func (s *stubBranchNameModel) CompleteJSON(ctx context.Context, _ string, _ string, prompt string) (string, error) {
	s.calls++
	s.prompts = append(s.prompts, prompt)
	if deadline, ok := ctx.Deadline(); ok {
		s.hasDeadline = true
		s.budget = time.Until(deadline)
	}
	if s.err != nil {
		return "", s.err
	}
	if len(s.answers) == 0 {
		return "", fmt.Errorf("the stub model holds no answer")
	}
	if s.calls <= len(s.answers) {
		return s.answers[s.calls-1], nil
	}

	return s.answers[len(s.answers)-1], nil
}

type stubCategoryVocabulary struct {
	counts map[string]int
	err    error
	calls  int
}

func (s *stubCategoryVocabulary) ObserveCategories(_ context.Context, _ string) (map[string]int, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}

	return s.counts, nil
}

func branchNameIssue() *jiraDomain.JiraIssue {
	return &jiraDomain.JiraIssue{
		Key: "CRMMS-77911",
		Fields: jiraDomain.JiraIssueFields{
			Summary:     branchNameIssueSummary,
			Description: acceptanceDescription,
		},
	}
}

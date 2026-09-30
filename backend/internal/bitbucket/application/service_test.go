package application

import (
	"context"
	"errors"
	"testing"

	"lunar/backend/internal/bitbucket/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type mockBitbucketRepo struct {
	pushes   []domain.PushBranch
	prs      []domain.PullRequest
	diff     *domain.PRDiff
	review   *domain.AICodeReview
	comment  *domain.PRComment
	lastAct  string
	createPr *domain.PullRequest
}

func (m *mockBitbucketRepo) ListPushes(ctx context.Context, pat string, filter string) ([]domain.PushBranch, error) {
	return m.pushes, nil
}

func (m *mockBitbucketRepo) ListPullRequests(ctx context.Context, pat string, filter string) ([]domain.PullRequest, error) {
	return m.prs, nil
}

func (m *mockBitbucketRepo) GetPullRequestDiff(ctx context.Context, pat string, repo string, prID string) (*domain.PRDiff, error) {
	return m.diff, nil
}

func (m *mockBitbucketRepo) CreatePullRequest(ctx context.Context, pat string, req domain.CreatePRRequest, author string) (*domain.PullRequest, error) {
	if m.createPr != nil {
		return m.createPr, nil
	}
	return &domain.PullRequest{
		ID:           "#999",
		Repo:         req.Repo,
		Title:        req.Title,
		SourceBranch: req.SourceBranch,
		TargetBranch: req.TargetBranch,
		Status:       "open",
		Author:       author,
	}, nil
}

func (m *mockBitbucketRepo) PostPRComment(ctx context.Context, pat string, repo string, prID string, comment string, author string) (*domain.PRComment, error) {
	return &domain.PRComment{
		ID:      "comm-1",
		PRID:    prID,
		Author:  author,
		Content: comment,
	}, nil
}

func (m *mockBitbucketRepo) ApplyReviewAction(ctx context.Context, pat string, repo string, prID string, action string) error {
	m.lastAct = action
	return nil
}

func (m *mockBitbucketRepo) GenerateAIReview(ctx context.Context, prID string, repo string) (*domain.AICodeReview, error) {
	return &domain.AICodeReview{
		PRID:             prID,
		Summary:          "Good job",
		Findings:         []string{"Finding 1"},
		GeneratedComment: "Copilot comment",
	}, nil
}

func (m *mockBitbucketRepo) ResetSeedData() {}

func TestBitbucketService(t *testing.T) {
	mockRepo := &mockBitbucketRepo{
		pushes: []domain.PushBranch{
			{ID: "p1", Branch: "feat/test", Status: "ready"},
		},
		prs: []domain.PullRequest{
			{ID: "#1", Title: "PR 1", Status: "open"},
		},
		diff: &domain.PRDiff{PRID: "#1", Repo: "lunar/api-gateway"},
	}

	service := NewBitbucketService(mockRepo, nil, nil)
	ctx := context.Background()

	t.Run("ListPushes", func(t *testing.T) {
		pushes, err := service.ListPushes(ctx, "usr-1", "all")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pushes) != 1 || pushes[0].ID != "p1" {
			t.Errorf("unexpected pushes: %v", pushes)
		}
	})

	t.Run("ListPullRequests", func(t *testing.T) {
		prs, err := service.ListPullRequests(ctx, "usr-1", "all")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(prs) != 1 || prs[0].ID != "#1" {
			t.Errorf("unexpected prs: %v", prs)
		}
	})

	t.Run("GetPullRequestDiff", func(t *testing.T) {
		diff, err := service.GetPullRequestDiff(ctx, "usr-1", "lunar/api-gateway", "#1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if diff.PRID != "#1" {
			t.Errorf("expected #1, got %s", diff.PRID)
		}

		_, err = service.GetPullRequestDiff(ctx, "usr-1", "lunar/api-gateway", "")
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest, got %v", err)
		}
	})

	t.Run("CreatePullRequest Validation", func(t *testing.T) {
		_, err := service.CreatePullRequest(ctx, "usr-1", domain.CreatePRRequest{
			Repo:         "lunar/api-gateway",
			SourceBranch: "feat/foo",
			TargetBranch: "main",
			Title:        "",
		})
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest for empty title, got %v", err)
		}

		created, err := service.CreatePullRequest(ctx, "usr-1", domain.CreatePRRequest{
			Repo:         "lunar/api-gateway",
			SourceBranch: "feat/foo",
			TargetBranch: "main",
			Title:        "Feature Foo",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if created.Title != "Feature Foo" {
			t.Errorf("unexpected title: %s", created.Title)
		}
	})

	t.Run("PostPRComment", func(t *testing.T) {
		_, err := service.PostPRComment(ctx, "usr-1", "lunar/api-gateway", "#1", "")
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest for empty comment, got %v", err)
		}

		comm, err := service.PostPRComment(ctx, "usr-1", "lunar/api-gateway", "#1", "LGTM")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if comm.Content != "LGTM" {
			t.Errorf("unexpected content: %s", comm.Content)
		}
	})

	t.Run("ApplyReviewAction", func(t *testing.T) {
		err := service.ApplyReviewAction(ctx, "usr-1", "lunar/api-gateway", "#1", "invalid")
		if !errors.Is(err, sharedErrors.ErrBadRequest) {
			t.Errorf("expected ErrBadRequest for invalid action, got %v", err)
		}

		err = service.ApplyReviewAction(ctx, "usr-1", "lunar/api-gateway", "#1", "approve")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mockRepo.lastAct != "approve" {
			t.Errorf("expected approve, got %s", mockRepo.lastAct)
		}
	})

	t.Run("GenerateAIReview", func(t *testing.T) {
		review, err := service.GenerateAIReview(ctx, "usr-1", "lunar/api-gateway", "#1", "DeepSeek-V4 Pro (Thinking)")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if review.Summary != "Good job" {
			t.Errorf("unexpected summary: %s", review.Summary)
		}
	})
}

package application

import (
	"context"
	"fmt"
	"strings"

	authApp "lunar/backend/internal/auth/application"
	"lunar/backend/internal/bitbucket/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

type CopilotReviewer interface {
	ReviewPullRequest(ctx context.Context, userID string, repo string, prID string, diffText string, modelName string) (*domain.AICodeReview, error)
}

type BitbucketService struct {
	repo            domain.BitbucketRepository
	authService     *authApp.AuthService
	copilotReviewer CopilotReviewer
}

func NewBitbucketService(
	repo domain.BitbucketRepository,
	authService *authApp.AuthService,
	copilotReviewer CopilotReviewer,
) *BitbucketService {
	return &BitbucketService{
		repo:            repo,
		authService:     authService,
		copilotReviewer: copilotReviewer,
	}
}

func (s *BitbucketService) resolveUserPAT(ctx context.Context, userID string) (string, string) {
	if s.authService == nil || userID == "" {
		return "", "Current User"
	}
	decrypted, err := s.authService.GetDecryptedSecrets(ctx, userID)
	if err != nil || decrypted == nil {
		return "", "Current User"
	}
	author := decrypted.BitbucketUsername
	if author == "" {
		author = "Current User"
	}
	return decrypted.BitbucketPAT, author
}

func (s *BitbucketService) ListPushes(ctx context.Context, userID string, filter string) ([]domain.PushBranch, error) {
	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.ListPushes(ctx, pat, filter)
}

func (s *BitbucketService) ListPullRequests(ctx context.Context, userID string, filter string) ([]domain.PullRequest, error) {
	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.ListPullRequests(ctx, pat, filter)
}

func (s *BitbucketService) GetPullRequestDiff(ctx context.Context, userID string, repo string, prID string) (*domain.PRDiff, error) {
	if strings.TrimSpace(prID) == "" {
		return nil, fmt.Errorf("%w: pull request id is required", sharedErrors.ErrBadRequest)
	}
	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.GetPullRequestDiff(ctx, pat, repo, prID)
}

func (s *BitbucketService) CreatePullRequest(ctx context.Context, userID string, req domain.CreatePRRequest) (*domain.PullRequest, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("%w: title is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(req.SourceBranch) == "" {
		return nil, fmt.Errorf("%w: source branch is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(req.TargetBranch) == "" {
		return nil, fmt.Errorf("%w: target branch is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(req.Repo) == "" {
		return nil, fmt.Errorf("%w: repo is required", sharedErrors.ErrBadRequest)
	}

	pat, author := s.resolveUserPAT(ctx, userID)
	return s.repo.CreatePullRequest(ctx, pat, req, author)
}

func (s *BitbucketService) PostPRComment(ctx context.Context, userID string, repo string, prID string, comment string) (*domain.PRComment, error) {
	if strings.TrimSpace(prID) == "" {
		return nil, fmt.Errorf("%w: pull request id is required", sharedErrors.ErrBadRequest)
	}
	if strings.TrimSpace(comment) == "" {
		return nil, fmt.Errorf("%w: comment content cannot be empty", sharedErrors.ErrBadRequest)
	}

	pat, author := s.resolveUserPAT(ctx, userID)
	return s.repo.PostPRComment(ctx, pat, repo, prID, comment, author)
}

func (s *BitbucketService) ApplyReviewAction(ctx context.Context, userID string, repo string, prID string, action string) error {
	normalized := strings.ToLower(strings.TrimSpace(action))
	if normalized != "needs-work" && normalized != "approve" && normalized != "decline" {
		return fmt.Errorf("%w: invalid review action", sharedErrors.ErrBadRequest)
	}

	pat, _ := s.resolveUserPAT(ctx, userID)
	return s.repo.ApplyReviewAction(ctx, pat, repo, prID, normalized)
}

func (s *BitbucketService) GenerateAIReview(ctx context.Context, userID string, repo string, prID string, model string) (*domain.AICodeReview, error) {
	if strings.TrimSpace(prID) == "" {
		return nil, fmt.Errorf("%w: pull request id is required", sharedErrors.ErrBadRequest)
	}

	if s.copilotReviewer != nil {
		pat, _ := s.resolveUserPAT(ctx, userID)
		diff, err := s.repo.GetPullRequestDiff(ctx, pat, repo, prID)
		if err == nil && diff != nil {
			var sb strings.Builder
			for _, file := range diff.Files {
				sb.WriteString(fmt.Sprintf("File: %s (+%d -%d)\n", file.NewPath, file.Additions, file.Deletions))
				for _, hunk := range file.Hunks {
					sb.WriteString(hunk.Header + "\n")
					for _, line := range hunk.Lines {
						sb.WriteString(line + "\n")
					}
				}
			}
			diffText := sb.String()
			if strings.TrimSpace(diffText) == "" {
				diffText = fmt.Sprintf("PR %s in %s with %d files", prID, repo, diff.TotalAdded)
			}
			return s.copilotReviewer.ReviewPullRequest(ctx, userID, repo, prID, diffText, model)
		}
	}

	return s.repo.GenerateAIReview(ctx, prID, repo)
}

func (s *BitbucketService) ResetSeedData() {
	s.repo.ResetSeedData()
}

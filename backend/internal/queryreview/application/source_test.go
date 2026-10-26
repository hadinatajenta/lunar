package application

import (
	"errors"
	"testing"

	"lunar/backend/internal/queryreview/domain"
	sharedErrors "lunar/backend/internal/shared/errors"
)

func TestExplicitSelectedSourceParsesACompareURLOnly(t *testing.T) {
	source, err := explicitSelectedSource(domain.CompareSource{
		CompareURL: "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat/CRMMS-1&targetBranch=development",
	})
	if err != nil {
		t.Fatalf("expected the compare URL to be parsed, got %v", err)
	}
	if source.projectKey != "NME" {
		t.Fatalf("expected project key NME, got %q", source.projectKey)
	}
	if source.repoSlug != "micro-payment" {
		t.Fatalf("expected repo slug micro-payment, got %q", source.repoSlug)
	}
	if source.sourceBranch != "feat/CRMMS-1" {
		t.Fatalf("expected the source branch, got %q", source.sourceBranch)
	}
	if source.targetBranch != "development" {
		t.Fatalf("expected the target branch, got %q", source.targetBranch)
	}
	if source.origin != "https://bitbucket.bri.co.id" {
		t.Fatalf("expected the origin, got %q", source.origin)
	}
	if source.pullRequestID != "" {
		t.Fatalf("expected no pull request id, got %q", source.pullRequestID)
	}
}

func TestExplicitSelectedSourceParsesAPullRequestURLOnly(t *testing.T) {
	source, err := explicitSelectedSource(domain.CompareSource{
		CompareURL: "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/pull-requests/184/overview",
	})
	if err != nil {
		t.Fatalf("expected the pull request URL to be parsed, got %v", err)
	}
	if source.projectKey != "NME" || source.repoSlug != "micro-payment" {
		t.Fatalf("expected the project and repo, got %q/%q", source.projectKey, source.repoSlug)
	}
	if source.pullRequestID != "184" {
		t.Fatalf("expected pull request 184, got %q", source.pullRequestID)
	}
	if source.origin != "https://bitbucket.bri.co.id" {
		t.Fatalf("expected the origin, got %q", source.origin)
	}
}

func TestExplicitSelectedSourceKeepsExplicitFieldsOverTheURL(t *testing.T) {
	source, err := explicitSelectedSource(domain.CompareSource{
		CompareURL:   "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff?sourceBranch=feat/a&targetBranch=development",
		ProjectKey:   "OTHER",
		RepoSlug:     "other-repo",
		SourceBranch: "feat/explicit",
		TargetBranch: "main",
	})
	if err != nil {
		t.Fatalf("expected the explicit fields to be accepted, got %v", err)
	}
	if source.projectKey != "OTHER" || source.repoSlug != "other-repo" {
		t.Fatalf("expected the explicit project and repo, got %q/%q", source.projectKey, source.repoSlug)
	}
	if source.sourceBranch != "feat/explicit" || source.targetBranch != "main" {
		t.Fatalf("expected the explicit branches, got %q/%q", source.sourceBranch, source.targetBranch)
	}
}

func TestExplicitSelectedSourceRejectsAnEmptySource(t *testing.T) {
	_, err := explicitSelectedSource(domain.CompareSource{})
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
}

func TestExplicitSelectedSourceRejectsAnUnparsableCompareURL(t *testing.T) {
	_, err := explicitSelectedSource(domain.CompareSource{
		CompareURL: "https://bitbucket.bri.co.id/projects/NME/repos/micro-payment/compare/diff",
	})
	if !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("expected a bad request error, got %v", err)
	}
}

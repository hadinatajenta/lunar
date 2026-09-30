package application

import (
	"context"

	authDomain "lunar/backend/internal/auth/domain"
	bitbucketDomain "lunar/backend/internal/bitbucket/domain"
	copilotDomain "lunar/backend/internal/copilot/domain"
	jiraDomain "lunar/backend/internal/jira/domain"
)

const testUserID = "usr-dashboard"

type fakeJiraProvider struct {
	issues []jiraDomain.JiraIssue
	err    error
}

func (f *fakeJiraProvider) GetMyIssues(ctx context.Context, userID string) ([]jiraDomain.JiraIssue, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.issues, nil
}

type fakeBitbucketProvider struct {
	openPullRequests     []bitbucketDomain.PullRequest
	assignedPullRequests []bitbucketDomain.PullRequest
	err                  error
	assignedErr          error
}

func (f *fakeBitbucketProvider) ListPullRequests(ctx context.Context, userID string, filter string) ([]bitbucketDomain.PullRequest, error) {
	if filter == "mine" || filter == "assigned" {
		if f.assignedErr != nil {
			return nil, f.assignedErr
		}
		return f.assignedPullRequests, nil
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.openPullRequests, nil
}

type fakeCopilotProvider struct {
	models []copilotDomain.ModelInfo
	err    error
}

func (f *fakeCopilotProvider) ListModels(ctx context.Context, userID string) ([]copilotDomain.ModelInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.models, nil
}

type fakeCredentialProvider struct {
	secrets *authDomain.RedactedSecrets
	err     error
}

func (f *fakeCredentialProvider) GetRedactedSecrets(ctx context.Context, userID string) (*authDomain.RedactedSecrets, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.secrets, nil
}

func configuredCredentials() *fakeCredentialProvider {
	return &fakeCredentialProvider{secrets: &authDomain.RedactedSecrets{UserID: testUserID, HasBitbucketPAT: true}}
}

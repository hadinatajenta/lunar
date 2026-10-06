package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

type adversarialBranchOperations struct {
	createCalls    []string
	createErrors   map[string]error
	coordinates    domain.RepositoryCoordinates
	coordinatesErr error
}

func (o *adversarialBranchOperations) CreateBranch(_ context.Context, _ string, repoName string, branchName string, baseBranch string) (domain.CreateBranchOutcome, error) {
	o.createCalls = append(o.createCalls, repoName+"|"+branchName+"|"+baseBranch)
	if createErr, found := o.createErrors[repoName]; found {
		return domain.CreateBranchOutcome{}, createErr
	}

	return domain.CreateBranchOutcome{Branch: branchName, BaseBranch: baseBranch, HeadCommit: strings.Repeat("a", 40)}, nil
}

func (o *adversarialBranchOperations) ReadBaseBranchCandidates(_ context.Context, _ string, _ string) (domain.BaseBranchCandidates, error) {
	return domain.BaseBranchCandidates{}, nil
}

func (o *adversarialBranchOperations) ReadRepositoryCoordinates(_ context.Context, _ string, _ string) (domain.RepositoryCoordinates, error) {
	return o.coordinates, o.coordinatesErr
}

type adversarialRemoteFinder struct {
	branches []string
	err      error
	calls    int
}

func (f *adversarialRemoteFinder) SearchBranches(_ context.Context, _ string, _ string, _ string) ([]string, error) {
	f.calls++
	return f.branches, f.err
}

func adversarialOutcomeCount(outcome BranchCreationOutcome) int {
	return len(outcome.Created) + len(outcome.Failed) + len(outcome.Skipped)
}

func adversarialDependencies(operations domain.BranchOperations) BranchCreationDependencies {
	return BranchCreationDependencies{
		Operations: operations,
		Memory:     NewRememberedBaseMemory(),
		UserID:     "user-a",
		RootPath:   "/workspace",
	}
}

func adversarialTargets(repoNames ...string) []CreateBranchTarget {
	targets := make([]CreateBranchTarget, 0, len(repoNames))
	for _, repoName := range repoNames {
		targets = append(targets, CreateBranchTarget{Repo: repoName, BaseBranch: "main"})
	}
	return targets
}

func TestAdversarialCreateRequestRefusesEmptyTargetsAndEmptyKey(t *testing.T) {
	emptyTargets := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x"}
	if _, err := ValidateBranchCreationRequest(emptyTargets); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("an empty targets array must be refused with a bad request, got %v", err)
	}

	emptyKey := BranchCreationRequest{Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("scratch")}
	if _, err := ValidateBranchCreationRequest(emptyKey); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("an empty key must be refused with a bad request, got %v", err)
	}

	missingBase := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: []CreateBranchTarget{{Repo: "scratch"}}}
	if _, err := ValidateBranchCreationRequest(missingBase); !errors.Is(err, sharedErrors.ErrBadRequest) {
		t.Fatalf("a target without a base branch must be refused with a bad request, got %v", err)
	}
}

func TestAdversarialRunBranchCreationAccountsForEveryTargetWhenOperationsAreMissing(t *testing.T) {
	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha", "beta", "gamma")}

	outcome := RunBranchCreation(context.Background(), adversarialDependencies(nil), request, "feature/CRMMS-77911-x")

	if adversarialOutcomeCount(outcome) != 3 {
		t.Fatalf("every target must be accounted for, got %+v", outcome)
	}
	if len(outcome.Failed) != 3 {
		t.Fatalf("expected three failures without git operations, got %+v", outcome)
	}
	for _, failed := range outcome.Failed {
		if !strings.Contains(failed.Reason, "not configured") {
			t.Fatalf("expected the reason to name the missing operations, got %q", failed.Reason)
		}
	}
}

func TestAdversarialRunBranchCreationAccountsForEveryTargetWhenRepositoriesRepeat(t *testing.T) {
	operations := &adversarialBranchOperations{createErrors: map[string]error{"beta": domain.ErrWorkingTreeDirty}}
	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha", "alpha", "beta", "gamma")}

	outcome := RunBranchCreation(context.Background(), adversarialDependencies(operations), request, "feature/CRMMS-77911-x")

	if adversarialOutcomeCount(outcome) != 4 {
		t.Fatalf("a duplicated repository must still be accounted for once per target, got %+v", outcome)
	}
	if len(outcome.Created) != 3 || len(outcome.Failed) != 1 {
		t.Fatalf("expected three created and one failed, got %+v", outcome)
	}
	if len(operations.createCalls) != 4 {
		t.Fatalf("expected one create attempt per target, got %v", operations.createCalls)
	}
}

func TestAdversarialRunBranchCreationPrefersTheDuplicateSkipOverTheDirtyFailure(t *testing.T) {
	operations := &adversarialBranchOperations{
		createErrors: map[string]error{
			"alpha": domain.ErrBranchAlreadyExists,
			"beta":  domain.ErrWorkingTreeDirty,
		},
	}
	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha", "beta")}

	outcome := RunBranchCreation(context.Background(), adversarialDependencies(operations), request, "feature/CRMMS-77911-x")

	if adversarialOutcomeCount(outcome) != 2 {
		t.Fatalf("every target must be accounted for, got %+v", outcome)
	}
	if len(outcome.Skipped) != 1 || outcome.Skipped[0].Repo != "alpha" {
		t.Fatalf("an existing branch must be skipped, got %+v", outcome)
	}
	if len(outcome.Failed) != 1 || outcome.Failed[0].Repo != "beta" {
		t.Fatalf("a dirty tree must be failed, got %+v", outcome)
	}
}

func TestAdversarialRunBranchCreationNeverCreatesWhenTheRemoteCheckFails(t *testing.T) {
	operations := &adversarialBranchOperations{coordinates: domain.RepositoryCoordinates{Project: "NME", Slug: "alpha"}}
	finder := &adversarialRemoteFinder{err: errors.New("bitbucket is unreachable")}
	dependencies := adversarialDependencies(operations)
	dependencies.Finder = finder

	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha", "beta")}
	outcome := RunBranchCreation(context.Background(), dependencies, request, "feature/CRMMS-77911-x")

	if adversarialOutcomeCount(outcome) != 2 {
		t.Fatalf("every target must be accounted for, got %+v", outcome)
	}
	if len(outcome.Failed) != 2 {
		t.Fatalf("a failed duplicate check must fail the target, got %+v", outcome)
	}
	if len(outcome.Created) != 0 {
		t.Fatalf("a failed duplicate check must never create, got %+v", outcome.Created)
	}
	if len(operations.createCalls) != 0 {
		t.Fatalf("no git create may run when the duplicate check failed, got %v", operations.createCalls)
	}
}

func TestAdversarialRunBranchCreationSkipsWhenTheRemoteCarriesTheKey(t *testing.T) {
	operations := &adversarialBranchOperations{coordinates: domain.RepositoryCoordinates{Project: "NME", Slug: "alpha"}}
	finder := &adversarialRemoteFinder{branches: []string{"feature/CRMMS-77911-Existing-Tail"}}
	dependencies := adversarialDependencies(operations)
	dependencies.Finder = finder

	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha")}
	outcome := RunBranchCreation(context.Background(), dependencies, request, "feature/CRMMS-77911-x")

	if len(outcome.Skipped) != 1 || len(outcome.Created) != 0 {
		t.Fatalf("a remote match must skip the target, got %+v", outcome)
	}
	if !strings.Contains(outcome.Skipped[0].Reason, "feature/CRMMS-77911-Existing-Tail") {
		t.Fatalf("the skip reason must name the existing branch, got %q", outcome.Skipped[0].Reason)
	}
	if len(operations.createCalls) != 0 {
		t.Fatalf("no git create may run when the remote already carries the key, got %v", operations.createCalls)
	}
}

func TestAdversarialRunBranchCreationCreatesWithoutAnyRemoteGuard(t *testing.T) {
	operations := &adversarialBranchOperations{}
	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha")}

	outcome := RunBranchCreation(context.Background(), adversarialDependencies(operations), request, "feature/CRMMS-77911-x")

	if len(outcome.Created) != 1 {
		t.Fatalf("expected the target to be created without a remote finder, got %+v", outcome)
	}
	if len(operations.createCalls) != 1 {
		t.Fatalf("expected one git create, got %v", operations.createCalls)
	}
}

func TestAdversarialRunBranchCreationRemembersOnlyTheRequestedBase(t *testing.T) {
	operations := &adversarialBranchOperations{}
	memory := NewRememberedBaseMemory()
	dependencies := adversarialDependencies(operations)
	dependencies.Memory = memory
	dependencies.UserID = "user-a"

	request := BranchCreationRequest{
		Key:    "CRMMS-77911",
		Branch: "feature/CRMMS-77911-x",
		Targets: []CreateBranchTarget{
			{Repo: "alpha", BaseBranch: "release/production", RememberBase: true},
			{Repo: "beta", BaseBranch: "develop", RememberBase: false},
		},
	}

	outcome := RunBranchCreation(context.Background(), dependencies, request, "feature/CRMMS-77911-x")
	if adversarialOutcomeCount(outcome) != 2 {
		t.Fatalf("every target must be accounted for, got %+v", outcome)
	}
	if memory.Recall("user-a", "alpha") != "release/production" {
		t.Fatalf("the requested base must be remembered, got %q", memory.Recall("user-a", "alpha"))
	}
	if memory.Recall("user-a", "beta") != "" {
		t.Fatalf("an unrequested base must not be remembered, got %q", memory.Recall("user-a", "beta"))
	}
	if memory.Recall("user-b", "alpha") != "" {
		t.Fatalf("a remembered base must not leak across users, got %q", memory.Recall("user-b", "alpha"))
	}
}
func TestAdversarialRunBranchCreationSkipsTheRemoteGuardWhenTheOriginIsNotParseable(t *testing.T) {
	operations := &adversarialBranchOperations{}
	finder := &adversarialRemoteFinder{branches: []string{"feature/CRMMS-77911-Existing-Tail"}}
	dependencies := adversarialDependencies(operations)
	dependencies.Finder = finder

	request := BranchCreationRequest{Key: "CRMMS-77911", Branch: "feature/CRMMS-77911-x", Targets: adversarialTargets("alpha")}
	outcome := RunBranchCreation(context.Background(), dependencies, request, "feature/CRMMS-77911-x")

	if finder.calls != 0 {
		t.Fatalf("a repository without parseable coordinates cannot be checked, expected no Bitbucket call, got %d", finder.calls)
	}
	if len(outcome.Created) != 1 {
		t.Fatalf("the guard is skipped silently, so the branch is created, got %+v", outcome)
	}
}

func TestAdversarialRunBranchCreationSkipsWhenOnlyALongerDigitRunBranchExistsRemotely(t *testing.T) {
	operations := &adversarialBranchOperations{coordinates: domain.RepositoryCoordinates{Project: "NME", Slug: "alpha"}}
	finder := &adversarialRemoteFinder{branches: []string{"feature/CRMMS-63970-other-issue"}}
	dependencies := adversarialDependencies(operations)
	dependencies.Finder = finder

	request := BranchCreationRequest{Key: "CRMMS-6397", Branch: "feature/CRMMS-6397-x", Targets: adversarialTargets("alpha")}
	outcome := RunBranchCreation(context.Background(), dependencies, request, "feature/CRMMS-6397-x")

	if len(outcome.Skipped) != 1 || len(outcome.Created) != 0 {
		t.Fatalf("the remote result is not re-verified against the key, so a longer digit run blocks the create, got %+v", outcome)
	}
	t.Logf("characterisation: key CRMMS-6397 was skipped because the remote returned %v", finder.branches)
}
func TestAdversarialIssueBranchesCannotBeCorruptedByTheCaller(t *testing.T) {
	scanner := keyedBranchScanner()
	service := NewIssueBranchService(scanner)

	first, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(first.Matches) == 0 {
		t.Fatal("expected matches to mutate")
	}

	first.Matches[0].Branch = "feature/tampered"
	first.Matches[0].Repository = "tampered"
	first.Matches = append(first.Matches, BranchMatch{Repository: "injected", Branch: "feature/injected"})

	second, err := service.IssueBranchesAtRoot(context.Background(), "/workspace", "CRMMS-6397")
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}

	if scanner.calls != 1 {
		t.Fatalf("expected the cached scan to serve both reads, got %d scans", scanner.calls)
	}
	if len(second.Matches) != len(first.Matches)-1 {
		t.Fatalf("the cached index must not see the caller mutation, got %+v", second.Matches)
	}
	for _, match := range second.Matches {
		if match.Repository == "tampered" || match.Branch == "feature/tampered" || match.Repository == "injected" {
			t.Fatalf("a caller mutation reached the cached index, got %+v", second.Matches)
		}
	}
}

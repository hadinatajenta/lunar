package infrastructure

import "context"

func ResolveHeadRef(ctx context.Context, repositoryPath string, branch string) (string, bool, error) {
	local, err := LocalBranchExists(ctx, repositoryPath, branch)
	if err != nil {
		return "", false, err
	}
	if local {
		return branch, true, nil
	}

	remote, err := RemoteBranchExists(ctx, repositoryPath, branch)
	if err != nil {
		return "", false, err
	}
	if remote {
		return "origin/" + branch, true, nil
	}
	return "", false, nil
}

func ResolveBaseRef(ctx context.Context, repositoryPath string, baseRef string) (string, error) {
	local, err := LocalBranchExists(ctx, repositoryPath, baseRef)
	if err != nil {
		return "", err
	}
	if local {
		return baseRef, nil
	}

	remote, err := RemoteBranchExists(ctx, repositoryPath, baseRef)
	if err != nil {
		return "", err
	}
	if remote {
		return "origin/" + baseRef, nil
	}
	return baseRef, nil
}

func LocalBranchExists(ctx context.Context, repositoryPath string, branch string) (bool, error) {
	return sopRefExists(ctx, repositoryPath, "refs/heads/"+branch)
}

func RemoteBranchExists(ctx context.Context, repositoryPath string, branch string) (bool, error) {
	return sopRefExists(ctx, repositoryPath, "refs/remotes/origin/"+branch)
}

func sopRefExists(ctx context.Context, repositoryPath string, reference string) (bool, error) {
	_, exitCode, err := runSOPGitCommand(ctx, repositoryPath, "show-ref", "--verify", "--quiet", reference)
	if err == nil {
		return true, nil
	}
	if exitCode == 1 {
		return false, nil
	}
	return false, err
}

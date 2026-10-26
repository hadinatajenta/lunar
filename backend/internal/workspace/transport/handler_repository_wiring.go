package transport

import workspaceApp "lunar/backend/internal/workspace/application"

func repositoryRemoteSource(remote workspaceApp.RemoteRepositoryProvider) workspaceApp.RemoteRepositorySource {
	if remote == nil {
		return nil
	}

	return workspaceApp.NewRemoteRepositoryCatalogueSource(remote)
}

func repositoryRemoteScanner(remote workspaceApp.RemoteRepositoryProvider) *workspaceApp.RemoteCandidateScanner {
	if remote == nil {
		return nil
	}

	return workspaceApp.NewRemoteCandidateScanner(remote)
}

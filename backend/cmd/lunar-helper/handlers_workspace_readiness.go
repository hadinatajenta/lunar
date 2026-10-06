package main

import (
	"net/http"

	workspaceApplication "lunar/backend/internal/workspace/application"
	workspaceTransport "lunar/backend/internal/workspace/transport"
	"lunar/backend/internal/workspace/infrastructure"
)

const readinessPath = "/readiness"

func registerWorkspaceReadinessRoute(mux *http.ServeMux, dependencies helperDependencies) {
	service := workspaceApplication.NewWorkspaceReadinessService(infrastructure.NewGitInspector(dependencies.allowedRoots), dependencies.allowedRoots)
	handler := workspaceTransport.NewWorkspaceReadinessHandler(service, nil, nil)

	mux.Handle("GET "+readinessPath, requireToken(dependencies.token, http.HandlerFunc(handler.GetHelperReadiness)))
}

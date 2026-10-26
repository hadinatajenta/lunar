# Jira agent tools: make the Jira BRI toggle functional

## Objective

Enabling the "Jira BRI" domain in Copilot must give the model real Jira tools, so a question like
"check my Jira, what is in the backlog?" is answered from the user's Jira instead of "I have no Jira
access".

## Findings

- The domain toggle already exists and is sent to the helper as `domains: ["jira"]`, but no Jira tool
  is registered, so `ScopedDefinitions` returns nothing and the model reports no access.
  `backend/internal/agent/application/registry.go` already maps `list_my_jira_issues`,
  `get_jira_issue_detail`, and `get_active_sprint_issues` to `DomainJira`, and `prompt.go` already has
  the Jira expertise block. Only the tools are missing.
- The helper registers tools in `backend/cmd/lunar-helper/agent_builder.go` with
  `agenttools.NewTools(indexStore, reader)`; it never receives the backend URL or the session token,
  which the agent request already carries (`backend_url`, `session_token`).
- The backend already exposes the Jira data with the vaulted PAT: `GET /api/jira/issues`,
  `GET /api/jira/backlog`, `GET /api/jira/issues/{key}`.

## Design

The Jira tools run in the helper and call the Lunar backend over HTTP with the user's session token, so
the Jira PAT never leaves the vault and the helper needs no Jira client.

New package `backend/internal/jira/tools/` implementing `agentdomain.Tool`:

| Tool | Backend call | Arguments |
|---|---|---|
| `list_my_jira_issues` | `GET /api/jira/issues` | none |
| `get_jira_issue_detail` | `GET /api/jira/issues/{key}` | `issue_key` (required) |
| `get_active_sprint_issues` | `GET /api/jira/backlog` | none |

- A small `BackendClient{ baseURL, sessionToken, httpClient }` with a `getJSON(ctx, path)` that sets
  `Authorization: Bearer <session_token>`, bounds the response body, and has its own timeout.
- Each tool returns a compact text summary plus `SourceReference`s (`type: "jira"`) so the UI shows
  citations, following `codeindex/application/tools` for the `Definition()` and `Execute()` shape.
- Arguments are validated (non-empty key, no path traversal) before the request.

Helper wiring (`backend/cmd/lunar-helper/agent_builder.go` and `handlers_agent.go`):

- `buildAgent` gains `backendURL` and `sessionToken`; `handleAgentAsk` passes them.
- The registry registers `agenttools.NewTools(indexStore, reader)` plus `jiratools.NewTools(client)`.
- `handleAgentAsk` keeps requiring the model and provider credentials; the code index is still needed
  for the code tools, so the existing `indexStore == nil` guard stays.

## Verification

- Unit tests with an `httptest` backend: each tool hits the right path with the bearer token, parses the
  response, bounds an oversized body, and validates its arguments.
- Scoping: `ScopedDefinitions(["jira"])` returns exactly the three Jira tools; `["servicemap"]` returns
  none of them.
- `go build ./...`, `go test -race ./internal/jira/... ./cmd/lunar-helper/...`, `go vet`, `gofmt`.
- A live check: enable Jira BRI, ask "what is in my backlog", and confirm a real answer with Jira
  citations.

## Risks

| Risk | Mitigation |
|---|---|
| The helper holds a Jira PAT | It never does; the backend performs the Jira call |
| A huge Jira response reaching the model | Bound the body and summarize to a compact list |
| The model calls a Jira tool while Jira is not configured | The backend returns the existing auth/validation error, surfaced as a tool result |

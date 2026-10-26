# Task checklist

- [x] `jira/tools`: `BackendClient` with bounded body and timeout
- [x] `list_my_jira_issues`, `get_jira_issue_detail`, `get_active_sprint_issues` with schemas and sources
- [x] Helper `buildAgent` receives `backendURL` and `sessionToken` and registers the Jira tools
- [x] Domain scoping returns the Jira tools for the `jira` domain only
- [x] Tests with `httptest`: path, bearer token, parsing, bounds, argument validation, scoping
- [x] `go build`, `go test -race`, `go vet`, `gofmt`
- [x] `guide.md` section 7 tool list and change log
- [x] Live check against real Jira: `domains: ["jira"]` answered from the user's own Jira with
      `list_my_jira_issues` and `get_active_sprint_issues` tool calls and `type: jira` sources

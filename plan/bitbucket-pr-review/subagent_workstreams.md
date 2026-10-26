# Subagent workstreams

Three workstreams ran, with file ownership split so that no two writers touched the same file.

| Workstream | Owner | Files | Handoff |
|---|---|---|---|
| Backend review pipeline | parent | `backend/internal/bitbucket/**`, `backend/internal/copilot/**` except the new test file | None. The contract was frozen in `implementation_plan.md` before either frontend workstream started. |
| Client option tests | subagent | `backend/internal/copilot/infrastructure/llm_client_options_test.go` only | Independent because the file did not exist. It asserts the behaviour of `ChatCompletion` through an `httptest` server, so it needs no writer access to the implementation. |
| End-to-end suite | subagent | `frontend/e2e/**` only | Received the frozen DOM contract and the frozen API contract up front, so it never had to read the implementation. The parent did not touch `frontend/e2e/**` at any point, which is why the two could run together. |

## Rules given to both subagents

- Zero code comments.
- English identifiers, strings, and test titles.
- No em dashes.
- Do not edit any file outside the owned set.
- Report the exact command, the raw output, and anything not verified.
- Never weaken an assertion to make a test pass.

## Verification boundary

The Go build cache lives outside the workspace sandbox, so every Go command ran with
`GOCACHE=/Users/erendt/Code/lunar/.gocache`. `go build ./...` and `go test ./...` cannot pass in this
working tree for a reason unrelated to this change: the untracked file
`backend/internal/workspace/application/remote_branch_scan_keys.go` contains a literal NUL byte inside
a string literal, which Go rejects. That file belongs to another change in flight and was left
untouched. Every package this change touches was built and tested on its own.

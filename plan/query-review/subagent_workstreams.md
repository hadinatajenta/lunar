# Subagent workstreams

Four workstreams. File ownership is split so no two writers touch the same file. The parent owns
`backend/internal/app/app.go`, `guide.md`, `plan/BACKLOG.md`, and the final integration and
verification.

| Workstream | Owner | Files | Depends on |
|---|---|---|---|
| WS-A query review backend | subagent `backend` | `backend/internal/queryreview/**`, `backend/internal/bitbucket/domain/repository.go`, `backend/internal/bitbucket/infrastructure/atlassian_bitbucket_client.go`, its test file | Frozen contract in `plan/query-review/implementation_plan.md` |
| WS-B UT docs backend and helper | subagent `backend` | `backend/internal/utdocs/**`, `backend/cmd/lunar-helper/handlers_utdocs.go`, `backend/cmd/lunar-helper/handlers_utdocs_test.go`, `backend/cmd/lunar-helper/server.go` | Frozen contract in `plan/ut-docs/implementation_plan.md` |
| WS-C Confluence panel | subagent `frontend` | `frontend/src/features/confluence/**`, `frontend/e2e/**` confluence specs and page objects | Frozen API contracts in both plans. The contracts are frozen before this workstream starts, so it never reads the backend implementation |
| WS-D integration and docs | parent | `backend/internal/app/app.go`, `guide.md`, `plan/BACKLOG.md`, verification | WS-A, WS-B, WS-C |

## Rules given to every subagent

- Zero code comments. English identifiers, strings, test titles, and UI copy.
- Never edit a file outside the owned set. The parent wires `app.go` and the helper registration is
  WS-B's only shared-file edit.
- Unit tests never open a database, never start a server, and never call a model. Stub the seams.
- Do not weaken an assertion to make a test pass. A test that cannot fail is a defect.
- Run the formatter, the build, and the package tests before reporting.
- Report the exact commands, the raw results, and anything not verified. No claims without output.
- Do not run `git commit`, `git checkout`, or any destructive command.
- `GOCACHE=/Users/erendt/Code/lunar/.gocache` for every Go command, as the previous plan recorded.

## Handoff

- WS-C receives the frozen HTTP contracts only. If the contract must change, the change is proposed
  to the parent first and the parent updates the plan.
- The parent reviews every subagent diff against the plan, checks naming and comment rules, runs the
  full build and test set, and updates `guide.md`.

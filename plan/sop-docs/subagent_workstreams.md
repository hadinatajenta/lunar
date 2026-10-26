# Subagent workstreams

| Workstream | Owner | Files | Depends on |
|---|---|---|---|
| Backend SOP and helper scan | subagent `backend` | `backend/internal/sop/**`, `backend/cmd/lunar-helper/handlers_sop.go` (+ test), `backend/internal/queryreview/application/mig_targets.go` for the all-repository reader | Frozen contract in `implementation_plan.md` |
| Frontend SOP panel | subagent `frontend` | `frontend/src/features/confluence/**`, `frontend/e2e/**` | Frozen HTTP contracts |
| Integration and docs | parent | `backend/internal/app/app.go`, `backend/cmd/lunar-helper/server.go`, `guide.md`, verification | Both |

## Rules given to both subagents

- Zero code comments. English identifiers; Indonesian only inside generated document content.
- Never open a database; unit tests use stubs and `httptest` only.
- Never overwrite content outside the managed region.
- Do not edit `app.go`; the parent wires it.
- Report the exact commands, raw results, and anything not verified.

## Handoff

The frontend receives the frozen HTTP contracts only. Any contract change is proposed to the parent first.

# OpenCode Zen Provider - Subagent Workstreams

## File ownership

One writer per file. A stream needing a change outside its list asks the parent.

| Stream | May write | Must not write |
|---|---|---|
| W1 | `backend/internal/copilot/infrastructure/provider_urls.go`, `backend/internal/copilot/infrastructure/llm_client.go`, `backend/internal/copilot/application/catalog.go`, `backend/internal/copilot/application/provider_key.go`, `backend/internal/agent/infrastructure/toolcall.go`, `backend/internal/config/config.go`, `backend/internal/app/app.go` | any frontend file, `guide.md` |
| W2 | `frontend/src/features/copilot/utils/models.ts`, `frontend/src/features/copilot/types.ts`, `frontend/src/features/settings/components/AiProvidersSettings.vue`, `frontend/src/features/copilot/utils/attachment-rules.ts` | any backend file, `guide.md` |
| W3 | `guide.md`, `plan/opencode-provider/**` | source files owned by W1 or W2 |

## Handoff rules

- The model table is the contract. W1 owns it; W2 reads the ids and names from it and may not invent
  an id or a display name.
- The `protocol` field is additive on the credentials response. W1 may not change an existing field
  or status code.
- W3 writes `guide.md` only after W1 and W2 report, so the documented behaviour matches what ships.
- Every stream reports what it could not verify, and why.

## Definition of done for a stream

1. Owned files written, no comments, under 300 lines each.
2. Its checklist items in `task.md` are true, each with named evidence.
3. Build and tests run, with the command and the raw result reported.
4. Anything unverified is stated explicitly.

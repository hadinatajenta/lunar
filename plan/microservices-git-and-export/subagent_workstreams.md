# Microservices BRI - Subagent Workstreams

## Rules

One writer per file. A stream needing a change outside its list asks the parent. Every phase ends
with a QA agent that owns nothing in `src/` and may only add tests and report.

At most three agents run per phase, QA included.

## Phase 1 ownership

| Agent | May write | Must not write |
|---|---|---|
| 1A backend | `backend/internal/workspace/**`, `backend/cmd/lunar-helper/**` for the new routes only | any frontend file, `guide.md` |
| 1B frontend | `frontend/src/features/microservices/**` | any backend file, `guide.md` |
| 1Q QA | `frontend/e2e/**`, backend `_test.go` under `workspace/` | production files |

Frozen contract:

```text
GET  /branches                  -> { branches: [{ name, is_current, updated_relative }] }
POST /checkout  { branch }      -> { branch }
POST /fetch                     -> { ahead, behind }
POST /pull                      -> { branch, updated_relative }
POST /stash                     -> { stash_count }
POST /discard                   -> { restored_count }
any failure                     -> { error: string } with 400, 404, 409, or 504
```

Helper paths sit at the root; server paths are the same under `/api/workspace`. `root` and
`name` are passed on every call so both sources resolve identically.

## Phase 2 ownership

| Agent | May write | Must not write |
|---|---|---|
| 2A backend | `backend/internal/postman/**` (new), `backend/cmd/lunar-helper/**` for the new route only, `backend/internal/app/app.go` for the route only | any frontend file, `guide.md`, `codeindex/infrastructure/walker.go` |
| 2B frontend | `ModelPickerModal.vue`, new export components, `microservices/api/**`, `microservices/types.ts` | any backend file, `guide.md` |
| 2Q QA | `frontend/e2e/**`, `backend/internal/postman/**/*_test.go`, fixtures under `backend/internal/postman/testdata/` | production files |

Frozen contract:

```text
POST /export  { root, name, model }   -> text/event-stream
  event: stage   { stage, detail }        selecting | prompt | model | validating
  event: result  { service, request_count, folder_count, collection }
  event: error   { error }

Failures:
  409  no provider key configured
  422  service cannot be exported, reason included
  422  the collection is invalid or has no requests
  504  the provider timed out
```

The collection travels in the result event. **The server writes nothing to disk.**

## Handoff rules

- The Phase 1 contract is frozen. 1A may not change a field without 1B in the same change.
- The Phase 2 event names, the result shape, and the completeness rules are frozen so 2B can build
  against them before 2A finishes.
- `codeindex/infrastructure/walker.go` must not be modified. The export owns its selector.
- The export must not write any file, anywhere. Any stream that adds a write is out of contract.
- The effort value is `medium` in the request payload and must never surface in the UI.
- 3B writes `guide.md` only after 3A reports, so the documentation matches what ships.
- Every stream reports what it did not verify, and why.

## Definition of done for a stream

1. Owned files written, no comments, under 300 lines each.
2. Its checklist items in `task.md` are true, each with named evidence.
3. Build and tests run, with the exact command and the raw result reported.
4. Anything unverified is stated explicitly rather than implied to work.

# Task checklist

## Backend module

- [x] `utdocs/domain`: test file, blocked file, run result, document result
- [x] `utdocs/application/classify.go`: destructive marker scan, integration tag, sqlmock override, PHP exclusion
- [x] `utdocs/application/service.go`: scan, classify, run, screenshot, document orchestration
- [x] `utdocs/infrastructure/runner.go`: fixed `go test` command, deadline, output cap, ANSI strip, executor seam
- [x] `utdocs/infrastructure/scanner.go`: changed files on the branch, candidate test files, PHP surfaced but never run
- [x] `utdocs/infrastructure/screenshot.go`: terminal PNG renderer with caps and truncation marker
- [x] `utdocs/infrastructure/prompt.go`: `utdocs-v1`, mock-only example rule, blocked files stated
- [x] `utdocs/infrastructure/model_client.go`: `ModelProvider` + `LLMClient`, JSON output, repair round
- [x] `utdocs/infrastructure/run_store.go`: `~/.lunar/ut-runs/`, `0700` directory, `0600` files, newest 20 kept
- [x] `utdocs/transport/handler.go`: `POST /api/confluence/documents/{id}/ut-docs`
- [x] Unit tests: classification table, PHP exclusion, integration tag, runner seam, screenshot caps, prompt invariants, model repair, run store
- [x] `go build`, `go vet`, `go test`, and `go test -race` on the touched packages

## Helper

- [x] `handlers_utdocs.go`: `POST /ut-runs` SSE with phases and streamed output
- [x] `GET /ut-runs/{id}/screenshot.png` with the bearer token
- [x] Allowlist gate reused from the destructive git operations
- [x] Run store on disk under `~/.lunar/ut-runs/`, `0600`, newest 20 kept
- [x] Helper endpoint tests against a scratch repository
- [x] `server.go` registration

## Frontend

- [x] `api/ut-docs-api.ts`: helper run request (SSE) and backend document call
- [x] `useUtDocs.ts`: phase state, run result, screenshot blob URL, error and retry
- [x] `UtRunSummary.vue`: status pill, counts, duration, executed and blocked files
- [x] `UtDocsPanel.vue`: generated document, blocked notice, screenshot, copy markdown
- [x] `DocumentActionsRow.vue` and `DocumentAiPanel.vue` integrate both new flows without stubs
- [x] Unit tests for the phase state machine
- [x] Playwright for the button states and the mocked run and document responses
- [x] `vite build`, `node --test`, `npx playwright test tests/confluence`

## Integration, documentation, live check

- [x] `app.go` and helper `server.go` wiring
- [x] `guide.md` section 5 and change log
- [x] `plan/BACKLOG.md` updated for the deferred Confluence write-back
- [ ] Live run on one real repository: checkout, blocked list, mock-only packages, screenshot, document. Not verified; coverage is unit and scratch-repository based

## Stage 2: MIG-driven runs and the Normal Test Scenario table (frontend)

- [x] MIG targets API client `fetchMigTargets` and apply client `applyUtDocs`
- [x] `useUtDocs` controller reworked to load targets, run one helper pass per repository, aggregate the document, and apply
- [x] `UtRunSummary.vue` reports status, counts, duration, stashed or restored, and the blocked files
- [x] `UtTestScenarioTable.vue` renders the seven columns, the group row, the result pill, and the per-repository screenshot
- [x] `utils/ut-test-table.ts` builds the table HTML and text with grouping and escaping
- [x] `Copy table` writes the table as rich HTML with a `text/plain` fallback
- [x] Instant apply wired for `ut` documents in `DocumentDetailView.vue` and `DocumentActionsRow.vue`
- [x] Unit tests for the table builder and the MIG-driven flow state
- [x] Playwright mocks for `mig-targets`, `ut-runs` (SSE), `ut-docs`, and `ut-docs/apply`, covering the seven headers, grouped layout, blocked note, and apply success or failure
- [x] `vue-tsc -b`, `vite build`, `node --test`, and Playwright `tests/confluence`

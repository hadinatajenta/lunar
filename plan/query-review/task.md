# Task checklist

## Backend

- [x] `queryreview/domain`: request, source, analyzed file, query item, result, error codes
- [x] `queryreview/application/service.go`: source resolution, diff fetch, filter, model call, assembly
- [x] `queryreview/application/discovery.go`: Bitbucket compare and PR link extraction from the document body
- [x] `queryreview/infrastructure/diff_source.go`: added-lines-only extraction with new line numbers, per-file and total budgets
- [x] `queryreview/infrastructure/review_client.go`: `ModelProvider` + `LLMClient`, JSON output, one repair round
- [x] `queryreview/infrastructure/prompt.go`: `queryreview-v1`, untrusted block, added-lines-only rules
- [x] `queryreview/transport/handler.go`: `POST /api/confluence/documents/{id}/query-review`
- [x] `bitbucket/infrastructure`: `GetCompareDiff` with tests
- [x] Unit tests: filter drops context and removed lines, dedup, empty result, discovery single/ambiguous/none, prompt invariants, model parse repair, total budget
- [x] `go build` and `go test` on the touched packages
- [x] `app.go` wiring behind the auth middleware
- [x] Parent review added a total prompt budget (`queryReviewTotalCharBudget`) after WS-A finished

## Frontend

- [x] `confluence-api.ts`: `requestQueryReview`, `requestUtDocs`
- [x] `useQueryReview.ts`: idle, loading, error, empty, success; stale-response drop
- [x] `QueryReviewPanel.vue`: model picker, source line, analyzed files, queries table, copy JSON and markdown, manual compare URL
- [x] `DocumentActionsRow.vue`: emits `generate`, shows busy state, no local stub
- [x] `DocumentAiPanel.vue`: renders the query review result
- [x] Unit tests for the composable state machine
- [x] Playwright: existing stub assertions replaced, positive and error cases added
- [x] `vite build` and `node --test` for the Confluence tests
- [x] `npx playwright test tests/confluence`: 31 passed

## Integration and documentation

- [x] `app.go`: construct and register the module behind the auth middleware
- [x] `guide.md` section 5 and change log
- [x] `plan/BACKLOG.md` B.7 narrowed, B.25 added for the deferred Confluence write-back

## MIG-driven multi-repository (stage 2)

- [x] Confluence page tree: `GetPageAncestors`, `GetChildPages`, `GetPageStorage` on the client, repository, and service
- [x] MIG resolver: immediate parent, `-MIG` title match with the issue key, raw storage read
- [x] MIG table parser: columns read by header name, repository/branch/link cells, per-row skip reporting
- [x] Single exclusion slice `excludedRepositorySlugs = {"fe-mms", "mms"}`
- [x] Multi-repository extraction with a bounded pool of three, per-repository status and error isolation
- [x] `[repository]` feature prefix and `repository` on every query
- [x] `AMBIGUOUS_COMPARE_LINK` retired; `NO_COMPARE_LINK` only when the MIG page is missing
- [x] Backend tests: page tree, MIG parser, resolver, multi-repository service, handler contract
- [x] Frontend: per-repository cards, skipped-repository note, manual URL retry for `NO_COMPARE_LINK`
- [x] Frontend tests: unit 30/30, Playwright confluence 34 passed
- [x] `guide.md` section 5 and change log, `plan/BACKLOG.md` B.26 updated

## Stage 3: Capture explain table, copy, and Instant apply

- [x] Backend Confluence write client `UpdatePageStorage` with `PUT /rest/api/content/{id}` and `version + 1`
- [x] `BuildCaptureExplainTable` and `ReplaceCaptureExplainTable` with escaping, bounds, and marker handling
- [x] `ApplyQueryReviewTable` service and `POST /api/confluence/documents/{id}/query-review/apply`
- [x] Repositories with no extracted query are skipped; at least one query overall is required
- [x] Frontend table columns `No, Fitur, Query Statement, Index, Capture Explain / Execution Plan`
- [x] Copy as rich HTML (`ClipboardItem`) with a plain-text fallback, per repository and whole result
- [x] Instant apply wiring through the query review store, success toast, in-panel error and retry
- [x] Backend tests, frontend unit 41/41, Playwright confluence 36 passed, full `go test ./...` green
- [x] `guide.md` section 5 and change log, `plan/BACKLOG.md` B.25 and B.7 updated

## Stage 4: EXPLAIN prefix, index guidance, and width

- [x] `EXPLAIN` forced in `normalizeQueries`, not only in the prompt
- [x] `create_index` on `QueryItem`, `ApplyQuery`, and the model payload
- [x] Index cell renders the possible key and the `CREATE INDEX` statement on screen, in copy, and in the Confluence table
- [x] Apply request carries `create_index` from the frontend
- [x] Per-query Copy button removed; whole-table and raw JSON copy kept
- [x] Detail page widened from 900px to 1440px
- [x] Backend tests, frontend unit 46/46, Playwright confluence 36 passed, full `go test ./...` green
- [x] `guide.md` section 5 and change log updated

## Stage 5: empty-result row, light mode, and document body tables

- [x] `BuildCaptureExplainTable` and the frontend builder fall back to a single `colspan="5"` row with the Indonesian message
- [x] The card renders that fallback row in the table instead of a paragraph
- [x] Instant apply is allowed with zero queries and writes the fallback row
- [x] AI action button, status pills, and notices use `--positive`/`--warning`/`--danger` so light mode is readable
- [x] Document body tables: content-driven layout, `th` background, collapsed cell paragraphs, horizontal scroll
- [x] Backend tests, frontend unit 49/49, Playwright confluence 36 passed, full `go test ./...` green
- [x] `guide.md` section 5 and change log updated

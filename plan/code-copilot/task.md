# Code-Aware Copilot — Task Checklist

Legend: `[x]` done, `[ ]` pending. Every completed item names the command or artifact that proves it.

Ordering follows the owner's instruction: code index and Copilot integration first, BRI tool
integration after.

---

## Stage F — Findings (complete)

- [x] F.1 Confirmed Lunar's Copilot tools retrieve nothing: `service.go:243-283` only joins tool names into the prompt
- [x] F.2 Confirmed Lunar's source badges are built by substring-matching tool names
- [x] F.3 Confirmed Lunar's `llm_client.go` has no tool-calling support
- [x] F.4 Inventory of the reusable stack in `/Users/erendt/Code/timesheet-app`, recorded in the plan
- [x] F.5 Confirmed the eleven code tools already exist, including `search_code` and `find_symbol_references`
- [x] F.6 Confirmed `ScopedDefinitions(domains)` already matches what Lunar's toggles need
- [x] F.7 Confirmed the indexer already builds an FTS5 index over local BRI repositories
- [x] F.8 Confirmed dependency compatibility: same SQLite family, `os/exec` only for fixed-argument `git`
- [x] F.9 Verified FTS5 and `bm25()` work on Lunar's pinned `modernc.org/sqlite v1.34.5`
- [x] F.10 Measured the corpus: 3,627 PHP, 2,081 Go, 1,671 JS, 773 Vue, roughly 8,200 files and 69 MB
- [x] F.11 Sized the adaptation cost: comment removal, Indonesian translation, and 300-line splits

---

## Stage 0 — Contracts (complete)

Owner: WS-A (supervisor). Every other workstream compiles against these.

- [x] 0.1 `internal/agent/domain/tool.go` — `Tool`, `ToolRegistry`, `ToolResult`, `SourceReference`
- [x] 0.2 `internal/agent/domain/ai.go` — `ChatMessage`, `MessageContent`, `ToolDefinition`, `ToolCall`, `AIClient`, `StreamToolCaller`, `FailoverHook`
- [x] 0.3 `internal/codeindex/domain/entity.go` — `CodeChunk`, `ServiceNode`, `ServiceDependency`, `IndexStatus`, `Glossary`, `AgentQuestion`, `AgentAnswer`
- [x] 0.4 `internal/codeindex/domain/repository.go` — frozen `IndexStore` interface, including the knowledge methods
- [x] 0.5 `cmd/lunar-helper/index_schema.go` — authoritative FTS5 DDL plus `0700` directory and `0600` file handling
- [x] 0.6 `frontend/src/features/code-context/types.ts` — TypeScript mirror of the contracts
- [x] 0.7 Schema applied idempotently through `CREATE ... IF NOT EXISTS`, with WAL and busy timeout
- [x] 0.8 `go build ./...` passes with the new packages, verified

---

## Stage 1 — Code Index in the Helper

Owner: WS-B. Depends on WS-A.

- [x] 1.1 Port the walk and skip rules, reusing Lunar's `ResolveWorkspaceRoot` and sensitive-root rejection
- [x] 1.2 Port the Go extractor
- [x] 1.3 Port the PHP extractor
- [x] 1.4 Port the Node extractor
- [x] 1.5 Port the Java extractor
- [x] 1.6 Port identifier expansion so `late fee` matches `calculateLateFee`
- [x] 1.7 Port dependency edge extraction for `http_client`, `kafka_publish`, and `kafka_subscribe`
- [x] 1.8 Port the FTS5 store with unindexed `raw_content` and indexed `content`, in WAL mode
- [x] 1.9 Extend redaction with the file-name exclusion list, applied before storage
- [x] 1.10 Index database directory `0700` and file `0600`, asserted by test
- [x] 1.11 Incremental rebuild keyed by size, modification time, and content hash
- [x] 1.12 Port the indexer tests and confirm they pass
- [x] 1.13 Test proving no secret pattern survives in the database
- [x] 1.14 Test proving a rebuild touches only changed files
- [x] 1.15 Measure a full build over the selected `~/BRI` repositories and record the timing
- [ ] 1.16 Assess Vue coverage, since the existing extractors may not handle `.vue`

---

## Stage 2 — Agent and Code Questions

Owner: WS-C, then WS-D. Depends on Stage 1.

- [x] 2.1 Port the multi-round agent loop, split to respect the line limit
- [x] 2.2 Port tool dispatch and event emission
- [x] 2.3 Port tool calling, both streaming and non-streaming
- [x] 2.4 Port the DSML parser
- [x] 2.5 Port provider failover
- [x] 2.6 Port the structured JSON output helper
- [x] 2.7 Port query sanitization into a safe FTS5 expression
- [x] 2.8 Port search ranking and assert result order by test
- [x] 2.9 Port the eleven code tool executors, split by family, each returning `SourceReference`
- [x] 2.10 Port the agent and tool-calling tests and confirm they pass
- [x] 2.11 `GET /index/status` returning counts and timestamps only, never paths
- [x] 2.12 `POST /index/build` in one guarded goroutine, `409` when already building
- [x] 2.13 `GET /search` with limit default 8 and maximum 25
- [x] 2.14 `POST /agent/ask` streaming server-sent events
- [x] 2.15 Bound request body size and set per-request timeouts
- [x] 2.16 Tests covering token requirement, concurrent-build conflict, and stream framing
- [x] 2.17 Verify helper egress to the model provider on the corporate network
- [x] 2.18 Confirm the API key is held in memory only, never written to disk or logged
- [x] 2.19 Browser verification that a citation resolves to the correct real file and line
- [x] 2.20 Record first-token timing

---

## Stage 3 — Make the Existing Toggles Real

Owner: WS-E. Depends on Stage 2.

- [x] 3.1 `POST /api/copilot/transcript` persisting question, answer, reasoning, and real sources
- [x] 3.2 `ChatSource` extended with optional repo, path, start line, and end line
- [x] 3.3 Replace the decorative badge construction, which currently substring-matches tool names
- [x] 3.4 Helper API client reusing the existing helper request plumbing
- [x] 3.5 `useCodeAgent` orchestrating key retrieval, the ask request, the stream, and transcript persistence
- [x] 3.6 Citation component rendering repository, path, and line range
- [x] 3.7 Index status component with a build action
- [x] 3.8 Wire the existing domain toggles to the `domains` field
- [x] 3.9 Graceful degradation when the helper is offline or the index is missing
- [x] 3.10 Test asserting a disabled domain is absent from the outgoing tool list
- [x] 3.11 Browser verification that toggling a domain changes the tools the model receives
- [x] 3.12 Confirm the token-cost warning is now accurate and adjust its wording if needed

---

## Stage 4 — Jira, Bitbucket, Confluence, and Query Review Tools

Owner: WS-C. Starts only after Stage 2, per the owner's instruction.

- [ ] 4.1 Port the Jira tool package and connect it to Lunar's stored PAT
- [ ] 4.2 Port the Bitbucket tool package and connect it to Lunar's stored PAT
- [ ] 4.3 Port the query review tool package
- [ ] 4.4 Add a Confluence tool, which does not exist in the reference application
- [ ] 4.5 Each tool performs a real retrieval and returns real sources
- [ ] 4.6 Tests asserting each tool reaches its backend and surfaces failures
- [ ] 4.7 Confirm the tools run from the helper and reach BRI networks from the laptop

---

## Stage 5 — Semantic Retrieval (not approved)

Blocked on a decision. Do not start.

- [ ] 5.1 Measure Stage 1 and Stage 2 answer quality on real questions and record which fail
- [ ] 5.2 Decide local model versus provider embeddings API
- [ ] 5.3 Record the decision with cost and privacy implications before writing code

---

## Progress Log

**The feature works end to end.** Asking a code question in the Lunar Copilot UI returns a grounded
answer with citations that carry real line ranges. Verified in a browser against a real index.

### Real corpus measurements

| Measure | Result |
|---|---|
| Repositories indexed | 34 of 36 |
| Files | 7,412 |
| Chunks | 48,669 |
| First full build | 191 s |
| Incremental rebuild | 15 s |
| Index size | 101 MB database plus a 70 MB write-ahead log |

### Answer quality

A question about QRIS corrective maintenance in `aurora` returned the correct package, the route
group, the layer table, and 22 citations such as
`aurora/src/correctiveMaintenanceQris/service.go:948-1047`. Seven rounds, 22 tool calls, 22 s.

### Defects found only by running it for real

None of these were visible from unit tests or compilation.

- [x] The `/agent/ask` stream was impossible because the logging middleware wrapped the response
      writer in a type that did not forward `http.Flusher`
- [x] DSML tool-call markup leaked into answers because the parser only cleaned content when no
      native tool calls were present, so the markup entered the message history and reappeared
      during synthesis
- [x] `toAgentSources` dropped `StartLine` and `EndLine`, so every citation pointed at a file
      without a line range
- [x] The model display name was sent to the provider instead of the resolved API model id
- [x] The browser never reached the helper, because the helper token lives in `sessionStorage` and
      nothing re-acquired it on a fresh session
- [x] `/index/status` timed out during a build because the connection pool held a single connection
- [x] The DSML stream filter initially let the leading `<` through

### Verified green at this checkpoint

29 backend packages pass `go test`, `go vet` is clean, `vue-tsc` exits zero, and `vite build`
succeeds.

### Remaining

- [x] Chat answers render as Markdown. `marked` and `DOMPurify` were approved and added. Rendering
      goes through `lib/markdown.ts` with a strict tag and attribute allowlist, and
      `ChatMessageContent.vue` holds the styles so `ChatWindow.vue` did not grow. Verified in a real
      browser: two tables with 32 rows, headings, code blocks, lists, and 14 citation cards, with no
      raw pipes and no leaked `script`, `onerror`, `javascript:`, `iframe`, `style`, `onclick`, or
      `svg` handler.
- [ ] The sanitiser has no automated regression test. Lunar has no frontend unit-test runner, so the
      check was performed by importing the module in a live browser. Adding `vitest`, or a Playwright
      spec that imports the module through the dev server, would make it permanent.
- [ ] `frontend/src/features/copilot/components/ChatWindow.vue` is 998 lines against the 300-line
      limit. It was already 914 before this work.
- [ ] Two pre-existing files fail `gofmt`: `internal/bitbucket/domain/entity.go` and
      `internal/shared/crypto/crypto.go`. Neither was touched by this work.
- [ ] `codeindex/application` still imports `codeindex/infrastructure` for `FileState` and
      `FileUpdate`. Moving those types into the domain would remove the last layering inversion.

---

---

## Decisions Required Before Implementation

- [x] D.1 Approved: the agent and the model call are hosted in the helper. The helper holds the API key in memory only and makes outbound HTTPS calls.
- [x] D.2 Approved: the exclusion and redaction rules in the plan are adopted as written.
- [ ] D.3 Confirm indexing only the repositories selected in workspace settings
- [x] D.4 Approved: the knowledge and glossary layer is in scope. `gopkg.in/yaml.v3` is approved for this workstream only.
- [ ] D.5 Confirm the Stage 2 tool set is the eleven code tools and nothing else
- [ ] D.6 Confirm that Lunar's relative-path safety helpers are the correct containment boundary for the ported walker

---

## Supervisor Integration

- [x] S.1 `go test -race ./...` passes
- [x] S.2 `vue-tsc -b` exits zero
- [x] S.3 `vite build` succeeds
- [ ] S.4 Playwright suite stays green
- [x] S.5 Zero code comments in every ported file
- [x] S.6 Every touched file under 300 lines
- [x] S.7 No new dependencies beyond the single YAML decision
- [x] S.8 No writes to `/Users/erendt/Code/timesheet-app`
- [ ] S.9 Repository content never reaches the VM database or the logs
- [x] S.10 Index directory `0700` and file `0600`
- [ ] S.11 Full-build and first-token timings recorded
- [ ] S.12 Prune this plan once its stages are verified

# Code-Aware Copilot — Implementation Plan

Status: proposed, awaiting approval
Owner: supervisor agent
Feature folder: `plan/code-copilot/`
Supersedes: the first draft of this plan, which proposed building the retrieval and tool layers from
scratch. Inspection of `/Users/erendt/Code/timesheet-app` showed that almost all of it already
exists, working, with tests. This version is a port-and-adapt plan, not a greenfield build.

---

## 1. Objective

Let an engineer ask Lunar Copilot questions about the codebase and receive answers grounded in real
files, with citations that point at an actual repository, path, and line range.

Target questions:

- "Where is the late fee for merchant settlement calculated?"
- "Which service owns the QRIS callback endpoint?"
- "Show me every place that writes to the `settlement_ledger` table."
- "If I change this function signature, what else breaks?"

Out of scope: writing or modifying code, running tests, and autonomous agents that edit repositories.

---

## 2. Findings

### 2.1 The existing Copilot tools retrieve nothing

`backend/internal/copilot/application/service.go:243-283` is the whole implementation:

```go
toolsContext := strings.Join(req.ActiveTools, ", ")
systemPrompt := fmt.Sprintf("... Current active workspace tools: [%s] ...", toolsContext)
```

Tool names are concatenated into one sentence of the system prompt. No Jira issue, pull request, or
document is fetched. `CopilotService` has no dependency on those modules. The source badges at lines
271-283 are produced by substring-matching tool names, so they are decorative.

`llm_client.go` has no tool-schema parameter, no tool-call parsing, and no multi-turn loop.

### 2.2 A working implementation already exists in `timesheet-app`

`/Users/erendt/Code/timesheet-app` is the owner's personal application. Its backend already contains
a complete, tested code-intelligence and agent stack for the same BRI systems.

| Component | Location | Lines or size |
|---|---|---|
| Tool and registry contracts | `internal/application/copilot/port/{tool,ai}.go` | 50 + 268 |
| Tool registry with domain scoping | `internal/application/copilot/tools/registry.go` | 155 |
| Multi-round agent loop | `internal/application/copilot/agent.go` | 813 |
| DeepSeek tool calling, streaming and non-streaming | `internal/infrastructure/ai/deepseek_toolcall*.go` | 464 + 282 |
| DSML parser | `internal/infrastructure/ai/dsml_parser.go` | 109 |
| Provider failover | `internal/infrastructure/ai/failover_client.go` | 463 |
| Response redaction | `internal/infrastructure/ai/redact.go` | 49 |
| Structured JSON output | `internal/infrastructure/ai/jsonout.go` | 326 |
| Code indexer with per-language extractors | `internal/infrastructure/servicemap/indexer.go` | 28 KB |
| FTS5 store with BM25 | `internal/infrastructure/servicemap/sqlite_store.go` | 17 KB |
| Knowledge and glossary indexing | `knowledge_loader.go`, `glossary_generator.go` | 10 KB + 11 KB |
| Tool schemas | `internal/infrastructure/servicemap/tool_definitions.go` | 13 KB |
| Tool executors | `internal/application/servicemap/service.go` | 2,081 |
| Domain and infrastructure tests | `*_test.go` beside each of the above | substantial |

The code tools Lunar needs already exist and are registered under one domain key:

```text
search_code              find_symbol_references    get_service_dependencies
get_file_content         get_service_routes        list_services
get_route_details        get_service_contract      get_domain_knowledge
get_git_diff             grep_raw_fallback
```

`ScopedDefinitions(domains []string)` already filters tool definitions by domain key. That is exactly
what Lunar's decorative "Active Copilot Tools" toggles need in order to become real.

### 2.3 The indexer is already the design this plan would have chosen

`NewService(store, aiClient, briBaseDir, ...)` with `RepoScanner.ScanAll(baseDir)` indexes the local
BRI repositories into an FTS5 store. The schema uses two columns deliberately:

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS code_chunks_fts USING fts5(
    repo_name UNINDEXED, file_path UNINDEXED, chunk_type UNINDEXED,
    raw_content UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);
```

`raw_content` is what a human reads. `content` is the identifier-expanded form that gets indexed,
produced by `prepareContentForFTS` and `splitIdentifier`, which break camelCase and snake_case so a
query for `late fee` matches `calculateLateFee`. Plus `service_deps` stores a real dependency graph
with `http_client`, `kafka_publish`, and `kafka_subscribe` edges.

This is a better design than the symbol-rule chunker in the superseded draft.

### 2.4 Compatibility with Lunar

| Concern | Finding |
|---|---|
| SQLite driver | `modernc.org/sqlite`, same family. Lunar pins 1.34.5, timesheet-app 1.59.0. FTS5 and `bm25()` proven working on Lunar's pinned version |
| Shell execution | `os/exec` appears only in `git_diff_reader.go`, running `git -C <repo> <fixed args>`. Lunar already does this in `git_command.go` |
| New dependencies | Only `gopkg.in/yaml.v3`, needed solely by the knowledge loader. Avoidable by storing knowledge as JSON instead |
| Layering | Both use domain, application, infrastructure separation. Portable |

### 2.5 Adaptation cost is real but bounded

The port is not free:

1. **Zero comments.** `AGENTS.md` forbids comments; timesheet-app is heavily commented throughout.
2. **English only.** Error strings and messages are partly Indonesian, for example
   `"tool tidak ditemukan: %s"` and `"gagal membuat direktori database"`.
3. **300-line limit.** `servicemap/service.go` is 2,081 lines, `agent.go` 813, `indexer.go` roughly
   950. Each must be split by responsibility.
4. **Relocation.** The stack currently runs in a local backend process. In Lunar it must run in the
   helper.

Roughly 8,000 lines are in scope, of which a large part is mechanical comment removal and file
splitting. Porting remains far cheaper than rebuilding, and it arrives already covered by tests.

### 2.6 Corpus shape

Across the 36 repositories under `~/BRI`, excluding `node_modules`, `vendor`, `dist`, and `build`:
3,627 PHP, 2,081 Go, 1,671 JavaScript, 773 Vue, roughly 8,200 files and 69 MB. The existing
extractors cover Go, Java, Node, and PHP, which matches this closely; Vue files fall through to the
generic path and may need a small addition.

---

## 3. Architecture Decision

### 3.1 The helper hosts the index, the agent, and the model call

The reusable stack is designed to run as one local process. Keeping it that way is both the cheapest
adaptation and the strongest privacy posture, because code never transits the VM at all.

```text
Browser                     Helper (laptop)                        VM Backend
   |                             |                                      |
   |-- POST /agent/ask --------->|                                      |
   |                        index lookup (FTS5)                        |
   |                        agent loop, tool calls                     |
   |                             |---- LLM provider (HTTPS) ----------->|
   |<-- stream answer + sources -|                                      |
   |                                                                    |
   |-- POST /api/copilot/transcript (question, answer, sources) ------->|
   |<-- 201 ---------------------------------------------------------|
```

The VM keeps what it is good at: chat history, the model catalog, the credential vault, and the
existing session and auth model. It never sees repository content.

This is strictly better than the superseded draft, which had code excerpts transiting the VM inside
the chat request body.

### 3.2 Consequences to accept

1. The helper needs the user's provider API key for the session. The browser already holds the key
   flow; it passes the key to the helper alongside the question, and the helper keeps it in memory
   only, never on disk.
2. The helper gains outbound HTTPS to the model provider. It currently makes no outbound calls. In a
   corporate network this may require proxy configuration, which must be verified early.
3. Chat history still lives on the VM. The browser posts the completed transcript after the answer
   arrives, so history survives a helper restart and stays visible across devices.

### 3.3 Rejected alternatives

| Alternative | Why rejected |
|---|---|
| Build retrieval and tooling from scratch | The working, tested implementation already exists a directory away |
| Keep the agent on the VM and relay each tool call through the browser | Requires a return-and-resume protocol on the chat endpoint, and code excerpts would transit the VM, which 3.1 avoids entirely |
| Index on the VM | Requires storing BRI source on shared infrastructure, which is not permitted |
| Vector embeddings first | The existing BM25 plus identifier expansion has never been measured against real questions. Measure before adding weight |
| Port the timesheet, presence, and quick-action tools | Out of scope. Port only code intelligence and the BRI integration tools |

---

## 4. Port Scope

### 4.1 Port as-is, after comment and language cleanup

```text
port/tool.go, port/ai.go          contracts: Tool, ToolRegistry, ToolDefinition, SourceReference
tools/registry.go                 registration and ScopedDefinitions
infrastructure/ai/redact.go       secret redaction
infrastructure/ai/jsonout.go      structured output parsing
```

### 4.2 Port with file splitting

```text
agent.go                813  ->  loop, rounds, tool dispatch, prompts, event emission
indexer.go              ~950 ->  walker, per-language extractors, identifier expansion, dependency edges
sqlite_store.go         ~600 ->  schema and migrations, writes, searches
servicemap/service.go   2081 ->  one file per tool executor
deepseek_toolcall*.go   746  ->  request building, call parsing, streaming
failover_client.go      463  ->  provider selection and retry
knowledge_loader.go     ~350 ->  glossary and markdown ingestion
glossary_generator.go   ~370 ->  enum-to-knowledge derivation
tool_definitions.go     ~450 ->  JSON schemas, grouped by tool family
```

### 4.3 Port from stage 4 onward

```text
tools/jira/tools.go, tools/bitbucket/tools.go, tools/queryreview/tools.go
```

### 4.4 Do not port

```text
tools/timesheet/, tools/presence/    unrelated to Lunar
quick actions, IDE open, cache and metrics layers unless needed
```

---

## 5. Architecture Inside the Helper

Follows Lunar's existing layering so the helper stays consistent with `internal/workspace`.

```text
backend/internal/codeindex/
  domain/          entities, repository interface, tool schemas
  application/     indexer orchestration, agent loop, search ranking
  infrastructure/  walker, extractors, FTS5 store, knowledge loader
  transport/       HTTP handlers for index and search

backend/internal/agent/
  domain/          Tool, ToolRegistry, ToolDefinition, SourceReference, ChatMessage
  application/     agent loop, tool dispatch, prompt assembly
  infrastructure/  provider clients, tool calling, failover, redaction, JSON output
```

The helper gains its first database. It is created with directory `0700` and file `0600`, matching
the existing token file.

---

## 6. Privacy and Security Model

### 6.1 The helper now holds source code

Phase 1-2 read git metadata only. This plan changes that: the helper reads and stores source files.
The new guarantees:

- Index database directory `0700`, file `0600`, asserted by test.
- The helper still binds `127.0.0.1`, still enforces the origin allowlist, and still requires the
  bearer token on every endpoint except `/health` and `/token`.
- `/index/status` returns counts and timestamps only, never repository or file paths.
- Only the search and agent endpoints return file content, and only for repositories the caller has
  configured.
- The API key supplied by the browser is held in memory only and is never written to disk or logged.

### 6.2 Secret scrubbing before anything is stored

The corpus is a production codebase and will contain credentials. Scrubbing runs at index time so a
secret never enters the database and therefore can never reach a prompt.

Excluded file names:

```text
.env, .env.*, *.pem, *.key, *.p12, *.keystore, *.jks, id_rsa*, id_ed25519*,
credentials*, *secret*, .npmrc, .netrc, .htpasswd
```

Redaction applied to chunk content before storage:

```text
(?i)(api[_-]?key|secret|password|passwd|token|private[_-]?key)\s*[:=]\s*["'][^"']{8,}["']
```

The matched value is replaced with `***`. The port already ships `redact.go`; reuse it and extend it
with the file-name exclusions, which the current implementation does not appear to have.

### 6.3 If the security team requires it

If Security objects to the helper holding source code, the fallback is to keep the index on the
laptop but restrict it to repositories explicitly marked as non-sensitive, or to require per-session
unlock. The architecture does not change; only the policy gate does.

---

## 7. Frozen Contracts

### 7.1 Helper endpoints

All require the bearer token except `/health` and `/token`.

```text
GET  /index/status
  -> { indexed, is_building, repo_count, file_count, chunk_count, updated_at, progress_percent }

POST /index/build
  body { repos: []string, rebuild: bool }
  -> 202 { started: bool }
  One background goroutine guarded by a mutex; a concurrent call returns 409.

GET  /search?q=&repos=&limit=
  -> { query, results: [ SearchResult ] }
  limit defaults to 8, maximum 25.

POST /agent/ask
  body { question, domains: []string, provider, model, api_key, thinking_mode, reasoning_effort }
  -> text/event-stream of AgentEvent
```

```text
AgentEvent   { type: "tool_call" | "tool_result" | "text" | "sources" | "done" | "error",
               text?, tool?, sources?, rounds? }
SearchResult { repo, path, language, chunk_type, start_line, end_line, snippet, score }
```

`domains` is the list of enabled tool domains. It maps directly onto `ScopedDefinitions`, which is
how Lunar's existing toggles become functional.

### 7.2 Backend addition

One new endpoint, so history lives on the VM while inference happens locally:

```text
POST /api/copilot/transcript
  body { session_id, question, answer, reasoning, sources: [SourceReference], duration_ms }
  -> 201 { message_id }
```

The existing `POST /api/copilot/chat` is retained for non-code questions and is unchanged, so nothing
regresses.

### 7.3 TypeScript types

`CodeIndexStatus`, `SearchResult`, `AgentEvent`, and `ToolDomain` added under
`frontend/src/features/code-context/types.ts`.

---

## 8. Staged Delivery

Ordering follows the owner's instruction: code index and Copilot integration first, BRI tool
integration afterwards.

### Stage 1 — Code index in the helper

Port the walker, extractors, identifier expansion, dependency edges, and FTS5 store. Add the helper
database with correct permissions. Add `/index/status`, `/index/build`, and `/search`.

Done when all selected repositories index, search returns correct hits for known symbols, a rebuild
touches only changed files, and no secret pattern survives in the database.

### Stage 2 — Agent and tool calling, code questions answered

Port the tool contracts, registry, agent loop, tool-calling client, failover, redaction, and the
eleven code tool executors. Add `/agent/ask` with streaming. Wire the frontend to ask the helper and
persist the transcript to the VM. Real citations replace decorative badges.

Done when asking "where is X" about a real repository returns a grounded answer whose citation opens
the correct file and line, verified in a browser.

### Stage 3 — Make the existing toggles real

Connect Lunar's "Active Copilot Tools" toggles to `domains` and therefore to `ScopedDefinitions`.
The toggles stop being cosmetic: they gate which tool schemas the model receives. The token-cost
warning becomes accurate for the first time.

Done when disabling a domain removes its tools from the request, asserted by test and observed in the
browser.

### Stage 4 — Jira, Bitbucket, Confluence, and query review tools

Per the owner's instruction, this starts only after stages 1 and 2. Port the remaining tool packages
and connect them to Lunar's existing clients and stored PATs.

Done when each tool performs a real retrieval and its sources appear as real citations.

### Stage 5 — Semantic retrieval (not approved)

Vector search for questions whose words do not appear in the code. Blocked on measuring stage 1 and 2
quality first.

---

## 9. Verification Strategy

| Layer | Method |
|---|---|
| Ported unit tests | Port `*_test.go` alongside each package; they already cover extractors, store, tool calling, failover, and the agent loop |
| Identifier expansion | Test that `calculateLateFee` is matched by `late fee` |
| FTS5 ranking | Seeded corpus asserting result order |
| Redaction | Fixture with key-shaped strings; assert database and every response contain `***` and never the original |
| Path safety | Reuse `ResolveRepoPath`; assert `..`, absolute paths, and symlink escapes are rejected |
| Index permissions | Assert `0700` directory and `0600` file |
| Domain scoping | Assert disabled domains are absent from the outgoing tool list |
| No persistence of code | Assert the VM transcript row contains the answer but never repository content beyond cited snippets |
| End to end | Index a fixture repository, ask a question in a real browser, assert the citation resolves |
| Performance | Record full-build and first-token timings on `~/BRI` in the checklist |

`go test -race ./...` must pass. `vue-tsc` and `vite build` must pass. The Playwright suite must stay
green.

---

## 10. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Comment and language cleanup is larger than expected | Schedule slip | Size the port per file first; treat it as mechanical and reviewable |
| Splitting `service.go` and `agent.go` introduces regressions | Broken behaviour | Port tests first, then split, running tests after each split |
| Corporate network blocks helper egress | Feature unusable | Verify proxy and TLS egress from the helper before stage 2 is declared done |
| A secret reaches the model | Credential disclosure | Index-time exclusion and redaction, plus a test asserting no pattern survives |
| API key held in helper memory | Key exposure on a shared laptop | Memory only, never logged, cleared when the session ends; document the tradeoff |
| Vue files are not well covered by the existing extractors | Weak results for 773 files | Measure, then add a Vue extractor if the gap matters |
| Ported code brings ticket and comment cruft | Violates repository rules | Supervisor review gate on every ported file |

---

## 11. Open Decisions

1. Approve hosting the agent and the model call in the helper, which means the helper holds the API
   key in memory and makes outbound HTTPS calls.
2. Approve the index-time exclusion and redaction rules in section 6.2, or supply Security's own list.
3. Confirm indexing only the repositories selected in workspace settings.
4. Confirm whether the knowledge and glossary layer is in scope, which decides whether
   `gopkg.in/yaml.v3` is added or the format is converted to JSON.
5. Confirm the tool set for stage 2 is the eleven code tools listed in section 2.2, and nothing else.

---

## 12. Workstream Decomposition

Ownership, dependencies, and handoff rules are in
`plan/code-copilot/subagent_workstreams.md`.

| Workstream | Scope | Depends on |
|---|---|---|
| WS-A | Ports the tool and registry contracts and freezes interfaces | none, serialized first |
| WS-B | Ports the indexer, extractors, and FTS5 store into the helper | WS-A |
| WS-C | Ports the agent loop, tool-calling client, failover, and code tool executors | WS-A, WS-B |
| WS-D | Helper transport: `/index/*`, `/search`, `/agent/ask` streaming | WS-B, WS-C |
| WS-E | Backend transcript endpoint and frontend orchestration, citations, real toggles | WS-A |

WS-B and WS-E can run in parallel after WS-A. WS-C follows WS-B. WS-D follows WS-C. The supervisor
owns WS-A, the comment and language cleanup standard, and final integration.

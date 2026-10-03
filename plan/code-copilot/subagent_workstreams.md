# Code-Aware Copilot — Subagent Workstreams

Supervisor owns WS-A, the porting standard, and final integration. Five workstreams, strict file
ownership, no two agents editing one file.

Source of truth for the port is `/Users/erendt/Code/timesheet-app/backend`. Read from it; never write
to it.

---

## Porting Standard

Applies to every ported file without exception.

1. Zero comments. Strip every line comment, block comment, and doc comment from ported code.
2. English identifiers, strings, and error messages. Translate Indonesian text, for example
   `"tool tidak ditemukan: %s"` becomes `"tool not found: %s"`.
3. Maximum 300 lines per file. Split by responsibility. A file that arrives larger is split before
   review.
4. Follow Lunar's layering: the helper uses `domain`, `application`, `infrastructure`, `transport`.
5. Port the tests with the code. A ported package without its tests is not accepted.
6. No new module dependencies except the single YAML decision recorded in the plan.
7. No shell execution beyond the existing fixed-argument `git` calls already used by Lunar.

---

## Dependency Graph

```text
WS-A  contracts: Tool, ToolRegistry, ToolDefinition, SourceReference, entities   (supervisor)
  |
  +--> WS-B  indexer, extractors, FTS5 store                          
  |      |
  |      +--> WS-C  agent loop, tool calling, failover, code executors
  |             |
  |             +--> WS-D  helper transport: /index/*, /search, /agent/ask
  |
  +--> WS-E  backend transcript endpoint, frontend orchestration, real toggles
```

WS-B and WS-E may run concurrently once WS-A is frozen. WS-C follows WS-B. WS-D follows WS-C.

---

## WS-A — Contracts (supervisor)

Files owned:

```text
backend/internal/agent/domain/tool.go              port of port/tool.go
backend/internal/agent/domain/ai.go                port of port/ai.go
backend/internal/agent/domain/registry.go          registry interface only
backend/internal/codeindex/domain/entity.go        port of domain/servicemap/model.go
backend/internal/codeindex/domain/repository.go    port of domain/servicemap/repository.go
backend/cmd/lunar-helper/index_schema.go           the FTS5 DDL
frontend/src/features/code-context/types.ts        TypeScript mirror of the contracts
```

Deliverables:

1. `Tool`, `ToolRegistry`, `ToolDefinition`, `ToolCall`, `ToolResult`, and `SourceReference`, ported
   with comments removed and Indonesian text translated.
2. Code index entities: chunk, dependency, glossary, status.
3. `CodeIndexRepository` interface covering the write path and the read path, so WS-B and WS-C have a
   frozen seam.
4. The FTS5 DDL from the existing schema, applied idempotently, with `0700` directory and `0600` file.
5. TypeScript types matching 1 and 2.

Definition of done: the project builds, existing tests pass, and no downstream workstream needs to
invent a type or a column.

---

## WS-B — Indexer and store

Files owned:

```text
backend/internal/codeindex/infrastructure/walker.go        port of indexer walk and skip rules
backend/internal/codeindex/infrastructure/extractor_go.go
backend/internal/codeindex/infrastructure/extractor_php.go
backend/internal/codeindex/infrastructure/extractor_node.go
backend/internal/codeindex/infrastructure/extractor_java.go
backend/internal/codeindex/infrastructure/identifier.go    port of prepareContentForFTS and splitIdentifier
backend/internal/codeindex/infrastructure/dependency.go    port of URL and kafka edge extraction
backend/internal/codeindex/infrastructure/store.go         port of sqlite_store.go
backend/internal/codeindex/infrastructure/redactor.go      port and extend infrastructure/ai/redact.go
backend/internal/codeindex/application/indexer.go          orchestration, incremental rebuild
backend/internal/codeindex/infrastructure/*_test.go        ported tests
```

Deliverables:

1. Repository walk reusing Lunar's `ResolveWorkspaceRoot` and sensitive-root rejection, with the
   existing skip-directory and indexable-extension rules.
2. The four language extractors, split one per file to respect the line limit.
3. Identifier expansion so `late fee` matches `calculateLateFee`, ported with its tests.
4. Dependency edge extraction for `http_client`, `kafka_publish`, and `kafka_subscribe`.
5. FTS5 store with `raw_content` unindexed and `content` indexed, WAL mode, and BM25 search.
6. Redaction extended with the file-name exclusion list from the plan, applied before storage.
7. Incremental rebuild keyed by size, modification time, and content hash.

Definition of done: a full build over the selected `~/BRI` repositories completes, a rebuild touches
only changed files, identifier expansion is asserted by test, and no secret pattern survives in the
database.

---

## WS-C — Agent and tool execution

Depends on WS-A and WS-B. Starts after the WS-B store lands.

Files owned:

```text
backend/internal/agent/application/loop.go             port of agent.go, split
backend/internal/agent/application/dispatch.go         port of agent.go tool dispatch, split
backend/internal/agent/application/prompt.go           port of agent prompt assembly, split
backend/internal/agent/infrastructure/toolcall.go      port of deepseek_toolcall.go
backend/internal/agent/infrastructure/toolcall_stream.go
backend/internal/agent/infrastructure/dsml.go          port of dsml_parser.go
backend/internal/agent/infrastructure/failover.go      port of failover_client.go
backend/internal/agent/infrastructure/provider.go      port of transport.go and provider clients
backend/internal/agent/infrastructure/jsonout.go       port of jsonout.go
backend/internal/agent/application/*_test.go           ported tests
backend/internal/agent/infrastructure/*_test.go        ported tests
backend/internal/codeindex/application/search.go       ranking and query sanitization
backend/internal/codeindex/application/tool_search.go  the eleven code tool executors, split per family
backend/internal/codeindex/application/tool_symbol.go
backend/internal/codeindex/application/tool_route.go
backend/internal/codeindex/application/tool_dependency.go
```

Deliverables:

1. Multi-round agent loop with a bounded round count, tool dispatch, and event emission.
2. Tool calling including streaming, with the DSML parser.
3. Provider failover and the structured JSON output helper.
4. Query sanitization converting user input into a safe FTS5 expression with no operator passthrough.
5. Ranking with the existing boosts, asserted by test.
6. The eleven code tool executors, split across files by family, each returning `SourceReference`
   entries that carry repo, path, and line range.

Definition of done: the agent answers a code question against a real index, citations resolve to the
correct file and line, a query containing FTS5 operators is neutralized rather than executed, and
`ScopedDefinitions` demonstrably filters tools.

---

## WS-D — Helper transport

Depends on WS-C.

Files owned:

```text
backend/cmd/lunar-helper/handlers_index.go       GET /index/status, POST /index/build
backend/cmd/lunar-helper/handlers_search.go      GET /search
backend/cmd/lunar-helper/handlers_agent.go       POST /agent/ask with SSE
backend/cmd/lunar-helper/server.go               edit: register the new routes
backend/cmd/lunar-helper/*_test.go               new transport tests
```

Deliverables:

1. The three endpoints behind the existing token middleware.
2. Server-sent events for the agent stream, with a bounded response and explicit flush.
3. One background build goroutine guarded by a mutex, returning `409` when a build is already running.
4. Per-request timeouts and a bound on request body size, matching the existing helper conventions.
5. Tests covering the token requirement, the concurrent-build conflict, and stream framing.

Definition of done: endpoints behave as documented, an unauthenticated request is rejected, and a
streamed answer arrives incrementally in a browser.

---

## WS-E — Backend transcript and frontend

Files owned:

```text
backend/internal/copilot/transport/handler.go               edit: add the transcript endpoint
backend/internal/copilot/application/service.go             edit: save a provided transcript
backend/internal/copilot/domain/entity.go                   edit: ChatSource gains repo, path, line range
frontend/src/features/code-context/api/code-index-api.ts    new
frontend/src/features/code-context/api/agent-stream.ts      new
frontend/src/features/code-context/composables/useCodeAgent.ts
frontend/src/features/copilot/composables/useCopilot.ts     edit
frontend/src/features/copilot/components/ChatWindow.vue     edit
frontend/src/features/copilot/components/CodeCitationList.vue
frontend/src/features/copilot/components/CodeIndexStatus.vue
frontend/src/features/copilot/components/DomainToggleModal.vue  edit: toggles feed domains
```

Deliverables:

1. `POST /api/copilot/transcript` persisting question, answer, reasoning, and real sources.
2. `ChatSource` extended with optional repo, path, start line, and end line.
3. Helper API client reusing the existing helper request plumbing rather than a second fetch layer.
4. `useCodeAgent` orchestrating: fetch the session API key, ask the helper, consume the stream,
   persist the transcript.
5. Citation component rendering repository, path, and line range.
6. Index status with a build action and progress.
7. Existing toggles wired to the `domains` field, replacing the decorative badge construction.
8. Graceful degradation when the helper is offline or the index is missing.

Definition of done: a real question answered in the browser, its citation opens the correct file and
line, disabling a domain removes its tools from the outgoing request, and the flow still works with
the helper stopped.

---

## Handoff Rules

1. Never write to `/Users/erendt/Code/timesheet-app`. It is read-only reference.
2. No workstream edits a file owned by another. Request changes through the supervisor.
3. WS-A completes and is verified before any other workstream starts.
4. Contract changes after WS-A are supervisor-approved and broadcast.
5. Every workstream reports files changed, line counts, the exact commands run, and raw output. A
   claim without command output is not accepted.
6. No commits, no pushes, no new dependencies, no dev servers started by subagents.

## Supervisor Integration Checklist

1. `go test -race ./...` passes.
2. `vue-tsc -b` exits zero and `vite build` succeeds.
3. Playwright suite stays green.
4. Zero code comments in every ported file.
5. Every touched file under 300 lines.
6. `code_context` and repository content never reach the VM database or the logs.
7. Index directory `0700` and file `0600`.
8. Helper egress to the model provider verified on the corporate network.
9. Full-build and first-token timings recorded.
10. Prune this plan once its stages are verified.

# Lunar Backlog

Open items carried forward from completed or deleted plans. Each entry names the evidence and what
deciding it requires.

---

## Blocking a broad rollout

- [ ] **B.1 Set `CORS_ORIGIN` to the real Lunar origin.**
  `backend/.env` currently allows the wildcard origin while `Access-Control-Allow-Credentials` is
  true. This is pre-existing and was deliberately left untouched. It is safe for local work and a
  team demo, and not safe to expose broadly. Requires the deployment origin from the platform team.

---

## Decisions needing an owner

- [ ] **B.2 Two Go entrypoints exist.**
  `backend/main.go` and `backend/cmd/server/main.go` are both tracked and both valid. The CI
  workflow builds the root one while local instructions use `./cmd/server`. One should be removed or
  the relationship documented. Requires a decision on which is canonical.

- [ ] **B.3 Helper distribution for a large audience.**
  The release workflow builds five platform binaries, but GitHub Actions artifacts are only
  reachable by people with repository access. A download page or release asset is needed. Its shape
  depends on BRI's internal distribution channel. Carried forward from the microservices plan.

- [ ] **B.4 Make the Jira, Bitbucket, and Confluence Copilot tools real.**
  Decision taken: they are built, but only after the code index and Copilot integration are
  finished. Tracked as Stage 4 in `plan/code-copilot/`. Today they contribute one sentence of prompt
  text and a decorative badge, and retrieve nothing. Working implementations for Jira, Bitbucket, and
  query review already exist in `/Users/erendt/Code/timesheet-app` and will be ported. Confluence has
  no reference implementation and must be written.

---

## Technical debt

- [ ] **B.5 `AppSidebar.vue` is 320 lines.**
  Above the 300-line limit. It was already over before the current work and was reduced from 361.
  Needs a split by responsibility.

- [ ] **B.6 `AiProvidersSettings.vue` is 493 lines.**
  Above the 300-line limit, pre-existing. The natural split is one component per provider.

- [ ] **B.7 `DocumentActionsRow.vue` contains toast-only stubs.**
  Actions such as "Extract query review" and "Instant apply" show a message saying the feature is
  ready for backend wiring, and do nothing. Either wire them or remove them so the UI does not
  promise a capability that does not exist.

- [ ] **B.8 `enabled` field in the Copilot `defaultDomains` is dead data.**
  It is never read anywhere. It should either be consumed by `loadEnabledDomains()` or removed, so
  setting it to `false` cannot silently do nothing.

- [ ] **B.9 The Jira seed fallback is disabled but still present.**
  `fallbackEnabled` is now `false`, which makes the fallback path unreachable. That leaves three
  things dead: the 13 `if c.fallbackEnabled` branches, the `seedIssues` and `seedSprints` fields, and
  the helpers `buildSeedIssues`, `buildSeedSprints`, `findSeedIssue`, `filterMySeedIssues`, and
  `SetFallbackEnabled`, all in `backend/internal/jira/infrastructure/`. `seed.go` alone is 13 KB of
  fabricated BRI tickets. It must not be able to resurface, so either delete it and the branches, or
  turn the fallback into an explicit opt-in whose responses carry a visible marker. Deleting is
  preferred.

- [ ] **B.10 The tool-call trail is not kept after an answer completes.**
  The trail renders while streaming, then disappears when the answer is persisted, because
  `ChatMessage` has no `toolCalls` field and `copilot_messages` has no column for it. The helper
  already emits every `tool_call` with its round, so the data exists; persisting it needs a column, a
  migration, and a field on the transcript contract.

- [x] **B.11 `ChatWindow.vue` is 702 lines.** Closed on 2026-10-05.
  Reduced from 999 by extracting `ChatMessage`, `ChatStreamingMessage`, `ChatReasoningPanel`,
  `ChatToolCalls`, `ChatStreamingIndicator`, and `ChatMessageActions`. The composer is now
  `ChatComposer.vue` with its model menu in `ModelPicker.vue`, so `ChatWindow.vue` is 196 lines.

- [ ] **B.12 The reasoning panel depends on the model returning reasoning text.**
  When the provider returns none, the panel does not render. That is correct behaviour, but there is
  no visible indication that thinking mode produced nothing, so a user cannot tell the difference
  between "no reasoning" and "reasoning not displayed".

- [x] **B.13 Diagram navigation did nothing.** FIXED.
  Two separate causes. The central gallery was recreated when the streaming message was replaced by
  the persisted message, resetting its index. Then, after diagrams moved in place, the element
  registry was keyed by the segment index while the lookup used the diagram index, so only the first
  card matched, and only by coincidence. Both are resolved: diagrams render in place, and the
  registry is keyed by the diagram index. Verified per card: next moves 1 to 2 and 2 to 3, previous
  moves 3 to 2 and 2 to 1, and the end buttons are disabled.

- [ ] **B.14 Diagram rendering depends on what the model emits.**
  `deepseek-chat` follows the strengthened prompt and emits `mermaid`; `deepseek-reasoner`, which is
  the model the app selects for thinking mode, emits `flow` and is converted instead. A model that
  emits neither produces no diagram. This is a prompt-adherence limit, not a renderer limit.

- [ ] **B.15 Three Copilot e2e tests mock an endpoint the app no longer calls.**
  `chat-conversation.spec.ts` and `chat-layout-visual.spec.ts` intercept `/api/copilot/chat`, but the
  Copilot answer path moved to `/agent/ask` through the local helper. The mock therefore never
  applies, the real helper answers, and the assertions fail on `.reasoning-panel` not existing.
  Two follow-on faults were found and fixed while investigating: the specs also referenced four
  class names that never existed in the current components (`.thought-section`, `.thought-toggle`,
  `.thought-content`, `.thought-title`, now `.reasoning-panel`, `.reasoning-toggle`,
  `.reasoning-body`, `.reasoning-title`). The remaining fix belongs in test infrastructure: abort the
  helper connection in the route mock so the mocked backend path is deterministic, or point the mock
  at the helper.

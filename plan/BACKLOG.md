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

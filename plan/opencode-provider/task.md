# OpenCode Go Provider - Task Checklist

`[x]` done and verified, `[~]` in progress, `[ ]` not started, `[!]` blocked.

## Gate

- [x] Q1: Claude dropped. It is absent from Go and no Anthropic transport is built
- [x] Q2: ids confirmed, `mimo-v2.6-flash` and `space-bunny-free`
- [x] Q3: Copilot-only scope
- [x] Scope: three models, all on `/chat/completions`
- [!] Q4: tool-calling support NOT yet confirmed against the live endpoint. No API key was available
      in this session, so this remains the one open item before either model is called working

## W1 - Backend

- [x] `opencode-go` -> `https://opencode.ai/zen/go/v1` in `provider_urls.go`
- [x] exact-match resolution before the substring matcher in `llm_client.go`
- [x] `opencode-go` case in `ChatCompletion`
- [x] `User-Agent: lunar/1.0` on outbound provider requests
- [x] `x-opencode-session` carries the conversation id on the code agent path
- [x] catalogue entries in `catalog.go`, iterating the single domain table
- [x] `domain/opencode_go.go` is the single source of truth for the three models
- [x] optional `SYSTEM_OPENCODE_GO_KEY`, and `/api/config` lists it for the settings badge
- [x] unit test: base path is `/zen/go/v1` and never `/zen/v1`
- [x] unit test: each catalogue id resolves to `opencode-go` plus the right upstream id
- [x] unit test: a regression guard keeps `DeepSeek-V4 Pro (Thinking)` on the deepseek provider
- [x] unit test: a Go model never receives `thinking`, `reasoning_effort`, or `output_config`
- [x] unit test: User-Agent present, `x-opencode-session` only for opencode-go with a non-empty id
- [x] both critical tests proven non-vacuous by mutation, then reverted
- [x] `llm_client.go` brought back under 300 lines by extracting the Claude transport

## W2 - Frontend

- [x] `types.ts` provider union includes `"opencode-go"`
- [x] `models.ts` entries for the three confirmed models
- [x] settings row with key field, configured badge, system fallback badge
- [x] `AiProvidersSettings.vue` refactored onto a reusable `AiProviderRow`, 493 -> 170 lines
- [x] existing selectors and input ids preserved, `opencode-go-key` added
- [x] `session_id` forwarded on the code agent request
- [x] image capability list unchanged, none of the three is a vision model

## W3 - Capabilities and docs

- [x] decision recorded: Go models do not request reasoning, thinking is gated to deepseek
- [x] `guide.md` provider table, Copilot provider row, and change log
- [x] `backend/.env.example` carries `SYSTEM_OPENCODE_GO_KEY`

## Verification

- [x] `go build ./...` passes
- [x] `go test` passes for copilot, agent, lunar-helper, and app packages
- [x] `vue-tsc` and `vite build` pass
- [x] Playwright settings suite passes, including the new SETTINGS-007 and the OpenCode Go subtitles
- [x] Playwright Copilot suite passes, including the new CP-MODEL-002 and the updated catalogue count
- [x] the provider locator no longer matches two rows
- [ ] a real answer verified per model against the live endpoint (needs the user's API key)

# OpenCode Go Provider

## Correction notice

The first revision of this plan was written against **OpenCode Zen** (`https://opencode.ai/zen/v1`).
That was the wrong product. The subscription in use is **OpenCode Go**, which is a different
endpoint, a different model set, and a different billing model:

| | Zen | **Go (in use)** |
|---|---|---|
| Base path | `https://opencode.ai/zen/v1` | `https://opencode.ai/zen/go/v1` |
| Billing | pay per request, prepaid credits | subscription, monthly usage limits |
| Catalogue size | 86 models | 43 models |
| Proprietary models | yes, including Claude | **no, open models only** |

Verified live:

```text
GET  https://opencode.ai/zen/go/v1/models              -> 200, 43 models, public
POST https://opencode.ai/zen/go/v1/chat/completions    -> 401 AuthError (endpoint valid)
GET  https://opencode.ai/zen/v1/models                 -> 200, 86 models, public
```

## Objective

Add OpenCode Go as an AI provider so its models appear in the Copilot picker, carry a stored API
key, and answer through the existing agent and chat paths.

## Availability of the requested models

Checked against the live catalogues, not the prose:

| Requested | On **Go** (subscribed) | On Zen |
|---|---|---|
| `deepseek-v4.1-flash` | **present** | present |
| `claude-sonnet-5-5` | **absent** | present |
| `claude-opus-5-5` | **absent** | present |
| `mimo-v2.6-flash-free` | absent as written; Go has `mimo-v2.6-flash` | present |
| `danspace-bunny-free` | absent; the id is `space-bunny-free` | present as `space-bunny-free` |

So one of the five works on Go exactly as named. Two need an id correction, and the two Claude
models are not obtainable through Go at all.

Go is explicitly "reliable access to popular open coding models". Its 43 models are Grok, GPT Luna,
GLM, Kimi, LongCat, DeepSeek, MiMo, MiniMax, Muse Spark, Qwen, Hy, and Space Bunny. There is no
Claude, and there is no Anthropic model of any kind in the Go catalogue.

Useful Go models that were not requested but are relevant:

- `deepseek-v4-flash-vision-exp` is a vision model, which the attachment feature can use.
- `deepseek-v4-pro`, `deepseek-v4-flash`, `deepseek-flash` overlap with models already in the
  catalogue under the `deepseek` provider. Namespacing is what keeps the two apart.

## Findings that change the implementation

### The base path is `/zen/go/v1`, not `/zen/v1`

This is not a config detail. Pointing at the Zen path with a Go subscription would bill credits
instead of the subscription and would expose the wrong catalogue.

### Go requires client identification, and we currently fail it

From the Go documentation, verbatim requirements on clients:

- "Send typical coding agent traffic."
- "Identify itself with its own user agent, such as `my-coding-agent/1.0`, rather than a generic
  SDK or HTTP-library name."
- "Send a stable session ID in `x-opencode-session` for each conversation so we can optimize
  routing and prompt caching."
- "Traffic is monitored for abuse that degrades the experience for other users."

Lunar today sends no explicit `User-Agent`, so Go receives the default `Go-http-client/1.1`,
which is exactly the generic name the policy warns about. The provider integration must therefore:

1. Set a stable, identifying `User-Agent` such as `lunar/`.
2. Send `x-opencode-session` with a stable per-conversation id.

Lunar already has a conversation id on every request (`session_id`), so the header is a plumbing
change rather than new state.

### Provider resolution is substring matching and will misroute these models

`copilot/infrastructure/llm_client.go:37`:

```go
case strings.Contains(lower, "deepseek"):
    provider = "deepseek"
```

`deepseek-v4.1-flash` contains "deepseek", so as written it resolves to the personal DeepSeek
provider and is rewritten to `deepseek-chat`. Any new provider must match **exactly, before** the
substring fallback. The same matcher is duplicated for the agent in
`agent/infrastructure/toolcall.go:26`.

### Thinking parameters are already gated away from a new provider

`agent/infrastructure/provider.go` has `supportsReasoningEffort()` true only for
`provider == "deepseek"`. `toolcall.go:80` then sends `thinking` / `reasoning_effort` /
`output_config`, or falls back to `temperature = 0.1`. A new provider inherits the safe branch, so
no DeepSeek-specific field leaks to Go. The cost is that Go models will not request reasoning even
though thinking mode is permanently on in the UI.

### API keys need no schema change

`auth/domain/entity.go` stores keys as `map[string]string` keyed by provider, encrypted as one
JSON blob. `opencode-go` is a new map key, not a migration.

## Confirmed scope

Decisions taken by the user:

| Question | Decision |
|---|---|
| Claude models | Confirmed absent from Go. Dropped from this change. No Anthropic transport is built |
| `mimo-v2.6-flash-free` | Corrected to `mimo-v2.6-flash` |
| `danspace-bunny-free` | Corrected to `space-bunny-free` |
| Scope | Copilot only. The microservices exporter keeps its own list |

The change therefore ships exactly three models, all on the OpenAI-compatible path:

| Model | Upstream id | Path |
|---|---|---|
| DeepSeek V4.1 Flash (Go) | `deepseek-v4.1-flash` | `/chat/completions` |
| MiMo V2.6 Flash (Go) | `mimo-v2.6-flash` | `/chat/completions` |
| Space Bunny Free (Go) | `space-bunny-free` | `/chat/completions` |

Because every one of them speaks OpenAI chat completions, **no new transport is required**. The
existing agent tool-calling loop and the existing chat client both work unchanged against Go, and
the work reduces to provider identity, resolution, headers, catalogue and UI.

## Open items

### O1. Does one API key cover both Go and Zen

Not needed for this change, because only Go is called. Recorded for completeness.

### O2. Tool calling on the free and preview models

`space-bunny-free` is a free model and `mimo-v2.6-flash` is a small one. The Go documentation does
not state per-model tool-calling support, and the `/models` catalogue carries no capability
metadata. The agent loop depends on tool calling, so this must be verified with the real key before
it can be called working. If a model does not support tools, it is still usable through the chat
fallback and must be labelled accordingly rather than silently returning an empty answer.

## Design

### Provider identity

```text
provider id   "opencode-go"
base url      "https://opencode.ai/zen/go/v1"
config id     "opencode-go/<model-id>"     matches the documented OpenCode convention
```

### Model entry

```text
id          "opencode-go-deepseek-v4.1-flash"     stable key used by the picker
name        "DeepSeek V4.1 Flash (Go)"            shown in the UI
upstreamID  "deepseek-v4.1-flash"                 sent to Go
protocol    "openai-chat"
```

Rules:

1. Resolution matches the catalogue id or name exactly, before any substring rule.
2. The upstream id is never derived from the display name by string surgery.
3. The protocol is a property of the model entry, not an assumption at the call site.

### Required request headers

Applied in the shared client so both the chat path and the agent path get them:

```text
User-Agent:            lunar/<version>
x-opencode-session:    <conversation/session id>
Authorization:         Bearer <api key>
```

The session id already exists on the request as `session_id`; it must be threaded to the HTTP
layer, which today only receives messages and a model.

## Files

| File | Change |
|---|---|
| `backend/internal/copilot/infrastructure/provider_urls.go` | `opencode-go` -> `https://opencode.ai/zen/go/v1` |
| `backend/internal/copilot/infrastructure/llm_client.go` | exact-match resolution before substring; `opencode-go` case in `ChatCompletion`; User-Agent |
| `backend/internal/copilot/application/catalog.go` | catalogue entries |
| `backend/internal/copilot/application/provider_key.go` | carry `protocol`; confirm the session id reaches the helper |
| `backend/internal/agent/infrastructure/provider.go` | `User-Agent` and `x-opencode-session` on outbound requests |
| `backend/internal/agent/infrastructure/toolcall.go` | exact-match `resolveModel` for catalogue ids |
| `backend/internal/config/config.go`, `backend/internal/app/app.go` | optional `SYSTEM_OPENCODE_GO_KEY` |
| `frontend/src/features/copilot/utils/models.ts` | the entries the picker renders |
| `frontend/src/features/copilot/types.ts` | `provider` union gains `"opencode-go"` |
| `frontend/src/features/settings/components/AiProvidersSettings.vue` | provider row and key field |
| `frontend/src/features/copilot/utils/attachment-rules.ts` | `deepseek-v4-flash-vision-exp` reads images |
| `guide.md` | feature table and change log |

## Stages

1. **Answer Q1-Q3.** No code first.
2. **Backend identity.** Base path, exact-match resolution, protocol field, User-Agent and session
   header, config key.
3. **Frontend.** Picker entries, provider union, settings row.
4. **Capabilities.** Image support; decide whether Go models request reasoning.
5. **Docs and evidence.**

## Verification

- `GET /api/copilot/models` lists the Go entries and marks them configured only once a key exists.
- One real question per model returns an answer, and the request is confirmed to have gone to
  `/zen/go/v1`, not `/zen/v1`.
- The outbound request carries `lunar/<version>` and `x-opencode-session`.
- The agent path is exercised, not only the chat path.
- A wrong key gives a readable error, not a raw 401 body.
- An unsupported id still 400s instead of being forwarded.
- `go build ./...`, `go test ./...`, `vue-tsc`, and the Playwright Copilot suite stay green.

## Risks

| Risk | Mitigation |
|---|---|
| Pointing at `/zen/v1` with a Go subscription | base path asserted in a unit test per provider |
| A generic User-Agent causes Go to treat traffic as abuse | set `lunar/<version>` and a session header |
| Substring resolution silently sends a Go model to DeepSeek | exact match first, tested per catalogue id |
| The user expects Claude and it does not exist on Go | surfaced in this plan before any code |
| Free or preview models are withdrawn | `longcat-2.5-preview-free` and `space-bunny-free` are marked limited time |
| `temperature = 0.1` rejected by a reasoning model | drop the field if a 400 names it |

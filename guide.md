# Lunar — Feature Guide

The authoritative inventory of every feature and subfeature in this application.

---

## How to keep this file current

This file is part of the definition of done, not optional documentation.

**When you create, change, or remove a feature, a subfeature, or a flow, update this file in the same
change.** That includes:

- A new page, route, or navigation entry.
- A new endpoint, or a changed request or response shape.
- A new background capability, such as indexing or an agent tool.
- A new user-facing control that changes what an existing feature can do.
- Removing or disabling any of the above.

An update means editing the relevant section **and** adding a line to the change log at the bottom.
A feature that exists in code but not here is considered undocumented, and an entry here that no
longer exists in code is considered a defect.

---

## How to run it

| Service | Port | Command | Purpose |
|---|---|---|---|
| Frontend | 5174 | `cd frontend && pnpm dev` | Vue application |
| Backend | 8081 | `cd backend && go run ./cmd/server` | API, credential vault, chat history |
| Helper | 5199 | `cd backend && go run ./cmd/lunar-helper` | Local repository access and the code index |

The helper is optional. Without it, every feature works except the ones that read repositories from
the local machine, and the code-aware Copilot.

Environment files: `backend/.env` and `backend/.env.example`.

---

## Architecture at a glance

```text
Browser  ──> Backend (8081)  ──> Jira, Bitbucket, Confluence, AI providers
   │              │
   │              └── SQLite: users, secrets vault, chat history, workspace settings
   │
   └──────> Helper (5199, loopback only)  ──> Local git repositories, code index, agent
```

Three properties worth knowing before changing anything:

1. **The helper is loopback only and origin allowlisted.** Only `localhost:5174` and
   `127.0.0.1:5174` are accepted. It holds source code, so it is not reachable from a network,
   including through a port forward.
2. **Credentials live in an encrypted vault on the backend.** Jira, Bitbucket, Confluence, and AI
   provider keys are written to the vault and never returned in plain text to the browser, except
   through the dedicated provider-key endpoint used by the local agent.
3. **The code index never leaves the machine.** Indexing, retrieval, and the agent loop all run in
   the helper, so repository content is not stored on the backend.

---

## 1. Authentication

| Subfeature | Detail |
|---|---|
| Sign in | Email and password against the local user store |
| Register | Creates an account by verifying a Jira personal access token, then setting a password |
| Jira identity verification | `POST /api/auth/verify-jira-pat` confirms the token and returns the profile |
| Duplicate identity guard | Rejects registration when the Jira identity already exists |
| Session | Bearer token held by the frontend and sent with every API call |
| Sign out | Clears session state and resets feature stores |
| Theme preference | Dark and light mode, remembered locally |

Routes `/forgot-password` and `/request-access` are redirects to the sign-in and register pages.
They are not features.

Source: `frontend/src/features/auth/`, `backend/internal/auth/`.

---

## 2. Workspace Overview

The landing page after sign-in.

| Subfeature | Detail |
|---|---|
| Metric cards | Jira assigned tickets, Bitbucket pull requests, technical documents, AI providers configured |
| Sync workspace | Refreshes the summary and records the last synchronised time |
| Quick access tiles | Shortcuts to Copilot, Jira, Bitbucket, and Settings |
| Integration credentials | Per-system state, showing configured or needs a token |
| Partial failure handling | One system failing does not blank the others; each card reports its own state |

`GET /api/dashboard/summary`.

Source: `frontend/src/features/dashboard/`, `backend/internal/dashboard/`.

---

## 3. Jira BRI

| Subfeature | Detail |
|---|---|
| Assigned to me | Issues assigned to the signed-in user |
| Backlog | Issues in the active sprint, as a kanban board |
| Kanban board | Columns by status, five cards per column with load more |
| Issue detail | Modal with issue fields and remote links |
| Issue type filters | All, DEV documents, bug or defect, subtasks |
| Document filter | Technical documents derived from issues |
| Refresh | Re-fetches issues on demand |
| Loading and cache | Skeleton loaders on first load; returning to the page keeps the board without a skeleton |
| Connectivity failure | Reports `502` with "check your VPN connection". No data is shown when BRI Jira is unreachable |

When BRI Jira cannot be reached, this page shows an error with a retry action. It must never show
fabricated data. The client previously fell back to hardcoded sample issues, which made an offline
machine look as though it had live tickets, with no indication that the data was invented. That
fallback is disabled and must stay disabled.

`GET /api/jira/issues`, `/api/jira/backlog`, `/api/jira/issues/{key}`, `/api/jira/issues/remotelinks`.

Source: `frontend/src/features/jira/`, `backend/internal/jira/`.

---

## 4. Bitbucket BRI

| Subfeature | Detail |
|---|---|
| Push activity | Recent pushes, filterable by all, ready, and stale |
| Pull requests | List with filters for all, assigned to me, and AI flagged |
| Create pull request | Modal that opens a pull request from a source branch |
| Review modal | Pull request detail with repo, branch, author, and file counts |
| Diff viewer | Toggleable diff for the pull request, with a fallback when the diff is large |
| AI code review | Generates a review with the configured provider, streamed into the comment box |
| Approve and comment | Status actions with confirmation toasts |
| Error handling | Distinct messaging for an invalid token, an unreachable host, and a forbidden response |

`GET /api/bitbucket/pushes`, `/api/bitbucket/prs`, `/api/bitbucket/prs/{id}/diff`,
`POST /api/bitbucket/prs`, `/api/bitbucket/prs/{id}/ai-review`, `/api/bitbucket/prs/{id}/action`,
`/api/bitbucket/prs/{id}/comments`, `/api/bitbucket/reset`.

Source: `frontend/src/features/bitbucket/`, `backend/internal/bitbucket/`.

---

## 5. Confluence BRI

| Subfeature | Detail |
|---|---|
| Document list | All technical documents, with a stats grid by type |
| Type filters | All, UT, query review, SOP |
| Search | Free text search across documents |
| Document detail | Full body, metadata, status, and labels |
| Comments | Per-document comment thread |
| AI panel | Per-document AI assistance |
| Document actions | Row actions such as opening a document |
| Error banners | Dedicated retry banners for authentication and connectivity failures |

Confluence page bodies are converted from Confluence storage format to HTML **and escaped** on the
backend, which is why the detail view can render them safely. Do not remove that escaping.

`GET /api/confluence/documents`, `/api/confluence/documents/{id}`.

Source: `frontend/src/features/confluence/`, `backend/internal/confluence/`.

---

## 6. Microservices BRI

Repository intelligence for the services a squad owns.

| Subfeature | Detail |
|---|---|
| Service table | Services derived from the repositories in the workspace, with search and filters |
| Service detail panel | Branch, last commit, changed files, and file list for a service |
| Repository picker | Modal to choose which repositories are displayed, with name search and checkboxes |
| Model picker | Chooses the AI model used for service analysis |
| Workspace state panel | Explains what is missing when nothing can be shown |
| Helper connection fields | Helper URL, token, connection test, and one-click connect |
| Helper setup guide | Instructions shown when the helper is not running |
| Health and version | Reports whether the helper is online and which version it runs |
| Metadata only | Reads git metadata: branch, changed files, last commit. Not file contents |

Source: `frontend/src/features/microservices/`.

---

## 7. Copilot

The AI assistant. It has two answer paths, and knowing which one ran explains what a given answer
contains.

### 7.1 Chat

| Subfeature | Detail |
|---|---|
| Conversation context | Each question carries the previous turns to the model, so a follow-up such as "and where is it used?" resolves against what was already discussed. The last ten turns are sent, and the agent truncates a long assistant answer rather than rejecting it |
| Sessions | Conversation history, created on the first message. Deleting a conversation asks for confirmation first, and the dialog can be dismissed with Cancel, Escape, or a backdrop click |
| New chat | The sidebar button, the `+` button in the slim header, and the `Cmd+Shift+O` / `Ctrl+Shift+O` shortcut |
| Streaming answers | Text arrives progressively, with a spinner pill and elapsed time |
| Answer layout | An avatar per role. The user message is a right-aligned bubble, the assistant answer is flat beside its avatar |
| Reasoning panel | A collapsible dashed panel with a duration, shown when the model returns reasoning |
| Tool trail | The tools the agent called, with the action and the round, expanded while running |
| Message actions | Copy the answer, and regenerate it |
| Copy controls | A copy button on every rendered code block and table |
| Code mockup | Code blocks render as a window mockup with three dots, the language label, and line numbers. Syntax is highlighted with highlight.js, tree-shaken to the languages the repositories use |
| Highlighted lines | A fence meta such as a language followed by `{3,5-7}` highlights exactly those lines, so an answer can point at the line that carries the behaviour |
| Markdown throttle | The answer re-renders at most once per animation frame while streaming, so a fast token stream does not re-parse on every token |
| Diagrams | Fenced `mermaid` blocks render as diagrams through Mermaid, loaded lazily. Arrow-notation blocks fenced as `flow` are converted to a top-down `flowchart TD` |
| Diagram placement | Each diagram renders in place, under the heading the model wrote it beneath. Diagrams are never collected at the end of the answer |
| Diagram tag | Every diagram card carries a name. The model supplies it as a mermaid frontmatter title; when absent, it is derived from the diagram type and its first participant or node |
| Diagram navigation | When an answer has more than one diagram, each card shows its position and previous and next arrows that scroll to the adjacent diagram |
| Diagram security | Mermaid runs with `securityLevel: strict` and its output is additionally sanitized with DOMPurify before injection. `script`, `foreignObject`, `iframe`, and event attributes are removed |
| Thinking mode | Always on. There is no control to switch it off, and `thinking_mode` is sent as true on every question |
| Providers | Six are selectable: OpenAI, Anthropic Claude, Google Gemini, DeepSeek, Xiaomi MiMo, and OpenCode Go. OpenCode Go is a subscription rather than a pay-per-request key, and it serves only open models, so it offers no Claude model whatever |
| Model and effort picker | A dropup that opens on the model trigger. Its first level offers only Model and Effort. The Model level adds a search field and a back control, filters the configured models live as a name is typed, and returns to the first level once a model is chosen. The Effort level lists Low, Med, and High with the same behaviour. `Escape` closes the menu from either level |
| Reasoning effort | Low, Med, or High, chosen from the picker's Effort level and persisted across reloads |
| Reasoning trace | Collapsible thinking output per answer |
| Composer | Its own component. `Enter` sends and `Shift + Enter` inserts a newline, with no hint text under the composer. Long input wraps to a new line and the field grows with the text, up to four tenths of the viewport height, then scrolls. The focused field draws no outline; focus is shown by the container border instead |
| Contextual header | The Copilot page replaces the 64px workspace topbar with a slim 48px header holding a breadcrumb, a single new chat button, and the code index chip centred between them, with the mobile menu trigger at or below 768px |
| Code index status | A chip centred in the slim header, with the same ready, building, or missing panel as before. It is no longer part of the composer area |
| Transcript persistence | Answers are saved to the backend so history survives a helper restart |

### 7.2 Code-aware answers

When the local helper is connected and the index is ready, questions about code are answered by an
agent running on the local machine.

| Subfeature | Detail |
|---|---|
| Code index | Repositories are walked, chunked by language, redacted, and stored in local FTS5 |
| Incremental rebuild | Only changed files are re-chunked, keyed by content hash |
| Agent loop | Multi-round: the model calls tools, reads results, and continues until it can answer |
| Code tools | `search_code`, `get_file_content`, `find_symbol_references`, `grep_raw_fallback`, `list_services`, `get_service_routes`, `get_service_dependencies`, `get_service_contract`, `get_route_details`, `get_domain_knowledge`, `get_git_diff` |

| Domain knowledge | A curated glossary and Markdown notes, indexed locally and searchable by the agent |
| Index status panel | Ready, building, or missing, with a build action and progress |
| Graceful degradation | With no helper or no index, the message is still answered, without code context, and the UI says why |
| Secret redaction | Credential-shaped values are redacted before storage; sensitive file names are skipped |

Agent behaviour that matters when changing it:

- Multi-word queries are matched with AND, so a natural-language question can retrieve nothing. The
  agent is expected to decompose questions and to fall back to `grep_raw_fallback`.
- Tool-call markup emitted by some models as text (DSML) is stripped before it can reach the answer
  or the message history.
- Only the `servicemap` domain has implemented tools. Selecting the Jira, Bitbucket, or Confluence
  domains changes which tool schemas the model is offered, but those tools are not implemented yet.

### 7.3 Tool domains

The "Active Copilot Tools" modal selects which tool domains are offered to the model. The selection
is sent to the helper and gates the tool schemas, so more tools means a longer prompt and more
tokens.

Source: `frontend/src/features/copilot/`, `frontend/src/features/code-context/`,
`backend/internal/copilot/`, `backend/internal/agent/`, `backend/internal/codeindex/`.

### 7.4 Attachments

A question can carry an attachment. The backend stores it, and the agent answer path decides how it
reaches the model. The composer control is not built yet; the endpoint and the agent wiring are.

| Subfeature | Detail |
|---|---|
| Upload endpoint | `POST /api/copilot/attachments`, `multipart/form-data` with a `file` field of at most 10 MB. Returns `201` with `{ id, name, kind, size, preview }`, or `400` with `{ error }` |
| Accepted types | `image/png`, `image/jpeg`, `image/webp`, `application/pdf`, `.xlsx`, `.xls`, `.docx`. A rejected upload names all of them |
| Image detection | An image is accepted by its content, not its name or its declared type, matching how the provider detects the format. Text content wearing a `.png` name is rejected |
| Document detection | A document is accepted by its extension. There is no content sniffing for documents |
| Size limit | The request body is capped with `http.MaxBytesReader` and the file with a `10 MB` limit. An oversized upload returns `400` without the file being read in full |
| Storage | SQLite, in `copilot_attachments`, scoped by user. A load by id is scoped to the owner, so one user cannot read another user's attachment. The newest 50 attachments per user are kept |
| Path safety | An id is a generated UUID and is never turned into a filesystem path. A client-supplied name is stripped of directories and control characters and is only ever displayed |
| Load endpoint | `GET /api/copilot/attachments/{id}` returns the stored text for a document and a base64 `data:` URL for an image. It exists so the helper can resolve an id passed on the question |
| PDF text | Extracted with `github.com/ledongthuc/pdf` |
| Excel text | `.xlsx` with `github.com/xuri/excelize/v2`, `.xls` with `github.com/extrame/xls`. Sheets and rows are labelled and separated |
| Word text | `.docx` is read as a ZIP archive and its `word/document.xml` text runs are collected, paragraph by paragraph |
| Text cap | Extracted text is capped at 60000 characters, with a truncation marker, so a large document cannot exhaust the context window |
| Untrusted text | Extracted text is prepended to the question as a delimited `ATTACHMENT START`/`ATTACHMENT END` block, in the user message only, never as a system message. Markers inside the document are neutralized so the block cannot be closed early |
| Images | An image is sent inline as a base64 `data:` URL in a content-parts array, and only when the selected model accepts images. Only `deepseek-flash`, which the catalog exposes as "DeepSeek Flash", reads images among the DeepSeek models |
| Model that cannot read images | The request is rejected with a `400` that names the selected model rather than silently dropping the image |
| Attachment limit | At most 5 attachments per question |

Source: `backend/internal/attachment/`, `backend/internal/agent/domain/ai.go`,
`backend/internal/agent/application/prompt.go`, `backend/cmd/lunar-helper/attachments.go`.

---

## 8. Settings

| Tab | Subfeature | Detail |
|---|---|---|
| Integrations | Atlassian tokens | Jira, Bitbucket, and Confluence personal access tokens and usernames |
| Integrations | AI provider keys | Per-provider API keys with configured state |
| Workspace | Repository source | **On this computer** (local helper) or **On the server** (Lunar backend). The labels name where the folder lives, not which component reads it |
| Workspace | Repository root | The folder holding the repositories, with validation |
| Workspace | Helper connection | URL, token, connection test, and one-click connect |
| Workspace | Repository selection | Which repositories are displayed |
| AI providers | OpenAI | Key entry, per-provider model list, configured state |
| AI providers | Anthropic Claude | As above |
| AI providers | Google Gemini | As above |
| AI providers | DeepSeek | As above, including system fallback |
| AI providers | Xiaomi MiMo | As above |
| AI providers | OpenCode Go | Key entry, the three models the Go subscription serves, configured state, and system fallback through `SYSTEM_OPENCODE_GO_KEY` |
| Security | Encryption status | Confirms keys are encrypted at rest with AES-256-GCM |

`GET` and `PUT /api/auth/secrets`, `GET` and `PUT /api/workspace`,
`PUT /api/workspace/selections`.

Source: `frontend/src/features/settings/`, `backend/internal/workspace/`.

---

## 9. Local Helper

A separate binary that runs on the engineer's machine. It is not part of the backend.

| Subfeature | Detail |
|---|---|
| Loopback binding | Listens on `127.0.0.1` only |
| Origin allowlist | Accepts only `localhost:5174` and `127.0.0.1:5174` |
| Token authentication | Generated on first run, stored `0600` in a `0700` directory |
| Token reveal | `GET /token` lets the browser connect in one click, without copying a token by hand |
| Repository listing | `GET /repositories` returns repository metadata for a root |
| Service detail | `GET /services` returns branch, commit, and changed files for named repositories |
| Path validation | `POST /validate-path` confirms a folder is readable and inside an allowed root |
| Code index | `GET /index/status`, `POST /index/build`, `GET /search` |
| Agent | `POST /agent/ask`, streamed as server-sent events |
| Secret handling | Redacts credential-shaped values and skips sensitive file names before storage |

Helper file paths are validated against the configured allowed roots, and sensitive roots are
rejected.

Source: `backend/cmd/lunar-helper/`. User documentation: `docs/microservices/HELPER.md`.

---

## 10. Cross-cutting capabilities

| Capability | Detail |
|---|---|
| Credential vault | AES-256-GCM encryption at rest, per user, never returned in plain text |
| Session scoping | Feature stores are reset when the signed-in user changes |
| Theming | Dark and light mode driven by CSS custom properties |
| Main menu sidebar | The navigation sidebar collapses to icons with a persisted preference, hiding the Lunar brand and the user account block so only the collapse control and the icon rail remain. It becomes an overlay drawer with a backdrop at or below 768px |
| New chat button colour | The sidebar new chat button is drawn from the theme surface, border, and text tokens, so it differs between dark and light instead of resolving to the same inverted block in both |
| Markdown rendering | Answers render as Markdown through `marked` and `DOMPurify`, with a strict tag allowlist |
| Toasts | Shared notification component |
| Error states | Each feature reports its own failure rather than blanking the page |
| Plan workflow | Features are planned under `plan/<feature>/` before implementation |
| End-to-end tests | Playwright, grouped by feature area |

---

## 11. Not yet implemented

Listed so that nothing here is mistaken for working. Each has a plan or a backlog entry.

| Item | Where |
|---|---|
| Jira, Bitbucket, and Confluence agent tools | Stage 4 of `plan/code-copilot/` |
| Attachment picker in the composer | `plan/chat-ux/` W4. The upload endpoint, the storage, and the agent wiring exist and are verified by API; the composer control, the chips, and the progress state do not |
| Shared code index for cross-team questions | Open decision in `plan/code-copilot/` |
| Semantic code search using embeddings | Stage 5 of `plan/code-copilot/` |
| Guided tours for the Atlassian pages | `plan/user-guide/` |
| Helper download and distribution page | `plan/BACKLOG.md` |
| `CORS_ORIGIN` still wildcard | `plan/BACKLOG.md`, must be set before broad deployment |
| Several files above the 300-line limit | `plan/BACKLOG.md` |

---

## Documentation map

| Document | Covers |
|---|---|
| `guide.md` | This file. Every feature and subfeature |
| `AGENTS.md` | Operating manual: engineering rules and quality bar |
| `SKILLS.md` | Index of the skills, rules, and agents available in this repository |
| `docs/microservices/HELPER.md` | Helper installation, endpoints, and troubleshooting |
| `plan/BACKLOG.md` | Open items carried across features |
| `plan/<feature>/` | Per-feature plan, task checklist, and workstream breakdown |
| `.agents/` | Specialized agent definitions, rules, and skills |
| `frontend/e2e/` | End-to-end specifications grouped by feature area |

---

## Change log

Newest first. Every feature change adds a line here.

| Date | Change |
|---|---|
| 2026-10-05 | Settings and Copilot: OpenCode Go joins as a sixth AI provider, reaching `https://opencode.ai/zen/go/v1` and offering DeepSeek V4.1 Flash (Go), MiMo V2.6 Flash (Go), and Space Bunny Free (Go). Because every one of those speaks OpenAI chat completions, no new transport was needed. Model resolution now matches the OpenCode Go catalogue exactly before the legacy substring rule, which would otherwise have sent `deepseek-v4.1-flash` to the personal DeepSeek provider as `deepseek-chat`. Outbound provider requests now identify themselves with `lunar/1.0`, and the code agent sends `x-opencode-session` carrying the conversation id, both required by the Go terms. The AI providers settings screen is rebuilt around a reusable `AiProviderRow`, taking `AiProvidersSettings.vue` from 493 to 170 lines, and the Playwright provider locator now matches the provider name exactly instead of any text in the row. The provider row uses the OpenCode mark shipped as `frontend/public/icons/opencode.svg`, drawn as a white silhouette in dark mode like the other monochrome marks and left in its original two-tone artwork on white in light mode |
| 2026-10-05 | Copilot: the composer input no longer draws a focus outline, which resolved to near black in light mode. Focus is marked on the composer container through its border and a soft ring instead, so the field stays visibly focused without the hard outline |
| 2026-10-05 | Copilot: the code index chip moves out of the composer control bar into the slim header, centred between the breadcrumb and the new chat button. The composer area now holds only the composer and its optional hint |
| 2026-10-05 | Copilot: the model menu is now a two-level picker. It opens on Model and Effort; the Model level adds a live name filter and a back control and returns to the first level on selection, and the Effort level lists Low, Med, and High the same way, with `Escape` closing from either level. Thinking mode is always on and its toggle is removed, so `ThinkingControl.vue` and `EffortControl.vue` are deleted. The composer wraps long input and grows to four tenths of the viewport before scrolling. The Copilot page gets a slim 48px contextual header with a breadcrumb and a single new chat button in place of the 64px topbar, supplied through a new `PageLayout` header slot. The sidebar new chat button is re-tokenised to follow each theme, and the collapsed main menu now hides the Lunar brand and the user account block |
| 2026-10-05 | Copilot: attachment backend. `POST /api/copilot/attachments` accepts a `multipart/form-data` upload of at most 10 MB, enforces png, jpeg, webp, pdf, xlsx, xls, and docx with content sniffing for images and the extension for documents, stores it in SQLite scoped to the owner, and returns `{ id, name, kind, size, preview }`. PDF, Excel, and Word are reduced to text, capped at 60000 characters, and prepended to the question as a delimited untrusted block in the user message. `GET /api/copilot/attachments/{id}` lets the helper resolve an id, and an image is sent inline as a base64 `data:` URL only when the model reads images, otherwise the request is rejected with the model named. New dependencies: `github.com/xuri/excelize/v2`, `github.com/ledongthuc/pdf`, `github.com/extrame/xls` |
| 2026-10-05 | Main menu: the navigation sidebar collapses to a 72px icon rail from a control in its header, with the active route still highlighted and each label kept for assistive technology and as a tooltip. The preference is persisted as `menu_sidebar_collapsed`, read through a new `useMainSidebar` composable. At or below 768px the sidebar becomes an overlay drawer opened from a header button, with a backdrop, `Escape`, a focus trap, and focus returned to the opener, and the grid drops to a single column so no empty gutter is left behind |
| 2026-10-05 | Copilot: the `Copilot Workspace <model>` header is removed, so the conversation gains that height back, and the code index chip moved into a new control bar above the composer. The composer is extracted into `ChatComposer.vue`, with the model menu in `ModelPicker.vue`, which brings `ChatWindow.vue` from 703 to 196 lines and closes B.11. The `Enter to send, Shift + Enter for new line` hint text is removed while both shortcuts still work, so the composer sits 24.5px lower (measured). Thinking mode is now an icon-only brain control (`ThinkingControl.vue`) that renders gold when on and grey when off, at 8.00:1 and 5.86:1 contrast in dark and 6.27:1 and 6.92:1 in light. Reasoning effort (`EffortControl.vue`) moved out of the composer beside it, states that it applies only while thinking is on, and is dimmed and `aria-disabled` while thinking is off |
| 2026-10-05 | Copilot: fixed missing conversation context. The agent request never carried the previous turns, so a follow-up question had no referent and the model answered as if the conversation had just started. The request now sends the last ten turns, and the helper validates them without rejecting a long answer |
| 2026-10-05 | Copilot: code blocks render as a window mockup with line numbers and syntax highlighting, styled after the FlyonUI code mockup without adding FlyonUI. highlight.js is tree-shaken to eleven languages, and a brace fence meta highlights specific lines |
| 2026-10-03 | Copilot: deleting a conversation now asks for confirmation. Added a reusable `ConfirmDialog` in the shared UI components, focused on Cancel by default so Enter cannot confirm a destructive action by accident |
| 2026-10-03 | Copilot: fixed the diagram navigation arrows, which matched the wrong element because the registry was keyed by segment index while the lookup used diagram index |
| 2026-10-03 | Copilot: diagrams render in place instead of being collected at the end of the answer, which had left headings such as "6.1 Interaksi Antar Service" with nothing beneath them. The counter and arrows moved onto each card and scroll to the adjacent diagram |
| 2026-10-03 | Copilot: diagram cards carry a tag name, supplied by the model as a mermaid frontmatter title with a derived fallback. Flowchart labels now render: Mermaid is configured with `htmlLabels: false` so labels are SVG text rather than `foreignObject`, which the sanitizer removes |
| 2026-10-03 | Copilot: removed the "Source references" citation list. `CodeCitationList.vue` is deleted, along with the citation store and its plumbing, which became dead once nothing rendered it |
| 2026-10-03 | Copilot: answers render Mermaid diagrams with a gallery, a counter, and previous and next arrows. The system prompt now requires diagrams with a worked example, and arrow-notation `flow` blocks are converted to a top-down flowchart as a fallback |
| 2026-10-03 | Copilot: fixed a phantom blank line between every block in an answer, caused by `white-space: pre-wrap` on the Markdown container. Measured gap went from 33px to the intended 10px |
| 2026-10-03 | Copilot: assistant answer presentation rebuilt to match the timesheet-app reference: per-role avatars, flat assistant answer, collapsible reasoning panel, tool-call trail, copy and regenerate actions, copy controls on code and tables, and a streaming render throttle. `ChatWindow.vue` went from 999 to 702 lines by extracting `ChatMessage`, `ChatStreamingMessage`, `ChatReasoningPanel`, `ChatToolCalls`, `ChatStreamingIndicator`, and `ChatMessageActions` |
| 2026-10-03 | Settings: workspace source labels renamed to "On this computer" and "On the server", so the choice states where the folder lives rather than which component reads it |
| 2026-10-02 | Jira: disabled the hardcoded seed fallback, so an unreachable BRI Jira now reports `502` with a VPN hint instead of showing fabricated tickets |
| 2026-10-02 | Created this guide from a verified inspection of the running application |
| 2026-10-02 | Copilot: Markdown rendering for answers, using `marked` and `DOMPurify` |
| 2026-10-02 | Copilot: code-aware answers, local index, agent loop, and real line-range citations |
| 2026-10-02 | Helper: one-click connect through `GET /token`, replacing manual token copying |
| 2026-10-02 | Settings: light-theme logos for OpenAI and MiMo |
| 2026-10-02 | Copilot: redesigned New chat button and the `Cmd+Shift+O` shortcut |
| 2026-10-02 | Microservices: repository picker, model picker, and helper setup guide |
| 2026-10-02 | Removed the sidebar status card, the decorative header search, and the Confluence New page button |

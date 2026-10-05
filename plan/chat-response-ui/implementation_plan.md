# Chat Response UI — Implementation Plan

Status: in progress
Reference implementation (read-only): `/Users/erendt/Code/timesheet-app/frontend/src/components/servicemap/` and `components/copilot/CopilotPanel.vue`

---

## 1. Objective

Bring the assistant's answer presentation in lunar up to the standard already used in timesheet-app:
an avatar per role, a flat assistant answer beside its avatar instead of a bubble, a dedicated
reasoning panel, a visible tool-call trail, per-answer actions, and copy buttons on code and tables.

---

## 2. What the reference actually contains

The reference renders, in order, for one assistant message:

| Element | Component | Ported? |
|---|---|---|
| Avatar, assistant and user | inline in `CopilotPanel.vue` | yes |
| Agent progress while streaming | `ServiceMapProgressTracker.vue` | partly, see 4.4 |
| Tool call trail | `ServiceMapToolCalls.vue` | yes |
| Reasoning panel | `ServiceMapReasoningBubble.vue` | yes |
| Answer body | `ServiceMapMarkdownContent.vue` | layout only, see 3 |
| Created-subtask cards | inline in `CopilotPanel.vue` | no, timesheet-specific |
| Streaming pill with elapsed | inline in `CopilotPanel.vue` | yes |
| Source references | `ServiceMapSourceReferences.vue` | restyle |
| Token usage badge | `ServiceMapTokenUsageBadge.vue` | no, see 3 |
| Message actions | `ServiceMapMessageActions.vue` | yes |

---

## 3. Deliberate exclusions

**The Markdown parser is not ported.** The reference hand-rolls 763 lines in
`utils/markdownRenderer.ts` with no sanitizer. Lunar already renders Markdown with `marked` plus
`DOMPurify` behind a strict tag allowlist, verified in a browser against `script`, `onerror`,
`javascript:`, `iframe`, `style`, `onclick`, and `svg onload`. Replacing that with a hand-rolled
parser would be a security and maintainability regression. Only the presentation around it changes.

**The token usage badge is not ported.** The helper's SSE stream emits `text`, `reasoning`,
`tool_call`, `tool_result`, `sources`, `done`, and `error`. It carries no prompt, completion,
reasoning, or cache-hit token counts. Rendering the badge would require extending the agent loop and
the stream contract first. Recorded in `plan/BACKLOG.md`.

**Mermaid diagrams are not ported.** They would add a large dependency for a capability the code
answers do not currently need.

---

## 4. Work

### 4.1 Message layout

- Assistant: avatar at the left, answer flat beside it, no bubble background.
- User: right-aligned bubble, `max-width` bounded, asymmetric top-right corner, avatar at the right.
- Keep the existing timestamp, moved to a meta line that does not compete with the answer.

### 4.2 Reasoning panel

Replaces the current inline `thought-section`. Dashed border, subtle surface, a lightbulb, an
expand toggle, an estimated token count, and a blinking cursor while reasoning is still arriving.
Lunar has `msg.reasoning` and `msg.thinkingDurationMs`; the reference shows a token estimate, so
lunar shows the existing duration instead.

### 4.3 Tool call trail

New in lunar. The helper already streams `tool_call` events carrying `tool`, `action`, and `round`,
and the composable already tracks `ActiveToolCall[]`. The trail lists each call with its round, and
collapses when the answer is complete.

### 4.4 Streaming indicator

A pill with a spinner, a status label, and elapsed seconds measured on the client. The reference also
shows an agent phase; lunar has no phase in the stream, so the pill shows the current tool name when
one is running, and elapsed time otherwise.

### 4.5 Message actions

Copy the answer, and regenerate. Regenerate re-sends the preceding user question and replaces the
answer. It must be disabled while a stream is active.

### 4.6 Copy buttons on code and tables

Rendered inside the sanitized output. Because the HTML comes from `DOMPurify`, the buttons are added
after sanitization or via delegated click handling on the container, never as raw injected markup.

### 4.7 Streaming render throttle

The reference re-renders Markdown at most once per animation frame, roughly every 32ms. Lunar
currently re-parses on every token. Adopt the throttle.

### 4.8 File size

`ChatWindow.vue` is 999 lines and violates the 300-line rule. This work extracts the message block
into components, so the file must end smaller, not larger. Target under 400 lines with the message
rendering delegated.

---

## 5. Design tokens

Lunar and the reference use different token names. Map them, and never introduce a new colour.

| Reference | Lunar |
|---|---|
| `--color-text` | `--text` |
| `--color-text-secondary` | `--text` at reduced emphasis, or `--muted` |
| `--color-text-muted` | `--muted` |
| `--color-surface` | `--surface` |
| `--color-surface-subtle` | `--subtle` |
| `--color-surface-hover` | `--surface-hover` |
| `--color-border` | `--border` |
| `--color-border-hover` | `--border-strong` |
| `--color-primary` | `--accent` |
| `--color-primary-soft` | `--accent` at low alpha |
| `--color-noncoding` | `--warning` |

Lunar uses scoped CSS with custom properties, not Tailwind. Every ported component is written in
lunar's style, not the reference's utility classes.

---

## 6. Workstreams

| Workstream | Owns | Depends on |
|---|---|---|
| WS-A | `ChatWindow.vue` layout, wiring, `guide.md` | none, supervisor |
| WS-B | `ChatReasoningPanel.vue`, `ChatToolCalls.vue`, `ChatStreamingIndicator.vue` | none |
| WS-C | `ChatMessageActions.vue`, `ChatMessageContent.vue` copy buttons, `CodeCitationList.vue` restyle | none |

WS-B and WS-C create new files only. The supervisor wires them into `ChatWindow.vue` after both
report, so no two agents edit one file.

---

## 7. Verification

- `vue-tsc -b` exits zero, `vite build` succeeds.
- Browser check on `/copilot` with a real question: avatar per role, user bubble right-aligned,
  assistant flat, reasoning panel expands, tool trail shows rounds, copy button copies, regenerate
  re-runs, streaming pill shows elapsed.
- Copy buttons work on a rendered table and a code block.
- Sanitizer still blocks `script`, `onerror`, `javascript:`, and `iframe` after the copy-button work.
- The existing Playwright suite stays green.
- Every new file is under 300 lines and contains zero comments.

---

## 8. Risks

| Risk | Mitigation |
|---|---|
| Copy buttons reintroduce an XSS path | Add them after sanitization, and re-run the sanitizer check |
| Regenerate duplicates the question in the transcript | Reuse the existing user message, replace only the answer |
| Extracting from a 999-line file breaks existing behaviour | Wire after both workstreams report, then verify in the browser |
| Token names differ and a colour is invented | Token mapping table in section 5, enforced in review |

# Chat Response UI — Task Checklist

Legend: `[x]` done, `[ ]` pending.

---

## Stage F — Reference inspection (complete)

- [x] F.1 Read the reference message block in `CopilotPanel.vue`
- [x] F.2 Identified the ten elements the reference renders per assistant message
- [x] F.3 Found the reference hand-rolls its Markdown parser with no sanitizer
- [x] F.4 Confirmed lunar's `marked` + `DOMPurify` renderer is the safer engine and must be kept
- [x] F.5 Confirmed the helper stream carries no token counts, so the token badge cannot be ported
- [x] F.6 Confirmed the helper stream does carry `tool_call` events, so the tool trail can be ported
- [x] F.7 Mapped the reference colour tokens onto lunar's token set

---

## WS-B — Reasoning, tool trail, streaming indicator

- [x] B.1 `ChatReasoningPanel.vue`: dashed panel, expand toggle, duration, streaming cursor
- [x] B.2 `ChatToolCalls.vue`: list each call with tool, action, and round
- [x] B.3 `ChatToolCalls.vue`: collapse once the answer is complete
- [x] B.4 `ChatStreamingIndicator.vue`: spinner, status label, elapsed seconds
- [x] B.5 Every file under 300 lines, zero comments, lunar tokens only

---

## WS-C — Message actions, copy buttons, citation restyle

- [x] C.1 `ChatMessageActions.vue`: copy the answer, with a copied state
- [x] C.2 `ChatMessageActions.vue`: regenerate, disabled while streaming
- [x] C.3 Copy button on every rendered code block
- [x] C.4 Copy button on every rendered table
- [x] C.5 Copy controls added without reintroducing an XSS path
- [x] C.6 `CodeCitationList.vue` restyled to match the reference source references
- [x] C.7 Streaming render throttle, at most one Markdown parse per animation frame
- [x] C.8 Every file under 300 lines, zero comments, lunar tokens only

---

## WS-A — Layout and integration (supervisor)

- [x] A.1 Assistant message: avatar left, answer flat, no bubble
- [x] A.2 User message: right-aligned bounded bubble, asymmetric corner, avatar right
- [x] A.3 Timestamp moved to a meta line that does not compete with the answer
- [x] A.4 Wire `ChatReasoningPanel` in place of the inline `thought-section`
- [x] A.5 Wire `ChatToolCalls` from the existing `ActiveToolCall[]`
- [x] A.6 Wire `ChatStreamingIndicator` with client-measured elapsed time
- [x] A.7 Wire `ChatMessageActions`, with regenerate reusing the previous user message
- [ ] A.8 `ChatWindow.vue` ends under 400 lines
- [x] A.9 `guide.md` updated, with a change log line

---

## Verification

- [x] V.1 `vue-tsc -b` exits zero
- [x] V.2 `vite build` succeeds
- [x] V.3 Browser: avatar per role, user bubble right, assistant flat
- [x] V.4 Browser: reasoning panel expands and collapses
- [x] V.5 Browser: tool trail lists rounds during a stream
- [x] V.6 Browser: copy button copies the answer
- [x] V.7 Browser: regenerate re-runs without duplicating the question
- [x] V.8 Browser: copy works on a table and a code block
- [x] V.9 Sanitizer still blocks script, onerror, javascript:, iframe
- [x] V.10 Playwright suite stays green

# Chat Response UI — Subagent Workstreams

Supervisor owns WS-A and the integration. WS-B and WS-C run in parallel and create **new files only**,
so no two agents edit one file.

Reference implementation, read-only: `/Users/erendt/Code/timesheet-app/frontend/src/components/servicemap/`

---

## Rules for every workstream

1. **Never write to `/Users/erendt/Code/timesheet-app`.** It is read-only reference material.
2. **Do not port the reference Markdown parser.** Lunar uses `marked` + `DOMPurify` and that engine
   stays. Only presentation is ported.
3. **Write lunar style, not the reference style.** The reference uses Tailwind utility classes. Lunar
   uses `<style scoped>` with CSS custom properties. Translate, do not copy the class lists.
4. **Use lunar's tokens only.** Map from the reference: `--color-text`→`--text`,
   `--color-text-muted`→`--muted`, `--color-surface`→`--surface`,
   `--color-surface-subtle`→`--subtle`, `--color-surface-hover`→`--surface-hover`,
   `--color-border`→`--border`, `--color-border-hover`→`--border-strong`,
   `--color-primary`→`--accent`, `--color-noncoding`→`--warning`. Never invent a colour.
5. **Zero code comments.** Absolute.
6. **Every file under 300 lines.**
7. **Icons**: lunar does not use `@heroicons/vue`. Check what lunar already uses and follow it. Do not
   add a dependency.
8. Report files changed, line counts, the exact commands run, and their raw output.

---

## WS-B — Reasoning, tool trail, streaming indicator

Files owned, all new:

```text
frontend/src/features/copilot/components/ChatReasoningPanel.vue
frontend/src/features/copilot/components/ChatToolCalls.vue
frontend/src/features/copilot/components/ChatStreamingIndicator.vue
```

Props:

```ts
interface ChatReasoningPanelProps {
  reasoning: string
  isReasoning: boolean
  durationMs?: number
}

interface ChatToolCallsProps {
  calls: { tool: string; action: string; round: number }[]
  isStreaming: boolean
}

interface ChatStreamingIndicatorProps {
  label: string
  elapsedMs: number
}
```

Reference behaviour to match: the reasoning panel is a dashed, subtle, collapsed-by-default panel with
a toggle, a duration, and a blinking cursor while reasoning streams. The reference shows an estimated
token count; lunar already has `durationMs`, so show the duration instead.

The tool trail lists each call with its tool, action, and round, and collapses when the answer is
complete. The reference uses `ServiceMapToolCalls.vue` as the starting point.

The streaming indicator is a pill with a spinner, a label, and elapsed seconds. The reference also
shows an agent phase; lunar has no phase, so the caller passes a label.

---

## WS-C — Message actions, copy buttons, citation restyle

Files owned:

```text
frontend/src/features/copilot/components/ChatMessageActions.vue        new
frontend/src/features/copilot/components/ChatMessageContent.vue        edit
frontend/src/features/copilot/components/CodeCitationList.vue          edit
```

Props:

```ts
interface ChatMessageActionsProps {
  text: string
  disabled?: boolean
}
```

Emits `regenerate`.

Copy controls: the reference attaches copy buttons to code blocks and tables through delegated click
handling on the content container, using classes the renderer emits. Lunar's renderer does not emit
those classes, so add the controls **after sanitization**, by walking the sanitized DOM and injecting
the button element with `document.createElement`, never by concatenating HTML into the sanitized
string. Add a browser check that `script`, `onerror`, `javascript:`, and `iframe` are still blocked
afterwards, and report the raw result.

Streaming throttle: the reference re-renders at most once per animation frame with a 32ms floor. Adopt
it in `ChatMessageContent.vue`.

---

## Handoff

1. WS-B and WS-C report to the supervisor. No workstream edits `ChatWindow.vue`; the supervisor wires
   every component in after both report.
2. The supervisor then runs `vue-tsc`, `vite build`, the sanitizer check, and the browser verification.
3. Any contract change after work starts is supervisor-approved and broadcast to both workstreams.

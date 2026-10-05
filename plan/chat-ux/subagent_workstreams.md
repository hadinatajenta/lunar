# Chat UX and Attachments - Subagent Workstreams

## File ownership

One writer per file. A stream that needs a change outside its list must ask the parent instead of
editing it, so two streams never edit one file at once.

| Stream | May write | Must not write |
|---|---|---|
| W1 | `frontend/src/components/layout/AppSidebar.vue` and any new small composable under `frontend/src/components/layout/` | `ChatSidebar.vue`, `ChatWindow.vue`, `useCopilot.ts`, any composer component |
| W2 | `frontend/src/features/copilot/components/ChatWindow.vue`, `ChatComposer.vue`, `ThinkingControl.vue`, `EffortControl.vue` | `ChatSidebar.vue`, `useCopilot.ts` |
| W3 | `backend/internal/attachment/**`, `backend/internal/app/app.go` for the route only, `backend/cmd/lunar-helper/**` | any frontend file |
| W4 | `frontend/src/features/copilot/components/AttachmentPicker.vue`, `AttachmentChips.vue`, `frontend/src/features/copilot/api/**` | `ChatWindow.vue` beyond the agreed insertion point, `ChatSidebar.vue` |

`useCopilot.ts` is written by W1 only. W2 and W4 need values from it, and receive them as props or
as a request for the parent to add the export.

## Handoff rules

- The attachment contract in `implementation_plan.md` is frozen. W3 may not change a field, a status
  code, or an accepted type without W4 being updated in the same change.
- W2 owns the composer. W4 inserts its controls into the composer through a slot or a named prop
  agreed with the parent, and does not restructure the composer.
- Every stream runs `vue-tsc` or the Go build and tests before reporting done.
- Every stream reports what it did not verify, and why.

## Definition of done for a stream

1. The files it owns are written, with no comments and under 300 lines each.
2. Its checklist items in `task.md` are true, and it names the evidence for each.
3. Its build and tests pass, with the command and the raw result reported.
4. It reports what it could not verify.

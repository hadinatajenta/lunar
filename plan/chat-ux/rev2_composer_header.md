# Copilot Composer & Header Refinement (Rev 2)

## Objective

Refine the Copilot surface: replace the flat model list with a two-level Model/Effort menu,
make thinking mode permanently on, introduce a slim contextual header, make the composer wrap
instead of scroll, and fix theme consistency in the sidebar.

## Requirements

1. Model picker becomes a two-level dropup: root shows only "Model" and "Effort".
   - Click "Model" -> model list with a search input and a "<" back button (reuse the existing
     chevron-left icon path already used in AppSidebar). Selecting a model applies it and returns
     to the root menu.
   - Click "Effort" -> list of Low / Med / High. Selecting applies it and returns to root.
2. Thinking mode defaults on and cannot be disabled; remove the ThinkingControl toggle.
3. Copilot page uses a slim contextual header (transparent, backdrop-blur, breadcrumb + "+" new chat
   button only; no clock, no history, no theme toggle), matching the reference HTML header.
4. The chat textarea wraps to a new line instead of scrolling horizontally, and its max height is
   4/10 of the viewport (never half the page).
5. The new chat button color follows each theme (fix "black in both themes").
6. When the main sidebar is collapsed, hide the Lunar logo and the user initials/name.

## Decisions

- EffortControl.vue is folded into ModelPicker.vue as a menu view; the separate control-bar widget
  is removed. EffortControl.vue and ThinkingControl.vue are deleted once unused.
- thinkingMode stays a ref in useCopilot but is forced to true and no longer written back to a
  user-toggleable control; the stored default remains true.
- The slim header is rendered only on the copilot page via a PageLayout header slot/prop; other
  pages keep the 64px topbar.
- Copilot container height changes from calc(100vh - 64px) to calc(100vh - 48px) to match the slim
  header height.

## Workstreams

| Stream | Owns | Depends on |
|---|---|---|
| A | ModelPicker.vue (two-level model/effort menu) | none |
| B | ChatComposer.vue, ChatWindow.vue, useCopilot.ts, copilot-storage.ts, ChatSidebar.vue, chat-sidebar.css, ChatStreamingMessage.vue; delete ThinkingControl.vue + EffortControl.vue | none |
| C | PageLayout.vue, CopilotPage.vue, new CopilotHeader.vue, AppSidebar.vue, app-sidebar.css, app-sidebar-drawer.css | none |
| D | verification, e2e, guide.md | A,B,C complete |

A, B, C run in parallel; D runs after integration.

## Verification

- vue-tsc exits 0; vite build succeeds; go build/test unaffected.
- Model menu: root -> model (search + back) -> select -> root; effort -> low/med/high -> root.
- Thinking toggle absent; thinking always on; reasoning still streams.
- Composer wraps, max height 4/10 viewport.
- New chat button contrasts in both themes.
- Collapsed sidebar shows icons only (no logo, no initials).

## Risks

| Risk | Mitigation |
|---|---|
| PageLayout header swap breaks other pages | Slot/prop defaults to the existing topbar |
| Thinking forced true breaks streaming label | Keep thinkingMode=true in state; update label to always show reasoning |
| Sidebar collapse hides brand but breaks focus flow | Keep aria-labels; hide visually only |

# Copilot Composer & Header Refinement - Tasks

## A - Model & Effort menu
- [x] Root menu shows only "Model" and "Effort"
- [x] Model view: search input + "<" back button (reused chevron icon)
- [x] Model search filters the configured model list live
- [x] Selecting a model applies it and returns to root
- [x] Effort view: Low / Med / High list
- [x] Selecting effort applies it and returns to root
- [x] Escape closes the menu from either view (window listener, because the focused control is replaced)

## B - Composer, thinking, new-chat color
- [x] ThinkingControl removed; thinking always on
- [x] EffortControl removed from control-bar (folded into ModelPicker)
- [x] Composer textarea wraps (no horizontal scroll)
- [x] Composer max height 4/10 viewport (40vh) then scrolls
- [x] New chat button color follows theme (surface/border/text tokens, not an inverted block)
- [x] Conversation highlight and delete hover also re-tokenised for both themes

## C - Slim header & collapsed sidebar
- [x] Slim contextual header on copilot only (breadcrumb + "+" new chat)
- [x] Other pages keep 64px topbar via a new PageLayout header slot
- [x] Copilot container height matches 48px header; ChatWindow uses height 100%
- [x] Collapsed sidebar hides logo and user initials/name
- [x] Mobile menu trigger kept in the slim header so navigation is not lost at <=768px

## D - Verification & docs
- [x] vue-tsc / vite build pass (npm run build exit 0)
- [x] e2e: 14 passed, 6 failed; the same 6 fail on the baseline commit (pre-existing)
- [x] e2e specs updated and extended (controls, model-selection, chat-layout-visual, new header-layout)
- [x] guide.md updated + changelog entry

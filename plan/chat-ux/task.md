# Chat UX and Attachments - Task Checklist

Status of every item. Updated as work completes. `[x]` done and verified, `[~]` in progress,
`[ ]` not started, `[!]` blocked.

## W1 - MENU sidebar collapse and mobile

The sidebar in scope is the MAIN NAVIGATION menu at
`frontend/src/components/layout/AppSidebar.vue` (Overview, Jira BRI, Bitbucket BRI, Confluence BRI,
Microservices BRI, Copilot, Settings). It is NOT the conversation history column.

An earlier attempt misread this and edited the conversation history sidebar instead. That work is
kept because it is functional and because reverting it would also revert unrelated work in the same
files, but it was not requested. See "Unrequested" at the end of this file.

- [x] Collapse control in the sidebar header, icon-only with an `aria-label`
- [x] Collapsed state persisted like the other Copilot preferences
- [x] Sidebar collapses to icons, or hides, without breaking the active conversation highlight
- [x] Narrow viewport turns the sidebar into an overlay drawer with a backdrop
- [x] `Escape` closes the drawer
- [x] A backdrop click closes the drawer
- [x] Focus returns to the control that opened the drawer
- [x] Focus is trapped inside the drawer while it is open
- [x] Sidebar keeps its own scroll, and the conversation keeps its own
- [x] Verified at 1440px and at 390px
- [x] Verified with the keyboard alone

## W2 - Header, composer, thinking, effort

- [x] `ChatComposer.vue` extracted from `ChatWindow.vue`
- [x] `ChatWindow.vue` drops below 300 lines, closing B.11 (196 lines)
- [x] Composer props and events unchanged by the extraction, streaming still works
- [x] `Copilot Workspace <model>` header removed
- [x] Conversation gains the width the header occupied
- [x] `Enter to send - Shift + Enter for new line` text removed
- [x] `Enter` still sends
- [x] `Shift + Enter` still inserts a newline
- [x] Composer lowered by the height the hint occupied (measured 24.5px)
- [x] Thinking toggle removed from the composer
- [x] Thinking toggle shows a brain icon only, no text
- [x] Thinking on renders gold, off renders grey, in both themes
- [x] Contrast of both states meets WCAG AA (dark 8.00:1 and 5.86:1, light 6.27:1 and 6.92:1)
- [x] Effort control moved out of the composer to a visible, reachable place
- [x] The effort control states that it applies only while thinking is on
- [x] Effort selection persists across a reload
- [x] The removed hint leaves no dead style or dead constant behind

## W3 - Attachment backend

- [x] `POST /api/copilot/attachments` accepts `multipart/form-data`
- [x] Request body capped with `http.MaxBytesReader`
- [x] Accepted types enforced: png, jpeg, webp, pdf, xlsx, xls, docx
- [x] An unsupported type returns 400 naming the accepted types
- [x] An oversized file returns 400 without reading it fully
- [x] PDF text extraction
- [x] Excel text extraction
- [x] Word text extraction
- [x] Extracted text capped, so a large document cannot exhaust the context
- [x] Extracted text is delimited and treated as untrusted data
- [x] Attachment id passed on the question and loaded by the agent
- [x] An image is sent as an image part only when the model accepts one
- [x] A model that cannot read images says so instead of sending one
- [x] No path traversal: an id is never used to build a filesystem path
- [x] Unit tests cover extraction, the type allowlist, and the size limit

Notes: W3 adds `GET /api/copilot/attachments/{id}` beside the frozen `POST`, because the agent
question path runs in the separate `lunar-helper` process and must resolve an id over HTTP. The
frozen `POST` request and response shapes are unchanged. The `preview` field is a short text label
for both kinds: a whitespace-collapsed excerpt for a document, and `PNG image, 12x8` style text for
an image, not a data URL.

## W4 - Attachment frontend

- [ ] Attach control in the composer, icon-only with an `aria-label`
- [ ] File picker accepts the allowlisted types
- [ ] A selected file shows as a chip with its name and size
- [ ] A chip can be removed before sending
- [ ] Upload progress is shown, and a failure is shown with its reason
- [ ] A rejected type shows the accepted types
- [ ] Attachments are sent as ids on the question
- [ ] Attachment chips clear after a successful send
- [ ] Attachment chips survive a failed send, so the user does not lose their file
- [ ] An image attached to a model without vision shows a clear message
- [ ] Keyboard reachable, and each control has an accessible name

## Cross-cutting

- [ ] `guide.md` updated for every feature and subfeature
- [x] `guide.md` change log updated
- [x] `plan/BACKLOG.md` updated, B.11 closed
- [x] `vue-tsc` exits 0
- [x] `vite build` succeeds
- [x] `go build ./...` succeeds
- [x] `go test ./...` passes
- [x] No comments in any new or edited code
- [x] No file exceeds 300 lines
- [x] Both themes checked for contrast

## Unrequested

Work that exists in the tree but was not asked for. Listed so it can be accepted or removed
deliberately rather than by accident.

- [ ] The conversation history column (`ChatSidebar.vue`) also gained a collapse control and a
      mobile drawer. This was produced by the mis-assigned W1. It works and passes the type check.
      Remove it if it is not wanted.

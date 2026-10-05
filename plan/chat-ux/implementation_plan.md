# Chat UX and Attachments

## Objective

Improve the Copilot chat surface so the conversation gets more room, the sidebar survives a phone
screen, the composer states only what is true, and a question can carry an attachment.

Six changes were requested:

1. A sidebar collapse control, and a responsive sidebar on mobile.
2. Remove the `Copilot Workspace <model>` header, which narrows the conversation.
3. Remove the `Enter to send - Shift + Enter for new line` hint text while keeping both shortcuts.
4. Lower the composer by the height the removed hint occupied.
5. Move the thinking toggle out of the composer, show it as a brain icon with a gold on state and a
   grey off state, and drop the word "thinking".
6. Relocate the reasoning effort control, and add image, PDF, Excel, and Word attachments.

## Findings

### Reasoning effort is genuinely coupled to thinking mode

The user asked to be corrected if this was wrong. It is not wrong. In
`internal/agent/infrastructure/toolcall_stream.go`:

```go
if enableThinking && reasoningEffort != "" && reasoningEffort != "none" {
    requestBody[thinkingFieldName] = map[string]string{"type": thinkingModeEnabled}
    requestBody[reasoningEffortFieldName] = reasoningEffort
    requestBody[outputConfigFieldName] = map[string]string{"effort": reasoningEffort}
    return
}
requestBody[thinkingFieldName] = map[string]string{"type": thinkingModeDisabled}
requestBody["temperature"] = 0.1
```

When thinking is off, `reasoning_effort` is ignored entirely. The control must therefore stay
attached to the thinking toggle, and must say so.

### The composer is not a component

There is no `ChatComposer.vue`. The composer lives inside `ChatWindow.vue`, which is 703 lines and
already exceeds the 300-line limit in AGENTS.md section 6. Extracting the composer is required by
the requested changes and closes backlog item B.11.

### File parts are scaffolding only

`domain.FilePart{Type, FileID}` and `domain.TextPart` exist in `internal/agent/domain/ai.go`, and
`hasFileParts` and `stripFileParts` exist for the failover path. There is no upload client, no
upload endpoint, and no provider integration. A search for `UploadFile`, `/files`, and `multipart`
across both infrastructure packages returns nothing. The `file_id` shape implies a provider-side
upload that does not exist yet.

### DeepSeek does read images, but only through one model

Confirmed against the vendor documentation at `https://api-docs.deepseek.com/guides/vision/`:

- The `deepseek-flash` model accepts images. `deepseek-chat` and the reasoning model do not.
- `deepseek-flash` is already in the model catalog at `frontend/src/features/copilot/utils/models.ts`.
- Supported formats are JPEG, PNG, GIF, and WebP, detected from the file content rather than the
  MIME type or the file name.
- An image can be sent inline as a base64 `data:` URL, as an external URL, or by `file_id` after an
  upload to the Files API.
- Limits: 32 MiB per inline image, 48 MiB request body, 600 images per request, and at most 1024
  tokens per image after resizing.

The Files API accepts **images only**. `https://api-docs.deepseek.com/guides/files_api/` states the
supported formats are JPEG, PNG, GIF, and WebP. It does not accept PDF, Excel, or Word. So a document
must still be reduced to text before it reaches the model, while an image does not.

### The request path sends content as a string

`domain.MessageContent` and its parts are decoded from the model's reply, but there is no
`MarshalJSON` for it, and the request body carries `content` as a plain string. Sending an image
requires the request path to emit a content-parts array when parts are present. This is part of W3.

## Decisions

### Attachment strategy

Two paths, chosen by what the file is:

- A **document** (PDF, Excel, Word) is reduced to text on the backend and prepended to the question
  as a delimited block. This works with every provider and needs no provider-side upload.
- An **image** is sent inline as a base64 `data:` URL in a content-parts array, because a vision
  model reads it directly and extraction would lose the point of attaching it. Inline is chosen over
  the Files API because it needs no extra upload round trip, no server-side file store, and no
  expiry management.

An image is only sent to a model that accepts one. The model catalog marks which do, and the
composer states plainly when the selected model cannot read an image rather than sending one that
would be ignored.

### Sidebar collapse

The sidebar keeps a single source of truth for its collapsed state, persisted like the other Copilot
preferences. On a narrow viewport it becomes an overlay drawer with a backdrop, closes on `Escape`
and on a backdrop click, and returns focus to the control that opened it.

## Workstreams

| Stream | Owns | Depends on |
|---|---|---|
| W1 | `ChatSidebar.vue`, sidebar state in `useCopilot.ts` | none |
| W2 | `ChatWindow.vue`, new `ChatComposer.vue`, new `ThinkingControl.vue`, `EffortControl.vue` | none |
| W3 | Attachment backend: endpoint, extraction, provider image part | contract below |
| W4 | Attachment frontend: picker, chips, api client | W3 contract |

W1 and W2 touch different files and run in parallel. W3 and W4 run in parallel against the frozen
contract below. W4 must not start before the contract is agreed, and W3 must not change the contract
without W4 being updated in the same change.

### Frozen contract for W3 and W4

```text
POST /api/copilot/attachments
  multipart/form-data
  file: the uploaded file, at most 10 MB
  response 201: { "id": string, "name": string, "kind": "image" | "document", "size": number, "preview": string }
  response 400: { "error": string }  unsupported type, or over the size limit

The id is passed back on the question:
  AgentAskRequest.attachments: string[]

The backend loads each id, extracts its text, and prepends it to the question as a delimited block.
An image id is passed to the model as an image part only when the model accepts one.
```

Accepted types: `image/png`, `image/jpeg`, `image/webp`, `application/pdf`, `.xlsx`, `.xls`,
`.docx`. Everything else is rejected with a message naming the accepted types.

An image is validated by its **content**, not by its declared MIME type or its file name, matching
how the vendor detects the format.

## Verification

- Every workstream runs `vue-tsc`, `vite build`, and the Go build and tests.
- The sidebar is checked at 1440px and at 390px, with the keyboard alone for the drawer.
- The composer is checked for both shortcuts after the hint is removed, which is the point of the
  change.
- An attachment of each accepted type is sent, and the answer is checked to have used its content.
- A rejected type and an oversized file both show a message rather than failing silently.
- The gold and grey thinking states are checked in both themes, including contrast.

## Risks

| Risk | Mitigation |
|---|---|
| Extraction libraries are new dependencies | Approve before adding; prefer the standard library where it is enough |
| A large file exhausts memory | Cap the body with `http.MaxBytesReader`, cap the extracted text, and reject early |
| An extracted document injects instructions | Treat extracted text as untrusted data, delimit it, and never let it become a system message |
| The composer extraction breaks streaming | Move the composer without changing its props or its events, and verify streaming after |

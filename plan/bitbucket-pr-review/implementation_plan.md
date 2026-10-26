# Bitbucket PR review: model picker, modal layout, and review quality

## Objective

Make the Bitbucket pull request review modal follow the application's component and accessibility
standards, and make the generated AI review complete, evidence-based, and explicit about what is
only an assumption.

## Findings from the current implementation

Verified by reading the code. Each item names the file and the line that proves it.

| # | Finding | Evidence |
|---|---|---|
| F1 | The review prompt has no severity levels, no file or line references, no suggested fix, no "do not invent" rule, no strengths, and no approve or request-changes recommendation. | `backend/internal/copilot/infrastructure/llm_client.go:245-257` |
| F2 | PR title, description, author, and branches are never sent to the model, so the review cannot know the change's purpose and reads as generic. The description is already parsed from Bitbucket and then dropped. | `backend/internal/copilot/infrastructure/atlassian_bitbucket_client.go:53`, `backend/internal/bitbucket/application/service.go:117-131` |
| F3 | A JSON parse failure fabricates `"Automated AI review completed."` and posts the raw model text as the review comment. | `backend/internal/copilot/infrastructure/llm_client.go:279-286` |
| F4 | `max_tokens` is never sent on the OpenAI-compatible path, and the Claude path hardcodes `4096`, which is below a full structured review plus reasoning. A truncated response is invalid JSON, which lands in F3. | `backend/internal/copilot/infrastructure/llm_client.go:106-110`, `llm_client_claude.go:27` |
| F5 | `finish_reason` is not parsed, so truncation is silent. | `openAIChatResponse` in `llm_client.go:94-109` |
| F6 | The diff is cut at 24,000 bytes in the middle of a file and pasted into the prompt with no delimiter and no statement that it is data rather than instructions. | `backend/internal/copilot/infrastructure/llm_client.go:241-243`, `257` |
| F7 | `ReasoningEffortMedium` is passed to `ChatCompletion` and then unused for every OpenAI-compatible provider, so the effort setting has no effect there. | `backend/internal/copilot/infrastructure/llm_client.go:89-110` |
| F8 | When the diff request fails, the service silently falls through to the offline stub, which returns a canned review. An outage therefore looks like a real review. | `backend/internal/bitbucket/application/service.go:115-135` |
| F9 | The modal's model picker is a bare native `<select>`, while the application standard is `ModelPicker.vue`. | `frontend/src/features/bitbucket/components/PullRequestReviewModal.vue:182-194` |
| F10 | The modal is a single 680px column with the diff above the comment box, so a large diff pushes the review and the comment apart. | `PullRequestReviewModal.vue:332-344`, `464-472` |
| F11 | The meta row reads `0 files · +0 −0` against real Bitbucket because the list mapping never fills those fields, while the diff already carries the true totals. | `atlassian_bitbucket_client.go:288-305`, `PullRequestReviewModal.vue:128` |
| F12 | Findings render as plain text with no severity grouping, and the generated Markdown comment is only visible as raw text inside a textarea. `renderMarkdown` exists and is unused here. | `PullRequestReviewModal.vue:211-235`, `frontend/src/lib/markdown.ts:63` |
| F13 | `.btn` uses `transition: all`, the comment field has no `<label>`, focus is not moved into or trapped inside the dialog, and `Escape` is bound on `window`. | `PullRequestReviewModal.vue:230-235`, `746`, `74-87` |
| F14 | The dialog claims it is "Analyzing the diff, tests, and linked issues", but only the diff is sent. | `PullRequestReviewModal.vue:208` |
| F15 | `frontend/e2e/components/bitbucket/PrReviewModal.ts` is unused and its test ids match no element in the component. | no import site in `frontend/e2e` |

## Contracts

### `AICodeReview` (`GET`/`POST` `/api/bitbucket/prs/{id}/ai-review`)

```json
{
  "pr_id": "#184",
  "model": "DeepSeek-V4 Pro (Thinking)",
  "prompt_version": "pr-review-v2",
  "summary": "string",
  "findings": [
    {
      "severity": "blocker | suggestion | nitpick",
      "category": "correctness | security | performance | readability | testing | compatibility",
      "file": "src/auth/session.ts",
      "line": 42,
      "title": "short imperative title",
      "explanation": "why this is a problem",
      "suggested_fix": "what to change, with a code example when useful",
      "confidence": "confirmed | assumption",
      "evidence": "the exact diff line this finding is based on"
    }
  ],
  "assumptions": ["something the review could not verify from the diff"],
  "strengths": ["what the change does well"],
  "recommendation": "approve | request_changes",
  "generated_comment": "markdown, ready to post",
  "truncated": false
}
```

`truncated` reports that the diff exceeded the review budget and some files were not read. When it
is true, the model is told which files were omitted and must list them under `assumptions`.

### Prompt slots

`review_prompt.go` renders a template with named slots: `repository`, `pull_request`, `title`,
`purpose`, `author`, `branches`, `languages`, `files`, `tests_touched`, `omitted_files`, and
`diff`. Language detection, file inventory, test detection, and diff numbering are computed in Go,
so the model is told facts rather than asked to infer them.

## Stages

1. Backend domain: structured `ReviewFinding`, `AICodeReview.Model`, `PromptVersion`, `Assumptions`,
   `Strengths`, `Recommendation`, `Truncated`; `PullRequest.Description`.
2. Backend prompt: `review_prompt.go` with the versioned template, numbered diff rendering, language
   detection, test detection, and a per-file budget that records omitted files.
3. Backend client: `ChatOptions` (model output cap, JSON output mode, fail on truncation), parsing
   of `finish_reason`, one repair round on invalid JSON, and a typed error instead of the fabricated
   fallback.
4. Backend service: carry PR metadata into the review, and fail loudly when the diff cannot be read.
5. Backend tests: deterministic unit tests for numbering, language detection, budget omission,
   prompt invariants, and the truncation and repair paths.
6. Frontend: extend `ModelPicker.vue` with a bindable value and an optional effort control, rewrite
   the modal as a split dialog with tabs, severity groups, an assumptions block, a Markdown preview
   of the comment, a focus trap, and a `<label>` on the comment field.
7. End-to-end: update the page object, the fixtures, and the review specs; delete the dead component
   object.
8. Documentation: `guide.md` section 4 and change log.

## Verification

- `go build ./...` and `go test ./...` in `backend/`.
- `vue-tsc -b` and `vite build` in `frontend/`.
- `npx playwright test tests/bitbucket` in `frontend/`.

## Risks and decisions

- JSON output mode is enabled only for DeepSeek and OpenAI, the two providers whose support is
  documented. Other providers keep the prompt-only contract, and validation still applies.
- An output cap is set only where the provider's limit is known. Gemini, MiMo, and OpenCode Go are
  left uncapped rather than guessed, and that gap is recorded in `guide.md`.
- The review comment is the deliverable, so the model still writes it in full. The structured fields
  exist so the interface can group and label the findings before the reviewer posts them.

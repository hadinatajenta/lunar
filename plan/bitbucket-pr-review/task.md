# Task checklist

## Backend

- [x] `domain/entity.go`: structured `ReviewFinding`, extended `AICodeReview`, `PullRequest.Description`
- [x] `copilot/infrastructure/review_prompt.go`: versioned template, context builder, numbered diff, budget
- [x] `copilot/infrastructure/review_comment.go`: payload parsing, citation checking, severity groups
- [x] `copilot/infrastructure/review_format.go`: comment assembly and section formatting
- [x] `copilot/infrastructure/review_service.go`: `CodeReview`, repair round, token cap, review assembly
- [x] `copilot/infrastructure/llm_client.go`: `ChatOptions`, `finish_reason`, JSON output, no swallowed parse error
- [x] `copilot/infrastructure/llm_client_claude.go`: honour the configured output cap
- [x] `bitbucket/application/service.go`: pass PR context, stop fabricating on a diff failure
- [x] `bitbucket/infrastructure/atlassian_bitbucket_client.go`: `GetPullRequest`, description mapping, canned stub removed
- [x] `bitbucket/transport/handler.go`: `502` for a provider failure, `404` for a missing pull request
- [x] Unit tests for the prompt builder, the citation check, the comment assembly, and the client options
- [x] `go build` and `go test` on `internal/bitbucket/...` and `internal/copilot/...`
- [x] `gofmt -l` clean on both trees

## Frontend

- [x] `composables/useFocusTrap.ts`: shared trap, extracted from `AppSidebar.vue`, used by both
- [x] `features/copilot/components/ModelPicker.vue`: bindable value, optional effort, trigger test id
- [x] `features/bitbucket/types.ts`: structured review types and `PullRequest.description`
- [x] `features/bitbucket/components/PullRequestReviewDiff.vue`: extracted diff viewer, per-file collapse
- [x] `features/bitbucket/components/ReviewFindingGroup.vue`: severity group with evidence, fix, and assumption tag
- [x] `features/bitbucket/components/PullRequestReviewModal.vue`: split dialog, tabs, Markdown preview, confirm on decline
- [x] `features/bitbucket/pages/BitbucketPage.vue`: error state, retry, and review-state reset
- [x] `vue-tsc -b` reports zero errors under `src/`
- [x] `vite build` succeeds
- [x] `npm test` (frontend unit tests) still passes

## End-to-end

- [x] `e2e/pages/BitbucketPage.ts`, `e2e/data/bitbucket.data.ts`, and the review specs updated
- [x] Dead `e2e/components/bitbucket/PrReviewModal.ts` deleted
- [x] `npx playwright test tests/bitbucket --list` lists 24 tests
- [x] `tests/bitbucket` executed: 24 passed, using Brave as the Chromium binary because the bundled Playwright browser is not installed
- [x] Two runtime failures found by the suite and fixed in `src`: the missing `review-modal-repo` element, and a model menu that was clipped by the dialog so a configured model could not be clicked
- [x] `.gitignore` no longer hides `frontend/e2e/data/`, which had never been tracked

## Documentation

- [x] `guide.md` section 4
- [x] `guide.md` section 11 for the gaps this change did not close
- [x] `guide.md` change log

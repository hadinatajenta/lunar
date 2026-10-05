# Microservices BRI: Branch Picker, Repository Actions, Postman Export - Tasks

Nothing is implemented yet. `[ ]` not started, `[~]` in progress, `[x]` done and verified,
`[!]` blocked.

## Gate - all confirmed

- [x] Q1: choosing a branch switches the working tree, `git checkout`
- [x] Q2: discard is tracked-only, so the user can still pull
- [x] Q3: nothing is written on the server; the JSON streams and downloads in the browser
- [x] Q4: only models whose provider has a key; with no key at all the export is locked
- [x] Q5: one repository per export, the row that opened the modal
- [x] Q6: effort fixed to `medium` internally and never shown to the user

## Phase 1 - Git actions and branch picker  [DONE]

### 1A backend  [DONE]

- [x] branch list from `for-each-ref`, current branch marked and pinned first
- [x] strict branch name validator, tested with `;`, a space, `..`, a leading `-`, and a slash
- [x] `POST /checkout` switching the working tree, refusing a dirty tree with 409
- [x] `POST /fetch` on the current branch
- [x] `POST /pull` on the current branch
- [x] `POST /stash` creating a stash entry
- [x] `POST /discard` restoring tracked files only, never `git clean`
- [x] all five ported to the helper under its token guard
- [x] all five ported to the backend under auth
- [x] repository names resolved with `ResolveRepoName` and `ResolveRepoPath` on every call
- [x] git errors mapped to typed results without leaking internal paths
- [x] `runGit` reused unchanged: no shell, argument array, timeout, no credential prompt
- [x] unit tests per action against a scratch repository

### 1B frontend  [DONE]

- [x] `.branch-pill` becomes a button and keeps its current look
- [x] `BranchPicker.vue` opens dropup or dropdown by measured remaining space
- [x] picker pins the current branch, then up to five rows, scrollable
- [x] picker has a branch search field filtering locally
- [x] picker closes on Escape, outside click, and selection, returning focus
- [x] picker is keyboard operable with an accessible name
- [x] `RepositoryActionsMenu.vue` with a `...` trigger replaces the bare chevron
- [x] items: Pull, Fetch, Stash changes local, Discard changes, Details
- [x] every item carries an icon depicting its function
- [x] discard opens a confirmation modal, cancelled by Cancel, Escape, and backdrop
- [x] discard cannot proceed without an explicit confirm
- [x] busy state on the row while an action runs, menu disabled meanwhile
- [x] failures shown as a readable message, never a silent no-op
- [x] API client for branches and the five actions, helper and server sources

### 1Q QA  [DONE]

- [x] `go build ./...` and `go test ./...`
- [x] `vue-tsc` and `vite build`
- [x] e2e: open the picker, search, select, and confirm the switch
- [x] e2e: the kebab menu lists every action
- [x] e2e: discard Cancel closes with no repository change
- [x] adversarial review of the git argument path
- [x] regression run against the recorded baseline

## Phase 2 - Export pipeline

### 2A backend

- [ ] `selector.go` implementing the include and exclude rules
- [ ] `src/vendor` includes `panama/src/vendor`
- [ ] root `vendor/`, `node_modules/`, and `library/vendor/` excluded
- [ ] `*_test.go`, `test/`, `tests/`, and `mocks/` excluded so routes are not duplicated
- [ ] `wire.go` and `wire_gen.go` included
- [ ] file count and byte caps with an explicit refusal when exceeded
- [ ] `classifier.go` returning include or exclude plus a reason
- [ ] refuses `mms` as a CodeIgniter monolith
- [ ] refuses `fe-mms` as a Vue frontend with no server framework
- [ ] refuses a repository with no discoverable route
- [ ] prompt pack with the shared system prompt and one appendix per framework
- [ ] the eight stacks detected: Go net/http, Gin, Fiber, Echo, PHP, Laravel, Lumen, Node.js
- [ ] Wire appendix instructing the model to read `wire_gen.go`
- [ ] prompt states keys are mandatory and unknown values are empty strings
- [ ] provider call reusing the agent credential path, effort fixed to `medium`
- [ ] `normaliser.go` filling structural keys without inventing data
- [ ] `validator.go` refusing zero requests and incomplete requests
- [ ] fence unwrapping, then strict unmarshal into Postman v2.1
- [ ] SSE endpoint with stage and result events, on the helper and on the backend
- [ ] the response carries no write to disk anywhere
- [ ] unit tests for selector, classifier, prompt assembly, normaliser, and validator

### 2B frontend

- [ ] modal lists models from the configured provider catalogue, not the old provider list
- [ ] free models such as `space-bunny-free` badged
- [ ] with no provider key the export control is disabled with a reason and a Settings link
- [ ] no effort control and no effort wording anywhere in the modal
- [ ] `Export collection` enabled only with a usable model selected
- [ ] SSE progress rendered as stages, not a frozen spinner
- [ ] result opens a download modal offering `<service>-postman-collection.json`
- [ ] the request count is shown so a hollow result is obvious
- [ ] object URL revoked after download
- [ ] failure keeps the modal open with a readable reason and a retry
- [ ] repository model preference preserved

### 2Q QA

- [ ] selector verified against panama, merchant-service-laravel, micro-user-mms, mocash_api, and fe-mms
- [ ] `mms` and `fe-mms` refused with a reason
- [ ] a valid but hollow JSON fixture is refused
- [ ] a fenced JSON response is unwrapped
- [ ] a non-JSON response fails loudly
- [ ] export of `atlas` and `aurora`: every route appears exactly once, nested groups resolved
- [ ] every emitted request has a name, method, url path, headers, and query or body keys
- [ ] downloaded file imports into Postman
- [ ] no file written on the server, asserted after the run

## Phase 3 - Hardening, docs, end-to-end

### 3A frontend

- [ ] download modal keyboard reachable and dismissible
- [ ] empty, in-progress, failed, and succeeded states all readable
- [ ] retry after a failure without reopening the panel
- [ ] long repository names and file names do not break the layout

### 3B docs

- [ ] `guide.md` feature table for the branch picker, the actions, and the export
- [ ] `guide.md` change log
- [ ] `plan/BACKLOG.md` updated with anything deferred

### 3Q QA

- [ ] full-page e2e on atlas and aurora covering all three features
- [ ] keyboard-only pass on both new menus
- [ ] both themes checked
- [ ] no new failures against the recorded baseline

## Cross-cutting

- [ ] no comments in new or edited code
- [ ] every file under 300 lines, split by responsibility where needed
- [ ] `guide.md` updated for every user-visible change
- [ ] no repository operation reachable without an explicit user action

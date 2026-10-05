# Microservices BRI: Branch Picker, Repository Actions, and Postman Export

## Objective

1. The Branch column becomes interactive: clicking a repository's branch opens a picker listing
   branches with a search field, and choosing one switches the working tree to it.
2. The Action column becomes a kebab menu holding Pull, Fetch, Stash changes, Discard changes, and
   Details.
3. "Choose AI model" becomes a working export that reads one repository, asks a model for a Postman
   collection, validates it is complete, and downloads the JSON straight to the browser.

## Confirmed decisions

| Question | Decision |
|---|---|
| Q1 branch selection | Switch branch, `git checkout` |
| Q2 discard scope | Tracked files only, so the user can still pull. No `git clean` |
| Q3 output | **Nothing is written on the server.** The JSON streams to the browser and the modal offers the download |
| Q4 model list | Only models whose provider already has a key. With no key at all the export is locked |
| Q5 scope | One repository per export, the row that opened the modal. Choosing `aurora` downloads `aurora` |
| Q6 effort | Fixed to `medium` internally, and **never shown to the user**. No effort control, no effort text |

## Verified workspace facts

The live workspace root is `/Users/erendt/BRI` with **38 repositories**. These are real observations,
not assumptions, and every selection rule below is anchored to them.

### The two test repositories

`atlas` and `aurora`, the ones named for testing, are both **Go + Gin + Google Wire**:

```text
go.mod          github.com/gin-gonic/gin v1.10.0
                github.com/google/wire v0.6.0
src/wire.go         provider declarations
src/wire_gen.go     the generated constructors that actually wire handlers
app/wire_gen.go
src/<feature>/      routes.go delivery.go service.go repository.go dto.go dao.go
                    interface.go const.go mocks/ *_test.go
```

A real route block from `atlas/src/manage_user/routes.go`:

```go
func (routes *Routes) Register(r *gin.Engine) {
    route := r.Group("/v1/user")
    route.GET("/level", routes.delivery.GetAllLevelUser)
    route.POST("/create", routes.delivery.CreateUser)
    route.GET("/detail/:id", routes.delivery.GetUserByID)
}
```

`aurora` additionally nests groups, so the prompt must combine prefixes:

```go
route := r.Group("/v1")
{
    programMdr := route.Group("/program-mdr")
    {
        ...
    }
}
```

**Trap:** the same route strings also appear in `delivery_test.go`. The selector must exclude test
files or every endpoint is emitted twice.

### `src/vendor` exists in exactly one repository

```text
panama/src/vendor      <- the case named in the request, must be INCLUDED
```

### Framework vendors that must stay excluded

```text
merchant-service-laravel/vendor
micro-user-mms/vendor
mocash_api/vendor
service_qris/vendor
micro-cm-edc/library/vendor
fe-mms/node_modules
```

### Framework signatures across the workspace

| Signature | Repositories | Treatment |
|---|---|---|
| Go + Gin + Wire | atlas, aurora, and the other Go services | include |
| Laravel (`artisan` + `app/`) | merchant-service-laravel, micro-mms, micro-mocash, micro-user-mms, mocash_api, service_qris | include |
| Node (`package.json` + `app.js`) | mms-api | include when routes exist |
| CodeIgniter 3 (`application/` + `system/`) | **mms** | **exclude**, the monolith named in the request |
| Vue frontend (`"vue"` in package.json) | **fe-mms** | **exclude**, no endpoints |

The indexer's `walker.go` skips `vendor` unconditionally, so it cannot be reused. The export gets
its own selector and `walker.go` is not touched.

## Completeness requirement

The export must never "succeed" with an empty or hollow collection. Two mechanisms enforce it.

**In the prompt:** every key the model identifies must be emitted even when its value is unknown.
An unknown value is an empty string, never a missing key. Naming the key is mandatory; filling it is
not.

**In the pipeline:** a normaliser then a validator.

Normaliser, which repairs structure without inventing data:

```text
collection.info.name      <- "<service> API" when missing
collection.info.schema    <- the Postman v2.1 schema URL
item.request.method       <- "GET" only when truly absent
item.request.header       <- [] when absent
item.request.url.host     <- ["{{baseUrl}}"] when absent
item.request.url.path     <- derived from url.raw
item.response             <- [] when absent
```

Validator, which refuses the export when any of these fail:

```text
at least one request exists                                  else 422 "no endpoints found"
every request has a non-empty name                           else 422
every request has a method and a non-empty url.path          else 422
every request has a header array                             else 422
every query parameter has a key                              else 422
every body field has a key when a body is present            else 422
```

The frontend shows the request count before offering the download, so a hollow result is visible
immediately rather than after the user opens the file.

## Design

### Git operations, shared shape

```text
GET  /branches                     -> { branches: [{ name, is_current, updated_relative }] }
POST /checkout  { branch }         -> { branch }
POST /fetch                        -> { ahead, behind }
POST /pull                         -> { branch, updated_relative }
POST /stash                        -> { stash_count }
POST /discard                      -> { restored_count }
any failure                        -> { error } with 400, 404, 409, or 504
```

`checkout` requires the requested name to be an **existing local branch**. It is validated
syntactically first, then confirmed against `refs/heads`. A revision expression such as `HEAD`, a
tag, a commit SHA, or `refs/heads/x` is refused with 404, because git would accept one and leave
the repository on a detached HEAD while the endpoint promised a branch switch. Adversarial review
found this after the first implementation had already passed its tests.

Helper paths live at `127.0.0.1:5199`; the same paths sit under `/api/workspace` on the backend.
Every call carries `root` and `name`. Every handler resolves the repository with
`ResolveRepoPath`, validates the branch against a strict pattern, and calls the existing `runGit`,
which keeps the no-shell argument array, the context timeout, and `GIT_TERMINAL_PROMPT=0`.

`checkout` refuses a dirty tree with 409 and a typed reason instead of carrying uncommitted work
across branches. `discard` runs `git checkout -- .` only.

### Branch picker

`BranchPicker.vue` replaces the non-interactive `.branch-pill`.

- Opens on click, choosing dropup or dropdown by comparing the space below the trigger with the
  popup height, so a row near the bottom opens upward.
- Current branch pinned first, then up to five rows in a scrollable list.
- Search field filters locally; the list is fetched once per open and cached briefly.
- Closes on `Escape`, outside click, and selection, returning focus to the trigger.
- Keyboard operable with an accessible name.

### Kebab action menu

`RepositoryActionsMenu.vue` replaces the bare expand chevron with a `...` trigger.

| Action | Icon intent | Confirm |
|---|---|---|
| Pull | arrow down into a tray | no |
| Fetch | circular refresh arrows | no |
| Stash changes local | box with a downward arrow | no |
| Discard changes | counter-clockwise arrow with a cross | **yes** |
| Details | chevron | no |

The discard modal closes on Cancel, `Escape`, and backdrop, and only proceeds on an explicit
confirm. While an action runs the row shows a busy state and the menu is disabled.

### Export flow

```text
frontend                    helper or backend                provider
   |  pick model                  |                             |
   |----------------------------->|                             |
   |      event: stage             | 1 resolve repo path         |
   |<-----------------------------| 2 select files              |
   |      event: stage             | 3 classify service          |
   |<-----------------------------| 4 build prompt              |
   |      event: stage             | 5 call model                |
   |<-----------------------------|---------------------------> |
   |      event: result            |<--------------------------- |
   |<-----------------------------| 6 normalise + validate      |
   |  build blob, open download modal, revoke the object URL     |
```

The response is server-sent events, reusing the streaming shape the agent path already uses, so a
long model call reports progress instead of blocking silently. The final `result` event carries the
validated collection. The browser turns it into a `Blob`, and the modal offers
`<service>-postman-collection.json`. **The server never writes the file.**

### Model list and locking

The modal reads the configured providers, the same signal `useProviderConfiguration` already
exposes, rather than the old hardcoded provider list in `model-preferences.ts`.

- Lists models whose provider has a personal key or a system fallback.
- Badges models that cost nothing under their plan, such as `space-bunny-free`.
- **With no provider key at all, the export is locked.** Recommended presentation: keep
  "Choose AI model" visible but disabled, with a short reason and a link to Settings. Disabling
  beats hiding: the feature stays discoverable, the user learns why it is unavailable, and no
  extra conditional rendering logic is needed. Hiding also makes the panel look broken beside the
  other rows.

### File selection rules

Include:

- `src/**`, and specifically `src/vendor/**` because the request names it. `panama` is the only
  repository affected.
- `routes.go`, `delivery.go`, `service.go`, `repository.go`, `dto.go`, `dao.go`,
  `interface.go`, `const.go` and their PHP and Node equivalents.
- `wire.go` and `wire_gen.go`, since Wire keeps the real constructor wiring in the generated file.
- `main.go`, framework bootstrap, and route registration files.

Exclude:

- `*_test.go`, `tests/`, `test/`, `*_test.*`, `*.spec.*`, and `mocks/`, which would otherwise
  duplicate every route, as `delivery_test.go` proves.
- Framework vendor: root `vendor/`, any `vendor/` outside `src/`, `node_modules/`,
  `library/vendor/`.
- Build and tooling output: `dist/`, `build/`, `out/`, `target/`, `coverage/`, `tmp/`, `logs/`,
  `storage/`, `bootstrap/cache/`.
- VCS and editors: `.git/`, any dot-directory, `.idea/`, `.vscode/`.
- Non-source assets: `public/`, `assets/`, `resources/views/`, `*.html` at the repository root,
  which is what makes the `mms` monolith unusable.
- Files over the size cap, binaries, and symlinks.

### Service classification

```text
exclude  CodeIgniter 3 signature: application/ with system/      (mms)
exclude  package.json declares vue, react, next, nuxt,
         angular, or svelte with no server framework              (fe-mms)
exclude  no route or handler definition in the selected files
include  otherwise
```

The classifier returns a reason and the UI shows it, rather than skipping a repository silently.

### Prompt pack

Shared system prompt:

```text
You extract HTTP API endpoints from a source repository and emit one Postman Collection v2.1
document describing them.

You will receive a file manifest and the content of selected files. Read them together: a route is
often declared in one file and implemented in another.

Output contract:
1. Emit JSON only. No prose, no Markdown fence, no explanation.
2. Emit one Postman Collection v2.1 object with info, item, and variable.
3. Emit one request per endpoint, carrying method, url with :param placeholders, query parameters,
   headers, and the request body when the handler reads one.
4. Completeness is mandatory. For every key you identify, emit the key even when you do not know
   its value. An unknown value is an empty string. NEVER omit a key you identified.
5. Group requests by resource or controller into folders.
6. Never invent an endpoint you did not see. A route whose handler you cannot find is still emitted,
   with the description "handler not found".
7. Use {{baseUrl}} as the host variable.
8. Keep the description to one line, naming the source file.
```

Framework appendix, selected by detection:

| Detected | Instructions added |
|---|---|
| Go net/http | `http.HandleFunc`, `mux.Handle`, method-aware patterns |
| Gin | `r.Group` prefixes, including nested groups, combined with each `route.METHOD` path |
| Fiber | `app.Get/Post`, `Group`, `app.Use` |
| Echo | `e.GET/POST`, `e.Group`, bind targets |
| PHP plain | `$_GET`, `$_POST`, front-controller routing |
| Laravel | `routes/api.php`, `routes/web.php`, `Route::apiResource`, controller actions, form requests |
| Lumen | `routes/web.php`, `$router->get/post`, no `api.php` split |
| Node.js | Express `app.get/post`, router mounts, Koa and Fastify equivalents |
| Google Wire | read `wire_gen.go` for constructed handlers; `wire.go` only declares providers |

The vendor rule is stated in the prompt as well: files under `src/vendor` are first-party service
code and must be read; files under any other `vendor`, `node_modules`, or framework bundle must be
ignored.

### Test repository state, measured

| Repository | Current branch | Dirty entries | Local branches | Upstream |
|---|---|---|---|---|
| atlas | feature/CRMMS-77911-Enhancement-Manage-User | **6** | 8 | set |
| aurora | feature/CRMMS-69759-Maintenance-MDR-Program | **0** | 20 | set |

Three consequences for testing:

1. `atlas` being dirty makes it the natural case for the checkout refusal. A checkout against it must
   return 409 and must not move the branch.
2. `aurora` being clean with 20 branches exercises both the real switch and the five-row limit with
   search.
3. **Both repositories belong to the user.** A test that clicks checkout on a real repository would
   move their working branch, and a test that clicks discard would destroy their uncommitted work.
   Destructive paths must therefore run only against a scratch repository created in a temp
   directory. Against atlas and aurora the suite may exercise reads, the picker, and the refusal
   path, and must never confirm a discard or complete a switch.

Fetch and pull both reach `bitbucket.bri.co.id` and need the corporate VPN. Without it the correct
observable behaviour is a clear 504 or network error, which is what the suite should assert rather
than treating as a failure.

## Phases and agents

Three phases, at most three agents each, QA included.

### Phase 1 - Git actions and the branch picker

| Agent | Owns | Gate |
|---|---|---|
| 1A backend | branch list and the five actions on helper and server, with safety and tests | decisions confirmed |
| 1B frontend | `BranchPicker.vue`, `RepositoryActionsMenu.vue`, discard modal, API client, row state | frozen contract |
| 1Q QA | build, unit tests, e2e, destructive-path review | 1A and 1B |

### Phase 2 - Export pipeline

| Agent | Owns | Gate |
|---|---|---|
| 2A backend | selector, classifier, prompt pack, provider call, normaliser, validator, SSE endpoint on both sources | prompt contract frozen |
| 2B frontend | modal on the configured model catalogue, lock state, SSE progress, download modal, no effort text | frozen contract |
| 2Q QA | selector and classifier against atlas, aurora, panama, fe-mms, and mms; hollow-output rejection; real export | 2A and 2B |

### Phase 3 - Hardening, docs, end-to-end

| Agent | Owns | Gate |
|---|---|---|
| 3A frontend | download modal polish, error and retry states, empty state, accessibility | 2Q |
| 3B docs | `guide.md` feature table and change log, `plan/BACKLOG.md` | 3A |
| 3Q QA | full-page e2e on atlas and aurora, keyboard-only pass, both themes, regression run | 3A and 3B |

## Verification

- Each git action runs against a scratch repository with a known state, including the dirty-tree
  refusal.
- Branch names carrying `;`, a space, `..`, or a leading `-` are rejected, proven by test.
- Repository names carrying a separator, `..`, or a null byte are rejected, proven by test.
- Discard cannot run without an explicit confirm, and restores tracked files only.
- **Selector:** `panama` includes `src/vendor`; `merchant-service-laravel`, `micro-user-mms`,
  `mocash_api`, `service_qris`, and `fe-mms` exclude their vendors; no repository includes a
  `*_test.go` or `mocks/` file.
- **Classifier:** `mms` is refused as a CodeIgniter monolith, `fe-mms` as a Vue frontend, and both
  name the reason.
- **Export of atlas and aurora:** every route in `src/*/routes.go` appears exactly once, prefixes
  include nested groups, and each emitted route traces to its source file.
- **Completeness:** every request carries a name, method, url with a path, a header array, and query
  and body keys. A collection with zero requests is refused with 422, and a fixture response that is
  valid JSON but hollow is refused too.
- A model response wrapped in a Markdown fence is unwrapped; one that is not JSON at all fails
  loudly.
- No file is written on the server, proven by asserting the export directory does not exist
  afterwards.
- The downloaded file imports into Postman.
- `go build ./...`, `go test ./...`, `vue-tsc`, and the Playwright suites stay green.

### Ground truth for the atlas and aurora export

Counted from the real repositories, so the export can be checked against a number rather than a
feeling:

| Repository | Routes declared in `src/**/routes.go` | Route declarations inside `*_test.go` | Group calls | Wire files |
|---|---|---|---|---|
| atlas | 30 | **9** | 7 | 3 |
| aurora | 54 | **19** | 9 | 3 |

Two consequences. First, an export of atlas must yield about 30 requests and aurora about 54; a
result near 39 or 73 means test files leaked into the selection and every route from a test file was
emitted a second time. Second, the group counts confirm that `Group()` prefixes must be resolved,
including the nested groups aurora uses, or the emitted paths will be wrong even when the count is
right.

## Risks

| Risk | Mitigation |
|---|---|
| A route appears twice because tests mirror it | test and mock paths excluded, asserted against atlas |
| `src/vendor` is dropped by a reused indexer rule | the export has its own selector; `walker.go` untouched; panama is a test case |
| A hollow but valid JSON passes | normaliser plus a validator that refuses zero or incomplete requests |
| The model omits keys it cannot fill | the prompt states keys are mandatory and values may be empty |
| A crafted branch name reaches git | strict validator plus `--` separation, hostile input tested |
| A pull or fetch hangs on credentials | `GIT_TERMINAL_PROMPT=0` and the existing command timeout |
| Writing inside a repository dirties it | nothing is written at all |
| A monolith blows the context window | file and byte caps, and CI3 is refused |
| Long exports look frozen | SSE stage events, reusing the agent stream shape |
| Two sources drift | one shared command module and one shared export pipeline, thin handlers |

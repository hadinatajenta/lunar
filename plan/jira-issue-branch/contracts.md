# Contracts

Frozen. An agent that needs a field that is not here reports it to the parent instead of inventing
one. Nothing in this document is decided locally by a subagent.

## 1. Surfaces

Every route exists twice, exactly like the existing git and export routes.

| Surface | Prefix | Auth |
|---|---|---|
| Helper | route at the server root, for example `/issues/{key}/branches` | `requireToken` bearer |
| Backend | the same name under `/api/workspace` or `/api/jira` | session |

The helper resolves the workspace root from `?root=` and the repository name from `?name=`, in its
existing style. The backend takes the root from saved workspace settings and ignores a body root.

## 2. Shared conventions

### Error body

Every non-2xx response is `{"error": "<readable sentence>"}`. No stack traces, no internal paths, no
upstream URLs.

### Status codes

| Code | Meaning here | Existing precedent |
|---|---|---|
| 400 | Malformed input: an invalid issue key, an invalid branch name, a root that is not absolute | branch validation already returns 400 |
| 401 | No PAT, or a PAT Jira or Bitbucket rejected | `invalid or expired Jira PAT` |
| 403 | The workspace root is outside `LUNAR_WORKSPACE_ALLOWED_ROOTS` | the Phase 1 write guard |
| 404 | The repository does not exist, or the branch does not exist | `the branch does not exist in this repository` |
| 409 | The operation conflicts with current state: the branch already exists, or the tree is dirty | the Phase 1 checkout guard |
| 422 | The request is understood and cannot be satisfied: no workspace is configured, the workspace is not usable | the export classifier |
| 502 | Jira or Bitbucket could not be reached | `unable to reach Jira BRI API. Please check your VPN connection` |
| 504 | An upstream call exceeded its deadline | the Phase 1 git timeout |

### Constants, defined once on the backend

| Constant | Value | Why |
|---|---|---|
| Local scan cache TTL | 30 seconds | Branches change slowly, and a page can request the same scan several times |
| Bitbucket scan cache TTL | 10 minutes | The expensive tier, and branches do not move that fast |
| Bitbucket concurrency | 4 in flight | Only the candidate repositories are ever checked, so the pool stays small |
| Remote candidate limit | **5 repositories** | The authoritative check is scoped to the suggestion's top candidates, not the whole project. Checking all 65 was rejected: the cost is 65 calls against a benefit that only matters in the repositories that were plausible in the first place |
| Bitbucket request timeout | 20 seconds | Matches the raised Jira client timeout |
| Git scan timeout | 5 seconds per repository | Matches `GitCommandTimeout` |
| Key pattern | `(?i)\b[A-Z][A-Z0-9]{1,9}-[0-9]{1,6}(?![0-9])` | The negative lookahead is mandatory: `CRMMS-6397` must not match `CRMMS-63970` |
| Trusted base branch names | `master`, `main`, `release/production`, `develop`, `development` | A remote default outside this set is shown but flagged for confirmation, because it is unusual rather than necessarily wrong |

## 3. Endpoints

### 3.1 Workspace readiness

```
GET /readiness

200 {
  "state": "ready",
  "root_path": "/Users/erendt/BRI",
  "message": "Ready",
  "local_repository_count": 36,
  "remote_repository_count": 65,
  "missing_repository_count": 29,
  "missing_repositories": ["administrasi-service", "elbrus"]
}

state is exactly one of:
  "not-configured"  no path set
  "invalid"         refused, see message
  "empty"           valid path, no repositories inside
  "single-repo"     the path is itself a repository
  "foreign"         branches carrying Jira keys exist, and none of those keys belong to the
                    Jira project currently in view. A workspace with no keyed branches yet is
                    NOT foreign, because a fresh workspace is a normal starting state
  "partial"         fewer repositories locally than the projects hold
  "ready"           repositories present and at least one carries a key from this project

missing_repositories is capped at 50 entries and sorted, so the payload cannot grow without bound.
```

### 3.2 Branches for one issue

```
GET /issues/{key}/branches

200 {
  "key": "CRMMS-77911",
  "matches": [
    {
      "repo": "mms-api",
      "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
      "source": "both",
      "is_current": false,
      "is_cloned": true,
      "bitbucket_project": "NME",
      "last_commit": "6a1f0c9d2b7e4f8a1c3d5e7f9a0b2c4d6e8f0a1b",
      "last_commit_relative": "3 days ago",
      "ahead": 4,
      "behind": 0
    }
  ],
  "scope": {
    "local_repository_count": 36,
    "remote_repository_count": 65,
    "coverage_percent": 55,
    "verified_remotely": true,
    "remote_repositories_checked": 5
  }
}

source is exactly one of:
  "local"     found by the local scan only
  "remote"    found on Bitbucket among the candidate repositories, and the repository is NOT cloned
  "both"      found by both

remote_repositories_checked is how many repositories were actually asked of Bitbucket. It is at
most the remote candidate limit. **The screen must never imply that all 65 were checked.**

last_commit, last_commit_relative, ahead and behind are null when the source is "remote",
because they require a clone.

400  the key does not match the key pattern
401  no PAT
502  Jira or Bitbucket unreachable and neither could answer
```

A key with no match returns `"matches": []` and 200. **Absence is information, not an error.**

### 3.3 Branches for a whole sprint

```
GET /sprint/branches?squad=&board=

200 {
  "matches": { "CRMMS-77911": [ ...same shape as 3.2... ], "CRMMS-77745": [] },
  "scope": { ...same shape as 3.2... }
}
```

One scan serves every key. **An implementation that scans once per key is wrong**, because a sprint
has seven issues and the workspace has thirty six repositories.

### 3.4 Repository suggestions

```
GET /issues/{key}/repository-suggestions

200 {
  "key": "CRMMS-77911",
  "confidence": "high",
  "suggestions": [
    {
      "repo": "mms-api",
      "score": 9.0,
      "is_cloned": true,
      "bitbucket_project": "NME",
      "reasons": [
        { "signal": "epic_siblings", "weight": 5.0, "strength": 0.8, "text": "2 other issues in this epic were worked in mms-api" },
        { "signal": "explicit_hint", "weight": 4.0, "strength": 1.0, "text": "the issue names mms-api" }
      ]
    }
  ],
  "signals_unavailable": [
    { "signal": "code_index", "reason": "the repository is not cloned, so its files cannot be searched" }
  ],
  "scope": { ...same shape as 3.2... }
}

confidence is exactly one of "high", "medium", "low".
signal is exactly one of "epic_siblings", "explicit_hint", "code_index", "confluence", "model".
suggestions is ordered by score descending and capped at 3 entries.
Every suggestion MUST carry at least one reason. A suggestion with no reason is a defect.

400  the key does not match the key pattern
422  no workspace is configured
502  Jira unreachable
```

### 3.5 Create a branch

```
POST /branches
body { "repo": "mms-api", "branch": "feature/CRMMS-77911-Enhancement-Manage-User", "base_branch": "master", "remember_base": true }

base_branch is REQUIRED. There is no default and there is no fallback. A request without it is
refused with 400, so that no branch can ever be created from a base the user did not see.
remember_base is optional and defaults to false.

201 {
  "repo": "mms-api",
  "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
  "base_branch": "master",
  "head_commit": "6a1f0c9d2b7e4f8a1c3d5e7f9a0b2c4d6e8f0a1b"
}

The response is 201, not 200, because a branch was created.

**Before creating, the backend checks the named repository on Bitbucket for a branch carrying this
key.** This is one call, and it is the last line of defence against a duplicate: it catches the case
where the branch already exists on the remote but the repository is not cloned locally, which the
local scan cannot see. If a match is found the request fails with 409 naming the existing branch,
rather than creating a second one.

400  invalid branch name
403  the workspace root is outside the allow list
404  the repository does not exist
409  the branch already exists
409  the working tree has uncommitted changes
502  git failed
504  git timed out
```

### 3.6 Branch name suggestion

```
GET /issues/{key}/branch-name

200 {
  "key": "CRMMS-77911",
  "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
  "source": "generated"
}

source is "generated" when the name came from the summary, or "existing" when a branch for this key
already exists and its name is being echoed back.
The generated name is ALWAYS editable in the UI. It is never applied without the user seeing it.

**Generation rule.** These summaries are written in Indonesian with an English intent clause, for
example `Sebagai Maker & Approval di New MMS, I want to memiliki menu dan form terpisah untuk
mengajukan program bulk MDR QRIS Statis`. Taking the first words produces
`feature/CRMMS-77745-Sebagai-Maker-Approval`, which is useless.

| Step | Detail |
|---|---|
| 1 | If the summary contains `i want to`, take everything after it. That clause is the actual intent and the preamble before it is a role description |
| 2 | Otherwise use the whole summary |
| 3 | Split on anything that is not a letter or a digit |
| 4 | Drop filler words, including `sebagai`, `di`, `dan`, `untuk`, `yang`, `dengan`, `memiliki`, `mengajukan`, `pada`, `dapat`, and the English equivalents |
| 5 | Keep at most 5 words |
| 6 | Uppercase a word that is already all capitals or digits, otherwise capitalise its first letter |
| 7 | Prefix with `feature/<KEY>-` |

Measured against the real summaries this yields
`feature/CRMMS-77745-Menu-Form-Terpisah-Program-Bulk`,
`feature/CRMMS-77740-Melihat-Mengubah-PN-Pengelola-Form`, and for the summary
`Enhancement Manage User` it yields `feature/CRMMS-77911-Enhancement-Manage-User`, which is
byte-identical to the branch that actually exists in the workspace.
```

### 3.7 Base branch options for a repository

**The base branch is chosen by the user. It is never resolved automatically.**

This was originally designed as an automatic resolution with a priority list. Measurement killed that
design, and the reasoning is recorded here so it is not reintroduced.

| What was measured | Result |
|---|---|
| Local `origin/HEAD` versus what the remote declares | Both said `main` for `aurora`, and the owner of that repository states its production branch is `master`. The authoritative sources agree with each other and are still wrong |
| Bitbucket `/default-branch` | Answers for every repository including the ones never cloned, but is a repository setting that can be stale, as `aurora` shows: `main` is empty there while `master` has 822 commits |
| "Which branch do this repository's feature branches descend from" | **Disagrees with Bitbucket's default in 10 of 23 repositories**, and for `aurora` it answers `development` while the team's production branch is `master` |

The last row is the decisive one. The data contradicts itself: aurora's feature branches really are
cut from `development`, and the team still considers `master` production. Which branch a new feature
should come from is a **team convention**, not a fact stored in the repository, so no algorithm can
derive it. An automatic choice would be wrong for a large share of repositories and silently so.

```
GET /repos/{repo}/base-branch-options

200 {
  "repo": "aurora",
  "is_cloned": true,
  "remembered": null,
  "bitbucket_default": "main",
  "options": [
    { "name": "master", "last_commit_relative": "2 weeks ago", "source": "trusted" },
    { "name": "development", "last_commit_relative": "4 days ago", "source": "trusted" },
    { "name": "main", "last_commit_relative": null, "source": "bitbucket_default" },
    { "name": "pre/development", "last_commit_relative": "7 weeks ago", "source": "trusted" }
  ],
  "other_branch_count": 90
}

source is exactly one of "remembered", "bitbucket_default", "trusted", "other".
remembered is the base the user previously chose for this repository, or null.
other_branch_count is how many further branches exist behind a "show all" control.
```

**Ordering is a hint, not a decision.** Options are ordered: the remembered choice first, then the
Bitbucket default, then the trusted names present in the repository by most recent commit, and never a
`feature/` or `bugfix/` branch, because a new feature is not cut from another feature.

**Staleness is shown, not hidden.** Each option carries its last commit, so `main` being empty is
visible at the moment of choosing rather than discovered later.

The base is **remembered per repository** once chosen, so the choice is made once and confirmed
thereafter.

## 4. Frontend types

These are the exact shapes. Do not add fields without asking.

```ts
export type WorkspaceReadinessState =
  | "not-configured" | "invalid" | "empty" | "single-repo" | "foreign" | "partial" | "ready"

export interface WorkspaceScope {
  local_repository_count: number
  remote_repository_count: number
  coverage_percent: number
  verified_remotely: boolean
  remote_repositories_checked: number
}

export interface WorkspaceReadiness {
  state: WorkspaceReadinessState
  root_path: string
  message: string
  local_repository_count: number
  remote_repository_count: number
  missing_repository_count: number
  missing_repositories: string[]
}

export interface IssueBranchMatch {
  repo: string
  branch: string
  source: "local" | "remote" | "both"
  is_current: boolean
  is_cloned: boolean
  bitbucket_project: string
  last_commit: string | null
  last_commit_relative: string | null
  ahead: number | null
  behind: number | null
}

export interface BaseBranchOption {
  name: string
  last_commit_relative: string | null
  source: "remembered" | "bitbucket_default" | "trusted" | "other"
}

export interface RepositoryBaseBranchOptions {
  repo: string
  is_cloned: boolean
  remembered: string | null
  bitbucket_default: string | null
  options: BaseBranchOption[]
  other_branch_count: number
}

export interface IssueBranches {
  key: string
  matches: IssueBranchMatch[]
  scope: WorkspaceScope
}

export interface SprintBranches {
  matches: Record<string, IssueBranchMatch[]>
  scope: WorkspaceScope
}

export interface SuggestionReason {
  signal: "epic_siblings" | "explicit_hint" | "code_index" | "confluence" | "model"
  weight: number
  strength: number
  text: string
}

export interface RepositorySuggestion {
  repo: string
  score: number
  is_cloned: boolean
  bitbucket_project: string
  reasons: SuggestionReason[]
}

export interface RepositorySuggestions {
  key: string
  confidence: "high" | "medium" | "low"
  suggestions: RepositorySuggestion[]
  signals_unavailable: { signal: string; reason: string }[]
  scope: WorkspaceScope
}
```

## 5. UI states and exact copy

Copy is frozen so that two agents cannot produce two different sentences for the same state.

### Workspace readiness, shown on both the Jira page and the Microservices page

| state | Rendered message |
|---|---|
| not-configured | "No workspace folder is configured. Choose the folder that holds your service repositories." |
| invalid | the backend `message`, shown verbatim |
| empty | "No repositories were found in {root_path}." |
| single-repo | "{root_path} is itself a repository. Point Lunar at the folder that contains your repositories instead." |
| foreign | "The repositories in {root_path} carry branches for other Jira projects, but none for the project you are viewing." |
| partial | "Searching {local_repository_count} of {remote_repository_count} repositories. {missing_repository_count} are not cloned locally." |
| ready | nothing is shown |

### Branches for an issue

| Situation | Rendered |
|---|---|
| matches found locally | the branch list, each row naming its repository |
| a match whose source is remote | "in {repo}, which is not in your workspace" plus a clone hint |
| no matches, scope complete | "No branch found for {key} in the {local_repository_count} repositories Lunar can see." |
| no matches, scope partial | "No branch found in the {local_repository_count} of {remote_repository_count} repositories Lunar can see. {missing_repository_count} are not cloned locally." |
| no matches, remote checked | "No branch found in the {local_repository_count} repositories Lunar can see, or in the {remote_repositories_checked} most likely repositories checked on Bitbucket." |
| no matches, Bitbucket unreachable | "No branch found locally. Bitbucket could not be reached, so this is not conclusive." |

**"does not exist" must never be rendered.** Only "not found in N repositories".

### Repository suggestion

| Situation | Rendered |
|---|---|
| confidence high | the top repository preselected, with its reasons, styled as a recommendation |
| confidence medium | the top three, nothing preselected, with their reasons |
| confidence low | the top three, nothing preselected, with their reasons and the sentence "Lunar is not confident. The evidence is weak." |
| signals_unavailable not empty | each unavailable signal listed with its reason |
| no suggestions | "Lunar could not suggest a repository. Choose one yourself." |

A low confidence suggestion **must not** use the same visual treatment as a high confidence one.

### Create branch

**The flow is: branch name, then repository, then base branch, then create.** The name is generated
from the issue and is independent of the repository, so it comes first. The repository determines which
base branches exist, so it comes before the base. Nothing is created until all three are confirmed.

The confirmation states the repository, the branch name, the base branch, and how many repositories
were searched. The name field is pre-filled and editable.

| Element | Detail |
|---|---|
| Branch name | Pre-filled from the issue, editable, validated on submit |
| Repository | Pre-filled from the suggestion when confidence is high, otherwise empty and required |
| Base branch | **Never pre-filled automatically.** The picker lists the options from 3.7 with each one's last commit, and the user chooses |
| Remember | An unchecked checkbox: "remember this base for {repo}" |
| Create | Disabled until repository and base branch are both chosen |

## 6. Component ownership

| Component | Owns |
|---|---|
| `WorkspaceReadinessPanel.vue` | Every readiness state, used by both pages. One implementation, not two |
| `IssueBranchList.vue` | The branch rows for one issue, including the remote-only row treatment |
| `RepositorySuggestionList.vue` | The suggestions, the reasons, the confidence styling, and the unavailable signals |
| `CreateBranchModal.vue` | The confirmation, the editable name, and the base branch |
| `useIssueBranches.ts` | Fetching and caching branches for a sprint, one call for the whole sprint |
| `useRepositorySuggestion.ts` | Fetching suggestions for one issue |
| `useWorkspaceReadiness.ts` | Fetching readiness once and sharing it |

## 7. Decisions that must not be made locally

Each of these has been decided above, and a subagent changing one is a defect.

1. Never render "does not exist" for a branch. Only "not found in N repositories".
2. Never create a branch without an explicit confirmation.
3. Never preselect a repository when confidence is medium or low.
4. Never render a suggestion without at least one reason.
5. Never call Bitbucket per issue. One scan serves the whole sprint.
6. Never call Bitbucket on every render. Use the cache TTLs in section 2.
7. Never treat a repository that is not cloned as "no work started".
8. Never fall back to seed or sample data. Absence is shown as absence.
9. Never show a bare "no branch" when coverage is below 100%. The scope is always stated.
10. Never let the model be the only signal. Its reasoning is shown beside the others, not instead of
    them.
11. Never scan once per key. One scan, then a map.
12. Never invent a status code outside section 2.
13. Never add a response field the frontend does not need, and never remove one it does.
14. Never write to disk. The only mutation in this whole feature is the one branch creation.
15. Never check more than the remote candidate limit of repositories on Bitbucket, and never imply
    that the whole project was checked. The one exception is the single repository named in a create
    request, which is always verified.
16. Never report a workspace as foreign merely because it has no keyed branches yet. Foreign requires
    that keyed branches exist and none of them belong to the project in view.
17. **Never resolve the base branch automatically.** No default, no fallback, no priority list. The
    user chooses it. Measurement showed the local default, the Bitbucket default, and the branch the
    feature branches descend from all disagree, and for aurora every automatic answer was wrong.
18. Never send a create request without an explicit base_branch, and never create a branch before the
    base has been shown and chosen.
19. Never offer a `feature/` or `bugfix/` branch as a base. A new feature is not cut from another
    feature.
20. Never hide staleness when offering a base. Every option shows its last commit, so an empty branch
    is visible before it is chosen rather than after.

## 8. Revisions from design review

Three corrections, all measured against the real instance.

### 8.1 One issue spans several repositories

A new endpoint is built across three layers, in this order: the backend service creates the feature and
its endpoint, the API gateway exposes it, and the frontend consumes it. A single issue therefore
produces **up to three branches in three repositories**, not one.

The suggestion returns a **set** of repositories, each with a role.

```
role is exactly one of: "frontend", "gateway", "backend", "unknown"
```

```
GET /issues/{key}/repository-suggestions
200 {
  "key": "CRMMS-77911",
  "confidence": "high",
  "suggestions": [
    { "repo": "fe-mms",        "role": "frontend", "score": 7.0, "is_cloned": false, "reasons": [...] },
    { "repo": "mms-api",       "role": "gateway",  "score": 6.0, "is_cloned": true,  "reasons": [...] },
    { "repo": "micro-user-mms","role": "backend",  "score": 9.0, "is_cloned": true,  "reasons": [...] }
  ]
}
```

**The role is read from the acceptance criteria, not guessed.** Measured on the real descriptions:
CRMMS-77745 names the routes `/mdr-maintenance/menu-input-perubahan-mdr` and
`/input-perubahan-mdr-qris-statis`, which are frontend paths, and the columns
`mdr_qris_statis_on_us` and `mdr_qris_statis_off_us`, which are backend fields. One description
therefore identifies both layers.

### 8.2 Create spans several repositories, so partial failure is normal

```
POST /branches
body {
  "key": "CRMMS-77911",
  "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
  "targets": [
    { "repo": "fe-mms",         "base_branch": "release/development", "remember_base": false },
    { "repo": "mms-api",        "base_branch": "release/production",  "remember_base": true },
    { "repo": "micro-user-mms", "base_branch": "development",         "remember_base": false }
  ]
}

207 {
  "key": "CRMMS-77911",
  "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
  "created": [ { "repo": "mms-api", "base_branch": "release/production", "head_commit": "..." } ],
  "failed":  [ { "repo": "fe-mms", "reason": "the repository is not cloned" } ],
  "skipped": [ { "repo": "micro-user-mms", "reason": "the branch already exists" } ]
}
```

**207, not 201**, because the outcome is per target. A single status code cannot describe a request
that partly succeeded.

| Rule | Detail |
|---|---|
| Never roll back | A branch that was created stays created. Deleting it to make the response tidy is destructive and is not this feature's decision |
| Report every target | Each target appears in exactly one of created, failed, or skipped. A target that is silently missing is a defect |
| Continue past a failure | One unclonable repository does not stop the other two |
| An existing branch is skipped, not failed | It is not an error that work already started |
| base_branch is required per target | There is no default. 400 when any target omits it |

### 8.3 The base branch and the repository are chosen in a dropdown, not a modal

The design originally put the whole flow in a modal with steps. It is rejected: it hides the issue
behind a scrim and it makes a per-repository choice feel like a wizard.

| Element | Behaviour |
|---|---|
| Base branch | A trigger showing the chosen branch or "Choose a base branch". Clicking opens an **anchored popup** with a **search field** at the top, then the options. Same pattern as the AI model picker already in the application |
| Repository | The same popup pattern, with search |
| Placement | Dropup when there is not enough room below, dropdown otherwise, measured against the viewport |
| Popup rendering | **Teleported to the document body** and positioned fixed. A popup nested inside a styled container inherits its selectors, which is exactly the bug that made the search field 22 pixels wide during design review |
| Keyboard | Escape closes and returns focus to the trigger. Arrow keys move the active option. Enter selects |
| Outside click | Closes |
| Search | Filters by name, keeps its value while the list re-renders, and shows "No branch in {repo} matches {term}" when nothing matches |

### 8.4 The branch name comes from the acceptance criteria

Naming from the summary alone produces `feature/CRMMS-77745-Sebagai-Maker-Approval`, which is
useless. The name is `feature/<KEY>-<Category>-<Core>`.

Measured: for CRMMS-77911 the title is *"Sebagai Administrator, I want to melihat daftar user yang
dikelompokkan berdasarkan Personal Number (PN), mengelola level/jabatan mereka dalam satu halaman
detail, serta melihat riwayat perubahannya"* and the branch that actually exists is
`feature/CRMMS-77911-Enhancement-Manage-User`.

| Part | Source | Note |
|---|---|---|
| Category | **Inferred by the model** | `Enhancement` does not appear anywhere in that issue's summary or description. It is a human judgement about the kind of change, so it cannot be extracted |
| Core | **The model reads the acceptance criteria** | The description contains `*1. Halaman Utama Manage User (Grouping by PN)*`, which is where "Manage User" comes from |

The acceptance criteria live **inside the description**, not in a separate field:
`customfield_15310`, the Acceptance Criteria field, is **null on every issue checked**, while the
descriptions carry `h3. Deskripsi Acceptance Criteria (AC)` and numbered section headings.

```
GET /issues/{key}/branch-name
200 {
  "key": "CRMMS-77911",
  "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
  "category": "Enhancement",
  "core": "Manage-User",
  "source": "model",
  "evidence": "Halaman Utama Manage User (Grouping by PN)"
}

source is exactly one of:
  "model"     the model read the summary, the description and the acceptance criteria
  "fallback"  the model was unavailable or declined, so the keyword rule in 3.6 was used
  "existing"  a branch for this key already exists and its name is echoed back
```

**The evidence field is mandatory.** It names the phrase the name was taken from, so the user can
check the reasoning rather than accept a string.

The name is generated **only when the user opens the create surface for one issue**, never for the
whole sprint, because it costs a model call. It is cached per issue, and a name the user has edited is
never overwritten.

## 9. Additional rules

21. Never assume one repository. A new endpoint spans the backend, the gateway, and the frontend, and
    the suggestion returns all of them with their roles.
22. Never return a single status code for a multi target create. Use 207 and account for every target
    in created, failed, or skipped.
23. Never roll back a branch that was created because a sibling target failed.
24. Never put the base branch or the repository choice behind a modal. Use the anchored popup with a
    search field, teleported to the body.
25. Never nest the popup inside a container that styles inputs. The search field was 22 pixels wide
    because `.target input` matched it.
26. Never lose the search term when the option list re-renders.
27. Never derive the branch name from the summary alone. Read the acceptance criteria inside the
    description, and always return the phrase the name came from.
28. Never generate a branch name for a whole sprint at once. One model call, on demand, per issue.
29. Never overwrite a branch name the user has edited.

### 8.5 The suggestion is a convenience, the manual choice is the ground truth

The first design offered only checkboxes over the suggested repositories. If the suggestion was wrong,
incomplete, or empty, the user had no way forward. That is a dead end, and it is fixed here.

| Control | Behaviour |
|---|---|
| Add another repository | A trigger below the suggestions, always present, opening the same anchored popup with search over **every** repository, local and remote, across every accessible project |
| Role | A manually added repository gets a **role selector**, because the role drives the create order and the grouping. It cycles frontend, api gateway, backend, unknown |
| Source | A manually added row is labelled "added by you" and its reason reads "added by you", so it is never confused with a suggestion |
| Remove | A manual row can be removed. A suggested row is unticked instead, which is a different signal |
| Empty suggestion | When nothing could be suggested, the panel says so and the Add control is the only path. It never presents an empty list without a way forward |
| Not cloned | Shown on the row and warned about before create, but the row can still be planned |

**A repository the catalogue does not contain cannot be added**, because the picker lists real
repositories only. When the user cannot find theirs, the panel states how many the workspace has and
how many Bitbucket has, so the gap is visible rather than mysterious.

### 8.6 What the corrections teach the ranking

Every interaction is a signal, and they are not equal.

| Action | Signal | Weight |
|---|---|---|
| A repository is **added** that was not suggested | Strongest positive | High |
| A suggested repository is **removed** | Strong negative | Medium |
| A suggested repository is **unticked** for this issue | Weak negative, may be scoped to this issue only | Low |
| A suggested repository is **kept** | Weak positive | Low |

An addition outranks everything because it is the only case where Lunar was silent about a repository
that mattered. That is the same principle as the correction outweighing the confirmation in 8.4.

```
GET /issues/{key}/repository-suggestions
200 {
  "key": "CRMMS-77911",
  "confidence": "high",
  "suggestions": [ ... ],
  "catalogue": { "local_repository_count": 36, "remote_repository_count": 65 },
  "signals_unavailable": [ ... ]
}

GET /repositories?search=
200 {
  "repositories": [ { "repo": "micro-cm-qris", "project": "NME", "is_cloned": true } ],
  "local_repository_count": 36,
  "remote_repository_count": 65
}

POST /issues/{key}/repository-feedback
body { "repo": "micro-cm-qris", "action": "added" }
action is exactly one of: "added", "removed", "unticked", "kept"
202 { "recorded": true }
```

Feedback is fire and forget. It never blocks the create flow and it never changes what is on screen.

## 10. Rules for the manual path

30. Never present suggestions without a way to add a repository that was not suggested.
31. Never present an empty suggestion list without the Add control and a sentence explaining it.
32. Never allow a repository that does not exist. The picker lists real repositories only, from every
    accessible project, and states how many there are when the search finds nothing.
33. Never label a manually added repository as a suggestion. It reads "added by you".
34. Never let a not cloned repository block planning, only creating. Warn on the row and before create.
35. Never let feedback block or alter the current screen. It is recorded and shown, nothing more.
36. Never treat an addition and a confirmation as the same evidence. An addition is the stronger
    signal, because it is the case where Lunar was silent.

## 11. Contrast, measured

The design was reviewed and the Add control was reported as hard to see, especially in the dark
theme. It was measured rather than adjusted by eye.

| Element | Dark, before | Dark, after | Light, after | WCAG AA |
|---|---|---|---|---|
| Add repository | **3.42** | 16.38 | 16.30 | 4.5 |
| Section heading | **3.42** | 6.18 | 7.58 | 4.5 |
| Instruction text | **3.42** | 6.18 | 7.58 | 4.5 |

The cause is a single token: `--subtle` is `#626a75` in the dark theme, which is 3.42 to 1 against
`--surface` `#101217`. The light theme uses `#64748b` and reaches 4.76, so **only the dark theme
failed**. This is an application wide token, not a defect in this feature, and it is recorded here
because it was found here.

| Rule | Detail |
|---|---|
| Interactive text never uses `--subtle` | A control must reach 4.5 to 1. The Add control uses `--text` at 16.38 |
| Supporting text uses `--muted` | `--muted` reaches 6.18 in dark and 7.58 in light, so headings and instructions use it |
| `--subtle` is for decoration only | Dividers, disabled states, and the placeholder inside an already focused field |

## 12. Placement

The Add control was centred below the rows inside a dashed border, which read as a separator rather
than a control: centred text looks like a label, and a dashed border looks like a placeholder.

| Before | After |
|---|---|
| Centred, full width, dashed border, `--subtle` text | **Left aligned**, sized to its content, solid border, raised background, `--text` label, `+` prefix |
| Instruction text below it, on its own line | Instruction text **beside** it, on the same row, so the control and its explanation read as one unit |

A control that adds something belongs in the flow, aligned with the content it adds to, not centred
beneath it.

## 13. Rules for visibility

37. Never centre a control that adds content. Align it with the content it adds to.
38. Never style an actionable control with a dashed border or placeholder text. Dashed reads as
    absent, and placeholder colour reads as disabled.
39. Never put interactive text below 4.5 to 1. Measure it, do not judge it by eye.
40. Never use `--subtle` on a control. It reaches only 3.42 to 1 in the dark theme.


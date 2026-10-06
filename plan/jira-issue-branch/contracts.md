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
body { "repo": "mms-api", "branch": "feature/CRMMS-77911-Enhancement-Manage-User", "base_branch": "master" }

201 {
  "repo": "mms-api",
  "branch": "feature/CRMMS-77911-Enhancement-Manage-User",
  "base_branch": "master",
  "head_commit": "6a1f0c9d2b7e4f8a1c3d5e7f9a0b2c4d6e8f0a1b"
}

base_branch is optional. When omitted it is resolved by the rule in 3.7, which never falls back to
whatever happens to be checked out.
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
```

### 3.7 Base branch for a repository

```
GET /repos/{repo}/base-branch

200 {
  "repo": "aurora",
  "base_branch": "main",
  "resolved_from": "remote_default",
  "is_trusted": true,
  "alternatives": ["release/production", "development"],
  "is_cloned": true
}

resolved_from is exactly one of:
  "user_choice"      a remembered choice for this repository
  "local_default"    the local remote HEAD, git symbolic-ref refs/remotes/origin/HEAD
  "remote_default"   Bitbucket's default-branch endpoint
  "convention"       the trusted list, in order, when neither default could be read
  "current_branch"   the last resort, used only when nothing else resolved. is_trusted is then
                     reported so the screen can warn

is_trusted is true when base_branch is in the trusted list.
alternatives is every other branch matching the trusted list, capped at 5, for the picker.
```

**Resolution order, and why.**

| Priority | Source | Coverage measured on the real workspace |
|---|---|---|
| 1 | A remembered user choice for the repository | n/a |
| 2 | Local `origin/HEAD` | **36 of 36 repositories** |
| 3 | Bitbucket `/default-branch` | Every repository, including the ones never cloned |
| 4 | The trusted list, in order | Only when neither default could be read |
| 5 | The current branch | Last resort, and flagged |

**The checked-out branch must never be the default.** Measured on the real workspace, the checked-out
branch differs from the remote default in most repositories: `atlas` sits on
`feature/CRMMS-77911-Enhancement-Manage-User` while its default is `master`; `mms` sits on
`new_development3` while its default is `main`; `micro-cm-qris` sits on `pre/development` while its
default is `release/production`. Branching a new feature from another feature branch, or from a stale
one, is the defect this ordering exists to prevent.

**The repositories do not agree on a name, which is why the default is read rather than assumed.**
Across the 36 repositories: 23 default to `master`, 10 to `release/production`, 4 to `main`, and 2
report `init`, which is outside the trusted list and is therefore flagged rather than silently used.

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

export interface RepositoryBaseBranch {
  repo: string
  base_branch: string
  resolved_from: "user_choice" | "local_default" | "remote_default" | "convention" | "current_branch"
  is_trusted: boolean
  alternatives: string[]
  is_cloned: boolean
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

The confirmation always states: the repository, the branch name, the base branch, and how many
repositories were searched. The name field is pre-filled and editable.

| Situation | Rendered |
|---|---|
| base resolved from a default | "branching from {base_branch}", with a picker of the alternatives |
| base is not in the trusted list | "branching from {base_branch}, which is not a usual base. Confirm this is right." |
| base came from the current branch | "Lunar could not read this repository's default branch, so it would branch from {base_branch}, which is currently checked out. Confirm this is right." |

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
17. Never default the base branch to whatever is checked out. Read the remote default, and treat the
    current branch as a flagged last resort only.
18. Never create a branch without showing the base branch it will come from.

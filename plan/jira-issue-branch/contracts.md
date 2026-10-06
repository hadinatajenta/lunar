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
| Bitbucket concurrency | 6 in flight | 65 repositories at 6 in flight stays inside the existing client timeout |
| Bitbucket request timeout | 20 seconds | Matches the raised Jira client timeout |
| Git scan timeout | 5 seconds per repository | Matches `GitCommandTimeout` |
| Key pattern | `(?i)\b[A-Z][A-Z0-9]{1,9}-[0-9]{1,6}(?![0-9])` | The negative lookahead is mandatory: `CRMMS-6397` must not match `CRMMS-63970` |

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
  "foreign"         repositories exist, none carry a key from the Jira project in view
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
  "scope": { "local_repository_count": 36, "remote_repository_count": 65, "coverage_percent": 55, "verified_remotely": true }
}

source is exactly one of:
  "local"     found by the local scan only
  "remote"    found on Bitbucket only, so the repository is NOT cloned
  "both"      found by both

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

base_branch is optional. When omitted the repository's current branch is used.
The response is 201, not 200, because a branch was created.

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
| foreign | "The repositories in {root_path} do not contain any branch for the Jira project you are viewing." |
| partial | "Searching {local_repository_count} of {remote_repository_count} repositories. {missing_repository_count} are not cloned locally." |
| ready | nothing is shown |

### Branches for an issue

| Situation | Rendered |
|---|---|
| matches found locally | the branch list, each row naming its repository |
| a match whose source is remote | "in {repo}, which is not in your workspace" plus a clone hint |
| no matches, scope complete | "No branch found for {key} in the {local_repository_count} repositories Lunar can see." |
| no matches, scope partial | "No branch found in the {local_repository_count} of {remote_repository_count} repositories Lunar can see. {missing_repository_count} are not cloned locally." |
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

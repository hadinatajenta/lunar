# Jira issue to branch: matching, repository suggestion, and branch creation

> **`contracts.md` in this folder is frozen and authoritative.** Every endpoint, response field,
> status code, TypeScript type, UI string and constant is defined there. Anything not defined there is
> reported to the parent, not invented. Section 7 lists the decisions that must not be made locally;
> changing one of those is a defect, not a judgement call.

## Objective

Two capabilities, in the order they are used after sprint planning.

1. **Match a Jira key to branches.** Given an issue key, find which repositories and branches
   already reference it, so the state of the work is visible without leaving Lunar.
2. **Create the branch.** For an issue that has no branch, suggest which repository it belongs to,
   propose a branch name, and create it.

The second depends on the first: the suggestion is only meaningful once we can tell whether a branch
already exists.

## Measured starting point

Taken from the real workspace and the real Jira instance.

| Fact | Value |
|---|---|
| Local branches across the 38 repositories | 210 |
| Branches carrying a Jira-style key | 83, or 39% |
| Naming convention in use | `feature/CRMMS-77911-Enhancement-Manage-User`, `feature/CRMMS-72143-QC-QR-Static-API-PTEN` |
| Boards on the instance | 2746, one per squad |
| The user's active sprint | 7 parent issues, sub-tasks excluded |

The convention is consistent enough to match on, and the 61% without a key is why the feature must
also handle "no branch found" as a normal outcome rather than an error.

## Part 1: matching a key to a branch

### The matching rule

A key looks like `CRMMS-77911`: a project prefix of 1 to 10 uppercase alphanumerics, a hyphen, and
1 to 6 digits.

The rule must not match a longer key that starts the same way. `CRMMS-6397` must not match
`CRMMS-63970`, and the real workspace contains branches of both shapes. Matching is therefore
case-insensitive and anchored at the end:

```
(?i)\bCRMMS-6397(?![0-9])
```

The trailing negative lookahead is what prevents the longer key from matching, and it is the part
most likely to be got wrong, so it gets its own test with the real pair of branch names.

### Where to look

| Source | Coverage | Cost |
|---|---|---|
| Local branch names | The 83 measured | One `git for-each-ref` per repository |
| Commit subjects on the branch | Catches work started under an unrelated branch name | One `git log` per candidate branch |
| Remote branch names | Catches a branch someone else pushed | One more ref read, only when asked |

The default scan is local branch names only, because that is fast and answers the question. Commit
scanning is opt-in per issue, since it costs a log read per branch.

### Scan once, not per issue

A sprint has 7 issues. Scanning the workspace per issue would repeat the same 38 repository reads
seven times. The scan therefore runs **once** per request and produces a map from key to branches,
which every issue is then served from. This is the difference between roughly 40 git calls and 280.

### Contract

```
GET /api/jira/issues/{key}/branches
  -> 200 { key, matches: [ { repo, branch, is_current, last_commit, last_commit_relative, ahead, behind } ] }

GET /api/jira/sprint/branches?squad=&board=
  -> 200 { matches: { "CRMMS-77911": [ ... ], "CRMMS-77745": [] } }
```

A key with no match returns an empty array, not a 404. Absence is information.

## Part 2: suggesting the repository

This is the part that needs review. The design goal is that a suggestion is **explainable, never
silent, and improves when corrected**.

### Principle

Never guess silently. Every suggestion carries the evidence that produced it, and the user can
always pick a different repository. A suggestion the user cannot argue with is worse than no
suggestion.

### Signals

Five signals, weighted, combined into one score per repository. Each returns both a score and a
human readable reason.

| # | Signal | Confidence | Reason shown to the user |
|---|---|---|---|
| S1 | **Sibling issues in the same epic** | Highest | "4 other issues in this epic were worked in `aurora`" |
| S2 | **Explicit hint in the issue** | High | "the issue names `mocash_api`", or "component: MOCASH" |
| S3 | **Code index term search** | Medium | "`Penutupan Store Code` appears in 12 files in `mocash_api`" |
| S4 | **Linked Confluence page** | Medium | "the linked wiki page names `aurora`" |
| S5 | **AI ranking** | Fallback | the model's own explanation |

**S1 is the strongest and the cheapest.** Teams work an epic in one or two services, so what the
siblings did predicts what this issue will do. It needs one Jira call for the epic's issues, then
reuses the matcher from Part 1. When there is no epic it falls back to the other issues in the same
sprint, which is weaker but still useful.

**S2 is deterministic and needs no extra calls.** The issue is already loaded. It checks, in order,
the components field, the labels, and the literal presence of a repository name in the summary or
description. Both sides are normalised before comparing, so `mocash_api`, `mocash-api` and
`Mocash API` are the same token.

**S3 uses what we already have.** The code index can tell us which repositories contain the
distinctive terms from the issue. The naive version of this is noisy, so it is bounded three ways:
common terms are dropped, each term is weighted by how rare it is, and a repository must match at
least two distinct terms before it scores at all.

**S5 exists because the other four can all be silent.** It is never the default; it runs when
nothing else fired, or when the user asks for it.

### Scoring

```
score(repo) = sum over signals of  weight(signal) * strength(signal, repo)
```

| Signal | Weight | strength |
|---|---|---|
| S1 epic siblings | 5.0 | share of sibling branches in that repository |
| S2 explicit hint | 4.0 | 1.0 for an exact token, 0.6 for a partial |
| S3 code index | 2.0 | normalised term score, capped at 1.0 |
| S4 Confluence | 1.5 | 1.0 when the page names it |
| S5 AI | 1.0 | the model's own rank, normalised |

Weights are constants in one place so they can be tuned after real use rather than guessed at now.

### Confidence, and what to do at each level

| Level | Condition | Behaviour |
|---|---|---|
| High | S1 or S2 fired, and the leader is at least twice the runner-up | Preselect the leader, show the reasons |
| Medium | A leader exists but the margin is smaller | Show the top three, preselect nothing |
| Low | Nothing fired, or the top two are within 20% | Show the top three, say plainly that Lunar is not sure, offer to ask the model |

**Low confidence must never be presented as a recommendation.** The screen says the evidence is
weak, because a confident wrong answer costs more than an admitted guess.

### Feedback, so it improves

When the user picks a repository, the choice is recorded against the epic and the issue type. A
correction is worth more than a confirmation, because it is the only case where Lunar was wrong.
The next issue in the same epic reads that affinity and gives the corrected repository a boost.

This is a small table, not a model, and it is the difference between a feature that stays mediocre
and one that gets better with use.

```
issue_repo_affinity(epic_key, repo_name, confirmations, corrections, last_used_at)
```

### Where the AI fits, and where it does not

The model is good at reading a business description and recognising which service it implies. It is
bad at being the only evidence, because it cannot show its work in a way the user can check. It is
therefore the tie-breaker and the fallback, and when it runs, its reasoning is shown verbatim
alongside the other signals rather than replacing them.

## Part 3: creating the branch

### Backend operation

`CreateBranch(ctx, rootPath, repoName, branchName, baseBranch)`.

| Rule | Behaviour |
|---|---|
| Name validation | Reuses the existing branch name validator, the same one that already refuses `main; rm -rf /` |
| Write permission | Reuses `resolveWritableRepository`, so it returns 403 unless the workspace root is inside `LUNAR_WORKSPACE_ALLOWED_ROOTS` |
| Already exists | Refused with 409, naming the existing branch |
| Dirty working tree | Refused with the same message the existing checkout guard uses, since creating and switching carries uncommitted changes across |
| Base branch | Defaults to the repository's current branch, overridable |

It is implemented as `git checkout -b <name> <base>` through the existing hardened runner: fixed
binary, argument array, no shell, no credential prompt.

### Choosing the base branch

The repositories do not agree on a name, and the checked-out branch is not a safe default. Both were
measured.

| Remote default | Repositories |
|---|---|
| `master` | 23 |
| `release/production` | 10 |
| `main` | 4 |
| `init` | 2, outside any convention, so flagged rather than used silently |

And the checked-out branch differs from the default in most repositories:

| Repository | Remote default | Actually checked out |
|---|---|---|
| `atlas` | `master` | `feature/CRMMS-77911-Enhancement-Manage-User` |
| `aurora` | `main` | `feature/CRMMS-69759-Maintenance-MDR-Program` |
| `mms` | `main` | `new_development3` |
| `micro-cm-qris` | `release/production` | `pre/development` |
| `merchant-service-laravel` | `release/production` | `feature/CRMMS-77913-Optimasi-Fitur-Risk-Checking` |

Defaulting to the current branch would therefore branch a new feature off another feature branch, or
off a stale one, which is exactly the mistake worth designing out.

The base is resolved in this order: a remembered choice for the repository, then the local
`origin/HEAD` which is present in all 36 repositories, then Bitbucket's `default-branch` endpoint
which answers for the 41 repositories that are not cloned, then the trusted list, and only then the
current branch, flagged.

Bitbucket was verified to answer for repositories that were never cloned: `elbrus` returns `main`,
`administrasi-service` and `gearbox` return `master`.

The base is always shown before the branch is created, with a picker of the alternatives.

### Branch name generation

The convention is `feature/<KEY>-<FeatureName>`. Generating the feature name from the summary is
where this gets ugly: a real summary reads
`Sebagai Maker & Approval di New MMS, I want to mengajukan program perubahan MDR secara bulk dengan pilihan tipe "QRIS Open API"`.

Stripping role filler and Indonesian connectives, taking the meaningful phrase, and title-casing it
gives something usable, but it will not always be right. **The name is therefore always an editable
suggestion, never a final value**, and the generated text is shown in the field rather than applied
silently.

## Negative flows

Both features depend on the workspace, and the workspace can be absent, wrong, or incomplete. What
each case does today was measured rather than assumed.

| Situation | Backend result | What the user sees |
|---|---|---|
| No path configured | Refused, "path is empty" | Folder missing state |
| Path does not exist | Refused, "workspace root does not exist" | Folder missing state |
| Path is a file | Refused, "not a directory" | Folder missing state |
| Path is the home directory | Refused, sensitive root | Folder missing state |
| Relative path | Refused, "must be absolute" | Folder missing state |
| Path exists, holds no repositories | **Accepted, 0 repositories** | "No repositories were found in ..." |
| **Path is itself a single repository** | **Accepted, 0 repositories** | "No repositories were found", which is misleading: the repository is right there, one level up from where we look |
| **A different organisation's workspace** | **Accepted, its repositories** | Nothing fails, and nothing useful happens either |
| Different organisation, allow list points at BRI | Refused, 403 | Correct |
| **Incomplete workspace** | Accepted | **Wrong answers, silently** |

### The one that can do harm

**An incomplete workspace makes "no branch" a lie.** If the branch for an issue lives in a
repository the user has not cloned, Lunar finds nothing and concludes the work has not started. The
suggestion then offers to create a branch that already exists somewhere else. That is worse than no
feature, because it is confidently wrong and the user acts on it.

Three responses, all cheap:

1. **Phrase the scope, never the absolute.** The screen says "not found in the 38 repositories Lunar
   can see", never "no branch exists". This is the whole fix for the common case and it costs
   nothing.
2. **Show the scope where the decision is made.** The create confirmation states which repositories
   were searched, so the user can notice that a repository is missing before agreeing.
3. **Detect the gap against Bitbucket.** Bitbucket Server can list a project's repositories, which
   the current client does not do; it only ever addresses a repository by name. Adding that call
   turns "we searched 38" into "the project has 44 and 6 are not cloned locally". This is the honest
   version and it is a small addition to an existing integration.

### The one that looks fine and is not

**A different organisation's workspace silently produces nothing.** Point the workspace at
`/Users/xxx/BCA` and every path check passes. The repositories are real, the branches are real, and
the key matcher works. But the Jira instance is BRI's, so no branch in that workspace will ever carry
a key from the issues being viewed. Every issue reports no branch, and the user blames the feature.

The fix is a correspondence check, not a path check: after scanning, ask whether any branch in the
workspace carries a key belonging to the Jira project the user is actually looking at. If none do,
say so plainly rather than reporting a clean scan of the wrong thing.

```
workspace readiness
  not-configured   no path set
  invalid          refused by the checks above, with its own reason
  empty            valid path, no repositories inside
  single-repo      the path is itself a repository, so say that instead of "none found"
  foreign          repositories exist but none carry a key from this Jira project
  partial          fewer repositories than the Bitbucket project holds
  ready            repositories present and at least one carries a key from this project
```

### Where this state is shown

The Microservices page already has a workspace state panel with a folder missing state. The Jira page
has nothing, because until now it never needed the workspace. Once it reads branches, it needs the
same treatment, so the readiness state is defined once and used by both pages rather than
reimplemented.

### What is never allowed

- No branch is created without confirmation, and the confirmation always names the repository and
  the search scope.
- A missing workspace never falls back to a guess, an empty suggestion, or a sample repository.
- "Not found" is never rendered as "does not exist".

## Recommending a repository when the workspace is incomplete

Measured against the real environment rather than assumed.

| Fact | Value |
|---|---|
| Repositories in Bitbucket project NME | **65** |
| Repositories cloned into the workspace | **36** |
| Coverage | **55%** |
| Repositories not cloned locally | **41** |
| Local folders that belong to no repository in NME | **12**, so the workspace spans more than one project |
| Jira's dev-status API, which would aggregate the Bitbucket links per issue | **403, not permitted.** The integration is unavailable, so Jira cannot tell us about branches |
| Bitbucket branch search with `filterText` | **Works.** `CRMMS-77911` was found in `atlas` and `mms-api` without either being queried through a clone |

The last two are what decide the design. Jira cannot answer the branch question, and Bitbucket can.

### The principle

**The candidate list must not come from the workspace.** Asking "which repository is this issue in" while
only being able to see 55% of the repositories is asking a question we cannot answer, and answering it
anyway is how the feature becomes confidently wrong.

Bitbucket knows the complete set, so it supplies the universe. The workspace is a fast local cache of
part of that universe, not the source of truth.

### Two tiers

| Tier | Source | Cost | Scope | Used for |
|---|---|---|---|---|
| Fast | Local workspace scan | About 2 seconds | 55% of the project | Every request, and every keystroke-level interaction |
| Authoritative | Bitbucket branch search per repository | One call per repository, cacheable | The **candidate repositories only**, capped at 5 | Verifying that a branch does not already exist in a repository that matters |

**Checking all 65 repositories was rejected.** The cost is 65 calls, and the benefit only exists in
repositories that were plausible to begin with. The authoritative tier is therefore scoped to the
suggestion's top candidates, capped at five.

The authoritative tier is **not** run on every page load. It runs where its answer changes a decision:

1. When the local scan would report **no branch**, against the candidate repositories.
2. **When a branch is about to be created, against the named repository.** This is one call, and it is
   the last line of defence: it catches a branch that exists on the remote while the repository is not
   cloned locally, which is exactly the case that would otherwise produce a duplicate. It closes the
   dangerous path at the single point where it matters, regardless of whether the repository was ever
   a candidate.
3. When the user explicitly refreshes.

The consequence must be stated rather than hidden: the screen says how many repositories were checked
on Bitbucket, and never implies the whole project was.

### What an incomplete workspace does to each signal

| Signal | Needs local code | Behaviour at 55% coverage |
|---|---|---|
| S1 epic siblings | Branches only | **Promoted to Bitbucket.** Sibling branches are searched server side, so this survives the missing repositories |
| S2 explicit hint in the issue | No | **Unaffected.** Full weight |
| S3 code index terms | Yes | **Scaled by coverage.** It can only speak for the repositories that are present |
| S4 linked Confluence page | No | **Unaffected.** Full weight |
| S5 model ranking | A catalogue | **Unaffected in principle**, because the catalogue comes from Bitbucket, not from the local disk. This is the opposite of the naive design, where a partial workspace would have hidden repositories from the model too |

### Confidence is scaled, and the scale is shown

Coverage is not a single number. What matters is whether the repositories **relevant to this issue**
are present:

- If the epic's siblings were all found, coverage is effectively complete for S1 even at 36 of 65.
- If a sibling exists whose repository is not cloned, S1 is blind to it, and that is exactly what
  Bitbucket fixes.

The screen states the basis rather than hiding it:

```
Suggested: mms-api                       confidence: high
  the branch CRMMS-77911 already exists in mms-api on Bitbucket
  2 other issues in this epic were worked in mms-api and atlas
  searched 36 repositories locally and 65 on Bitbucket
```

and when the local tier is all we have:

```
Suggested: mms-api                       confidence: medium
  the issue names mms-api
  3 sibling branches were found locally
  searched 36 of 65 repositories, 29 not cloned
```

### The dangerous case, solved

The failure worth designing against is telling the user a branch does not exist when it does, because
that produces a duplicate branch and a confused squad. With the authoritative tier it becomes a
useful message instead:

```
CRMMS-77911  branch already exists, repository not cloned
  feature/CRMMS-77911-Enhancement-Manage-User
  in mms-api, which is not in your workspace
  [clone it]  [open on Bitbucket]
```

### The workspace spans projects

Twelve local folders belong to no repository in NME, among them `mms`, `mocash_api`,
`service_qris`, `micro-user-mms` and `vue-kanji`. The workspace is therefore a curated set drawn
from more than one Bitbucket project, not a clone of one.

That breaks the naive correspondence check. Comparing the workspace against a single project would
report a correct workspace as foreign. The check must ask whether the workspace contains repositories
from **any** project the user has access to, and the repository catalogue is the union of those
projects rather than one.

### When neither source is available

If Bitbucket cannot be reached and the workspace is partial, the feature does not guess. It reports
that it can only see the local repositories, marks the answer as limited, and never states that a
branch does not exist.

## Out of scope

- Creating the branch on the remote. Local only, consistent with every other git operation here.
- Creating a pull request. That belongs with the Bitbucket work.
- Any automatic action without confirmation.

## Verification

- `CRMMS-6397` matches its branch and does not match a `CRMMS-63970` branch.
- A scan of the real workspace finds the 83 keyed branches and completes in about two seconds.
- A key with no branch returns an empty list and the screen says so.
- The Pangestu sprint's issues resolve against the real workspace, and each of the three accounts is
  checked against its own sprint.
- Suggestion: with a seeded epic, the sibling signal ranks the correct repository first, and the
  reason names the siblings it used.
- Suggestion: low confidence is reported as low, not as a recommendation.
- A correction in one request changes the ranking in the next.
- Branch creation refuses a duplicate, a traversal name, a dirty tree, and a root outside the
  allow list, each with its own status code.
- Existing Jira and workspace tests still pass.

## Risks

- **The 61% of branches with no key** are invisible to matching. The screen must present "no branch
  yet" as the normal starting state, which is exactly what the create action is for.
- **Component names may not map to repositories.** S2 checks the components field, but on this
  instance the squad fields are empty, so components may be too. S1 and S3 carry the feature if so.
- **The epic link may be absent.** Many issues here have no epic. The sprint fallback is weaker.
- **Term search can be noisy.** Bounded by rarity weighting and the two-term minimum, but it may
  still suggest a shared library over the actual service.
- **Branch name generation will sometimes produce something poor.** Accepted, because the name is
  editable and shown before it is applied.

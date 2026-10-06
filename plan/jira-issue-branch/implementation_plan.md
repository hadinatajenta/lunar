# Jira issue to branch: matching, repository suggestion, and branch creation

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

### Branch name generation

The convention is `feature/<KEY>-<FeatureName>`. Generating the feature name from the summary is
where this gets ugly: a real summary reads
`Sebagai Maker & Approval di New MMS, I want to mengajukan program perubahan MDR secara bulk dengan pilihan tipe "QRIS Open API"`.

Stripping role filler and Indonesian connectives, taking the meaningful phrase, and title-casing it
gives something usable, but it will not always be right. **The name is therefore always an editable
suggestion, never a final value**, and the generated text is shown in the field rather than applied
silently.

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

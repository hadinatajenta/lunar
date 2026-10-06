# Subagent workstreams

> **`contracts.md` in this folder is frozen and authoritative.** Every endpoint, response field,
> status code, TypeScript type, UI string and constant is defined there. Anything not defined there is
> reported to the parent, not invented. Section 7 lists the decisions that must not be made locally;
> changing one of those is a defect, not a judgement call.

Eleven agents across three phases, at most three running at once. Every phase ends with its own QA
agent, and the next phase does not start until that QA reports.

The split follows file ownership, not convenience: no two agents in a phase write the same file.

## Phase 1: the matcher and the git operation

| Agent | Owns | Definition of done |
|---|---|---|
| **1A** backend | New `backend/internal/jira/**` and a new workspace scan surface: the key matcher, the single workspace scan producing a key to branches map, and `GET /api/jira/issues/{key}/branches` plus the sprint bulk variant | The boundary test passes for the real `CRMMS-6397` / `CRMMS-63970` pair; a scan of `/Users/erendt/BRI` finds 83 keyed branches and takes about two seconds; a key with no match returns an empty array |
| **1B** backend | `backend/internal/workspace/**` only: `CreateBranch` and its transport routes on both surfaces | Refuses a duplicate with 409, a traversal name with 400, a dirty tree with the existing message, and a root outside the allow list with 403. Unit tested on scratch repositories, never on the real ones |
| **1Q** QA | Tests only | Verifies both, adversarially reviews the matcher for false positives and the git operation for anything that could touch a real repository |

Phase 1 has no frontend work. The matcher and the git operation are useful on their own and are the
foundation both later phases stand on.

## Phase 2: the suggestion engine and the screen

| Agent | Owns | Definition of done |
|---|---|---|
| **2A** backend | The suggestion engine: signals S1 to S4, the scoring function, the confidence tiers, and the affinity table | S1 ranks the correct repository first on a seeded epic and its reason names the siblings used; low confidence is reported as low; a correction changes the next ranking |
| **2B** frontend | The Jira issue surface: matched branches per issue, the suggestion panel with reasons, and the create branch modal | Every suggestion shows at least one reason; the branch name is editable and pre-filled but never applied silently; low confidence is not styled as a recommendation |
| **2C** backend | The Bitbucket authoritative tier: list a project's repositories, search a repository's branches by key, and the two tier lookup that answers "does a branch exist" across all repositories rather than only the cloned ones | `CRMMS-77911` is found in `atlas` and `mms-api` on Bitbucket without either being cloned; the search is cached and concurrent; an unreachable Bitbucket degrades to the local tier with the limit reported, never to a guess |
| **2D** backend and frontend | The workspace readiness states, the cross-project correspondence check, and the shared readiness panel on both pages | A path that is itself a repository reports that, a workspace drawn from several projects is not reported as foreign, and no screen ever renders "not found" as "does not exist" |
| **2Q** QA | Tests only | Checks that the reasons shown match the scores produced, that a low confidence suggestion is never presented as certain, that the modal cannot create a branch without confirmation, and that every readiness state is reachable and reads honestly |

Phase 2 depends on Phase 1 for both the matcher and the git operation.

## Phase 3: the model signal and polish

| Agent | Owns | Definition of done |
|---|---|---|
| **3A** backend | Signal S5 only, plus the repository fingerprint catalogue it needs, reusing the reading pipeline built for the export feature | The model runs only when nothing else fired or when asked, and its reasoning is returned verbatim alongside the other signals rather than replacing them |
| **3B** frontend | Polish: both themes, keyboard access, empty and failure states, and the e2e specs for the whole feature | Every new control has an accessible name and is operable by keyboard; the theme assertions prove the theme applies rather than recording a screenshot |
| **3Q** QA | Tests only | Full regression, plus a live pass against the three real Jira accounts |

## Handoff rules

- One writer per file. An agent that needs a change in a file it does not own reports it to the
  parent instead of editing it.
- Backend agents may not touch `guide.md`; documentation is the parent's, and it is updated once per
  phase so the guide never describes a half finished state.
- Frontend agents may not touch backend files, and the reverse.
- No agent triggers a real AI call in a test. It costs money and takes a minute.
- No agent runs a mutating git operation against `/Users/erendt/BRI`. Those repositories are the
  user's real work; destructive paths are exercised on scratch repositories only.
- Every agent reports the exact command and the raw result, the final line count of each file it
  touched, and anything it could not verify. A claim without a command is not accepted.

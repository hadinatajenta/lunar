# Task list

> **`contracts.md` in this folder is frozen and authoritative.** Every endpoint, response field,
> status code, TypeScript type, UI string and constant is defined there. Anything not defined there is
> reported to the parent, not invented. Section 7 lists the decisions that must not be made locally;
> changing one of those is a defect, not a judgement call.

## Phase 1: matcher and git operation

### 1A matcher
- [ ] Key pattern matches a Jira key and is anchored against a longer key sharing the prefix
- [ ] Case insensitive matching
- [ ] One workspace scan produces a key to branches map, reused for every issue in a request
- [ ] `GET /api/jira/issues/{key}/branches` and the sprint bulk variant
- [ ] A key with no match returns an empty array, not an error
- [ ] Verified against the real workspace: 83 keyed branches found

### 1B create branch
- [ ] `CreateBranch` through the hardened git runner
- [ ] Base branch resolution: remembered choice, then local `origin/HEAD`, then Bitbucket `default-branch`, then the trusted list, then the current branch flagged
- [ ] The current branch is never the silent default
- [ ] A base outside the trusted list is flagged, not silently used
- [ ] The base is shown before creating, with a picker of alternatives
- [ ] Name validation reuses the existing validator
- [ ] Duplicate branch returns 409
- [ ] Dirty tree returns the existing checkout guard message
- [ ] Root outside the allow list returns 403
- [ ] Routes on both the helper and the backend

### 1Q QA
- [ ] Matcher reviewed for false positives against the real branch list
- [ ] Git operation exercised on scratch repositories only
- [ ] Real repositories confirmed untouched

## Phase 2: suggestion engine and screen

### 2A suggestion engine
- [ ] S1 epic siblings, with the sprint fallback when there is no epic
- [ ] S2 components, labels, and literal repository names
- [ ] S3 code index term search, bounded by rarity and a two term minimum
- [ ] S4 linked Confluence pages
- [ ] Scoring with weights in one place
- [ ] Confidence tiers, and low confidence never presented as a recommendation
- [ ] Affinity table, and a correction changing the next ranking

### 2B frontend
- [ ] Matched branches shown per issue
- [ ] Suggestion panel showing every reason
- [ ] Editable branch name, pre-filled but never applied silently
- [ ] Create branch modal with confirmation

### 2Q QA
- [ ] Reasons shown match the scores produced
- [ ] Low confidence is not presented as certain
- [ ] No branch can be created without confirmation

## Phase 2b: the authoritative tier and workspace readiness

### 2C Bitbucket tier
- [ ] List a Bitbucket project's repositories
- [ ] Search a repository's branches by key with `filterText`
- [ ] Two tier lookup: local scan first, Bitbucket where the local answer would be "not found"
- [ ] Concurrent and cached, since the project holds 65 repositories
- [ ] An unreachable Bitbucket degrades to the local tier and reports the limit

### 2D workspace readiness
- [ ] All seven readiness states implemented and reachable
- [ ] The readiness panel is shared by both pages, one implementation, not two
- [ ] Readiness states defined once: not-configured, invalid, empty, single-repo, foreign, partial, ready
- [ ] A path that is itself a repository says so instead of "none found"
- [ ] The correspondence check handles a workspace drawn from several projects, since 12 local folders belong to no repository in NME
- [ ] The correspondence check reports when no branch carries a key from the current Jira project
- [ ] "Not found in the N repositories Lunar can see", never "does not exist"
- [ ] The Jira page shows the same readiness panel the Microservices page already has
- [ ] The create confirmation names the repository and the number of repositories searched

## Phase 3: model signal and polish

### 3A model signal
- [ ] Repository fingerprint catalogue reusing the export reading pipeline
- [ ] S5 runs only as a fallback or on request
- [ ] Reasoning returned verbatim alongside the other signals

### 3B frontend polish
- [ ] Both themes
- [ ] Keyboard access and accessible names
- [ ] Empty and failure states
- [ ] e2e specs for the whole feature

### 3Q QA
- [ ] Full regression
- [ ] Live pass against the three real Jira accounts

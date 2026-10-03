# User Guide — Implementation Plan

Status: proposed, awaiting approval
Owner: supervisor agent
Feature folder: `plan/user-guide/`

---

## 1. Objective

Guide an engineer through the three Atlassian pages the first time they open each one. The tour
highlights the real controls on the page, explains what each one does, advances step by step, and can
be skipped at any point. It can be replayed later from a help button in the header.

In scope: Jira BRI, Bitbucket BRI, Confluence BRI.

Out of scope: Overview, Microservices, Copilot, Settings. Building the tour engine so that adding
those pages later is a data change only.

---

## 2. Decisions Taken

| Decision | Choice |
|---|---|
| Pages covered | The three Atlassian pages only |
| When it appears | Automatically on the first visit to each page, and replayable on demand |
| Engine | `driver.js` 1.9.0, MIT, zero dependencies |

---

## 3. Findings From Inspection

The anchors below were read from the running application, not guessed. Every selector in section 6
was verified to exist in the source.

### 3.1 Stable anchors already exist

| Page | Anchor | Source component |
|---|---|---|
| Jira | `tab-assigned`, `tab-backlog` | `JiraPage.vue` |
| Jira | `filter-all`, `filter-docs`, `filter-bugs`, `filter-subtasks` | `JiraPage.vue` |
| Jira | `doc-filter-all` | `JiraPage.vue` |
| Bitbucket | `push-filter-all`, `push-filter-ready`, `push-filter-stale` | `YourWorkList.vue` |
| Bitbucket | `pr-filter-all`, `pr-filter-mine`, `pr-filter-ai` | `PullRequestTable.vue` |
| Confluence | `doc-search` | `DocumentToolbar.vue` |
| Confluence | `doc-filter` combined with `data-filter` | `DocumentToolbar.vue` |

The Confluence filters share one `data-testid` but each carries a distinct `data-filter` value
(`all`, `ut`, `query`, `sop`), so `[data-testid="doc-filter"][data-filter="ut"]` targets one filter
uniquely. The existing end-to-end suite already relies on this, so no markup change is needed.

### 3.2 One anchor is conditional and must not be a tour step

`btn-retry` exists on the Atlassian pages only when an error banner is showing. It is not always
present, so a tour step must never depend on it. Any step whose element is missing is skipped at
runtime rather than shown pointing at nothing.

### 3.3 No tour code exists today

There is no tour, onboarding, or highlight code anywhere in the frontend. The only guide-like
component is `HelperSetupGuide.vue`, which is static instructions for the local helper and is
unrelated.

Available primitives are `Button`, `Checkbox`, `Input`, and `Toast`. None is needed for the tour
itself, but the help button should match `Button` conventions.

---

## 4. Architecture

```text
frontend/src/features/user-guide/
  tours/jira.ts            step definitions for /jira
  tours/bitbucket.ts       step definitions for /bitbucket
  tours/confluence.ts      step definitions for /confluence
  tours/registry.ts        route to tour definition lookup
  composables/useTour.ts   driver.js wrapper, completion storage, start and skip
  composables/useTourAutoStart.ts  watches the route and starts an unseen tour
  components/GuideButton.vue       header button, visible only when a tour exists
  user-guide.css           driver.js theming mapped to the application tokens
```

Flow:

1. The authenticated layout mounts `useTourAutoStart`.
2. A route change looks up a tour for the new route.
3. If a tour exists, has not been completed, and its first anchor is present, it starts.
4. On finish or skip, the tour is marked completed in local storage.
5. `GuideButton` in the header starts the same tour on demand, ignoring the completed flag.

### 4.1 Waiting for anchors

Pages fetch their data asynchronously, so the anchors may not exist when the route changes. The auto
start waits for the first anchor with a bounded poll of roughly three seconds before giving up
silently. A tour that cannot find its first anchor is not started and not marked completed, so it is
offered again on the next visit.

---

## 5. Frozen Contracts

### 5.1 Step and tour shapes

```ts
export type TourSide = "top" | "right" | "bottom" | "left"

export interface TourStep {
  element: string
  title: string
  description: string
  side?: TourSide
  align?: "start" | "center" | "end"
  optional?: boolean
}

export interface TourDefinition {
  id: string
  route: string
  pageName: string
  steps: TourStep[]
}
```

`optional: true` marks a step that is skipped when its element is absent. Steps without it are
treated as optional too, because a missing element must never produce an empty highlight; the flag
exists only to document intent.

### 5.2 Registry

```ts
export const findTourByRoute = (route: string): TourDefinition | null
export const listTours = (): TourDefinition[]
```

Route matching ignores query strings and trailing slashes. A detail route such as
`/confluence/:id` resolves to the `/confluence` tour so the tour still works on a document page.

### 5.3 Completion storage

Single key `lunar_tours_completed`, holding a JSON object of tour id to `true`. This matches the
existing local-storage conventions in the application, which use `lunar_theme` and
`copilot_enabled_domains`.

Reading tolerates missing, malformed, or partially written values and falls back to an empty object.
Writing merges rather than replaces, so completing one tour never clears another.

---

## 6. Step Definitions

Deliberately short. A tour that runs longer than about five steps gets dismissed rather than read.

### 6.1 Jira BRI (`/jira`)

| # | Element | Purpose |
|---|---|---|
| 1 | `[data-testid="tab-assigned"]` | Issues assigned to you |
| 2 | `[data-testid="tab-backlog"]` | The active sprint backlog |
| 3 | `[data-testid="filter-all"]` | Narrow the board by issue type |
| 4 | `[data-testid="doc-filter-all"]` | Technical documents linked to issues |

### 6.2 Bitbucket BRI (`/bitbucket`)

| # | Element | Purpose |
|---|---|---|
| 1 | `[data-testid="push-filter-all"]` | Push activity, and the stale filter that finds forgotten branches |
| 2 | `[data-testid="pr-filter-mine"]` | Pull requests waiting on you |
| 3 | `[data-testid="pr-filter-ai"]` | Pull requests the AI review flagged |

### 6.3 Confluence BRI (`/confluence`)

| # | Element | Purpose |
|---|---|---|
| 1 | `[data-testid="doc-search"]` | Search across every document |
| 2 | `[data-testid="doc-filter"][data-filter="ut"]` | Filter by document type |
| 3 | `[data-testid="doc-filter"][data-filter="query"]` | Query review documents |

---

## 7. Theming

`driver.js` ships a light default that would look wrong beside the application. Its colours are
driven by CSS custom properties, so `user-guide.css` maps them onto the existing tokens rather than
hardcoding values:

```text
--driver-popover-bg        -> var(--surface-raised)
--driver-popover-text      -> var(--text)
--driver-popover-title-color -> var(--text)
--driver-overlay-bg        -> a translucent var(--bg)
```

Both themes are covered because the tokens themselves are theme-aware. The popover must be legible
in light mode, which is the failure the provider logos had earlier, so both themes are checked in a
browser before the work is called done.

---

## 8. Accessibility and Behaviour

- Close is always available. The dismiss control is labelled **Skip** so the intent is explicit
  rather than an unlabelled cross.
- Escape closes the tour and marks it completed, matching skip.
- Left and right arrow keys move between steps, which `driver.js` provides.
- Progress is shown as "Step N of M" so the user knows how much is left.
- Focus returns to the help button after the tour closes.
- The tour never blocks the page: the overlay is dismissible at every step.

---

## 9. Failure Handling

| Situation | Behaviour |
|---|---|
| First anchor never appears | Tour does not start, nothing is marked completed |
| A later anchor is missing | That step is skipped, the tour continues |
| All anchors are missing | Tour ends immediately without marking completed |
| Stored completion is malformed | Treated as empty, tours are offered again |
| User skips | Marked completed, never auto-offered again, still replayable |
| User is on a detail route | Resolves to the parent tour so it still works |

---

## 10. Verification Strategy

| Layer | Method |
|---|---|
| Step definitions | A test asserting every selector in every tour resolves to exactly one element on its page, against the real application |
| Registry | Route lookup tests including a detail route and an unknown route |
| Completion storage | Malformed, missing, and partial JSON cases; merge must not clear other tours |
| Auto start | Browser check that the tour appears on the first visit, not on the second, and again after clearing storage |
| Replay | Browser check that the header button starts the tour after it was completed |
| Skip | Browser check that Skip closes the tour and suppresses the next automatic start |
| Theming | Screenshots in both dark and light mode |

The selector verification is the important one. A tour that points at a missing element is worse than
no tour, and that is exactly the failure a type checker cannot catch.

---

## 11. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Anchors change and a step silently disappears | The tour describes a control that moved | The selector test fails when a selector stops matching |
| Pages are slow, so the tour starts before content exists | Empty highlight | Bounded wait for the first anchor, and skip missing steps |
| The tour annoys returning users | Dismissed without reading | Per-page completion flag, never auto-starts twice |
| Popover unreadable in light mode | Same class of bug as the provider logos | Verify both themes in a browser |
| driver.js styling leaks into the application | Visual regression elsewhere | Scope the overrides to the driver.js root class |

---

## 12. Workstream Decomposition

| Workstream | Scope | Depends on |
|---|---|---|
| WS-A | `TourStep` and registry contracts, completion storage, `useTour` wrapper, theming | none, supervisor |
| WS-B | The three tour definitions and their selector test | WS-A |
| WS-C | `GuideButton`, `useTourAutoStart`, header wiring | WS-A |

WS-B and WS-C may run concurrently once WS-A is frozen. Detail, file ownership, and handoff rules are
in `plan/user-guide/subagent_workstreams.md`.

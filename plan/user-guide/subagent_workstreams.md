# User Guide — Subagent Workstreams

Supervisor owns WS-A and final integration. Three workstreams, strict file ownership, no two agents
editing one file.

---

## Dependency Graph

```text
WS-A  contracts, storage, useTour wrapper, theming        (supervisor, serialized first)
  |
  +--> WS-B  the three tour definitions and their selector test
  |
  +--> WS-C  header button, auto start, layout wiring
```

WS-B and WS-C may run concurrently once WS-A is frozen.

---

## WS-A — Contracts and Engine (supervisor)

Files owned:

```text
frontend/src/features/user-guide/tours/registry.ts        new
frontend/src/features/user-guide/tours/types.ts           new
frontend/src/features/user-guide/composables/useTour.ts   new
frontend/src/features/user-guide/user-guide.css           new
frontend/package.json                                     edit, driver.js only
```

Deliverables:

1. `TourStep` and `TourDefinition` exactly as frozen in the plan.
2. `findTourByRoute` with detail-route fallback, and `listTours`.
3. Completion storage under `lunar_tours_completed`, merging on write, tolerant on read.
4. `useTour` wrapping `driver.js`: start, skip, mark completed, return focus.
5. A bounded wait for the first anchor, and skipping of steps with absent elements.
6. Theme mapping through the driver.js custom properties, scoped to its root class.

Definition of done: the engine starts and stops a tour given a definition, malformed storage does not
throw, and nothing outside the tour is restyled.

---

## WS-B — Tour Content

Files owned:

```text
frontend/src/features/user-guide/tours/jira.ts             new
frontend/src/features/user-guide/tours/bitbucket.ts        new
frontend/src/features/user-guide/tours/confluence.ts       new
frontend/src/features/user-guide/tours/tours.test.ts       new
```

Deliverables:

1. One tour definition per Atlassian page, using the anchors listed in the plan.
2. Short descriptions that state what each control does.
3. A test asserting every selector in every tour resolves to exactly one element on its page,
   against the running application.
4. A test asserting registry lookup for a known route, a detail route, and an unknown route.

Constraints: at most five steps per page. Do not add tour steps for `btn-retry`, which appears only
during an error state. Do not modify the page components; every anchor already exists.

Definition of done: the selector test passes against the real application, and it fails when a
selector is changed to something that does not exist. Prove the second half by temporarily breaking
one selector and showing the failure.

---

## WS-C — Presentation and Triggering

Files owned:

```text
frontend/src/features/user-guide/components/GuideButton.vue       new
frontend/src/features/user-guide/composables/useTourAutoStart.ts  new
frontend/src/components/layout/AppHeader.vue                      edit
frontend/src/app/App.vue                                          edit, mounting the auto start
```

Deliverables:

1. A help button in the header, rendered only when the current route has a tour.
2. `useTourAutoStart`, watching the route and starting a tour that has not been completed.
3. Replay support: the header button starts the tour regardless of the completed flag.
4. An accessible name on the button, and focus returned to it when the tour closes.
5. No layout shift when the button appears.

Constraints: `AppHeader.vue` is small; keep the change to mounting the component. Do not put tour
logic in the header itself.

Definition of done: a first visit starts the tour, a second visit does not, and the header button
replays it.

---

## Handoff Rules

1. No workstream edits a file owned by another. Request changes through the supervisor.
2. WS-A completes and is verified before WS-B or WS-C start.
3. Contract changes after WS-A are supervisor-approved and broadcast.
4. Every workstream reports files changed, line counts, the exact commands run, and raw output. A
   claim without command output is not accepted.
5. No commits, no pushes, no dependencies beyond `driver.js`, no dev servers started by subagents.

## Supervisor Integration Checklist

1. `vue-tsc -b` exits zero.
2. `vite build` succeeds.
3. Playwright suite stays green.
4. Every step selector verified against the running application, not the source alone.
5. No selector points at `btn-retry` or any other conditional anchor.
6. Popover verified in dark and light mode.
7. Zero code comments, every new file under 300 lines.
8. Prune this plan once verified.

# User Guide — Task Checklist

Legend: `[x]` done, `[ ]` pending. Every completed item names the command or artifact that proves it.

---

## Stage F — Inspection (complete)

- [x] F.1 Confirmed no tour, onboarding, or highlight code exists in the frontend
- [x] F.2 Enumerated the interactive elements of the three Atlassian pages from the running application
- [x] F.3 Confirmed every planned anchor already carries a stable `data-testid`
- [x] F.4 Confirmed the Confluence filters are uniquely addressable through `data-filter`, so no markup change is needed
- [x] F.5 Found that `btn-retry` exists only when an error banner is shown, so it must never be a tour step
- [x] F.6 Confirmed `driver.js` 1.9.0 is MIT licensed with zero dependencies
- [x] F.7 Confirmed the existing UI primitives are `Button`, `Checkbox`, `Input`, and `Toast`

---

## Stage 1 — Contracts and Engine

Owner: WS-A.

- [ ] 1.1 Add `driver.js` 1.9.0 as a dependency
- [ ] 1.2 `tours/registry.ts` with `TourDefinition`, `TourStep`, `findTourByRoute`, and `listTours`
- [ ] 1.3 Detail routes such as `/confluence/:id` resolve to the parent tour
- [ ] 1.4 Completion storage under `lunar_tours_completed`, merging rather than replacing
- [ ] 1.5 Storage reading tolerates missing, malformed, and partial JSON
- [ ] 1.6 `useTour` wrapping driver.js: start, skip, completion marking, and focus return
- [ ] 1.7 Bounded wait for the first anchor before starting, silently giving up when absent
- [ ] 1.8 Steps whose element is missing are skipped rather than highlighted empty
- [ ] 1.9 Close control labelled Skip, with Escape mapped to the same outcome
- [ ] 1.10 Progress shown as "Step N of M"
- [ ] 1.11 `user-guide.css` mapping the driver.js custom properties onto the application tokens
- [ ] 1.12 Overrides scoped to the driver.js root class so nothing else is affected

---

## Stage 2 — Tour Content

Owner: WS-B. Depends on Stage 1.

- [ ] 2.1 Jira tour: assigned tab, backlog tab, issue-type filter, document filter
- [ ] 2.2 Bitbucket tour: push filter and stale, pull requests assigned to me, AI-flagged pull requests
- [ ] 2.3 Confluence tour: search box, UT filter, query review filter
- [ ] 2.4 Every description explains what the control does, in English, without filler
- [ ] 2.5 Selector test asserting every step selector resolves to exactly one element on its page
- [ ] 2.6 Registry test covering a known route, a detail route, and an unknown route

---

## Stage 3 — Presentation and Triggering

Owner: WS-C. Depends on Stage 1.

- [ ] 3.1 `GuideButton` in the header, visible only when the current route has a tour
- [ ] 3.2 The button follows the existing `Button` conventions and the header layout
- [ ] 3.3 `useTourAutoStart` watching the route and starting an unseen tour
- [ ] 3.4 The tour is offered once per page and never auto-starts a second time
- [ ] 3.5 The header button replays a completed tour
- [ ] 3.6 Accessible name on the button, and focus returns to it after the tour closes
- [ ] 3.7 No layout shift in the header when the button appears or disappears

---

## Stage 4 — Verification

- [ ] 4.1 Browser check: the tour appears on the first visit to each Atlassian page
- [ ] 4.2 Browser check: it does not appear on the second visit
- [ ] 4.3 Browser check: it appears again after clearing local storage
- [ ] 4.4 Browser check: Skip closes the tour and suppresses the next automatic start
- [ ] 4.5 Browser check: the header button replays a completed tour
- [ ] 4.6 Browser check: a step whose element is absent is skipped, and the tour still finishes
- [ ] 4.7 Screenshots of the popover in dark and light mode, confirming legibility in both
- [ ] 4.8 Confirm the tour does not break when a page is showing an error state
- [ ] 4.9 Confirm the existing end-to-end suite still passes

---

## Supervisor Integration

- [ ] S.1 `vue-tsc -b` exits zero
- [ ] S.2 `vite build` succeeds
- [ ] S.3 Playwright suite stays green
- [ ] S.4 Zero code comments in every new file
- [ ] S.5 Every new file under 300 lines
- [ ] S.6 Only the single approved dependency was added
- [ ] S.7 Prune this plan once verified

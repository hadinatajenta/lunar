# Jira backlog: squad scoping, active sprint only, and UI repair

## Reported problems

1. A user outside the Fortune squad still sees the Fortune squad backlog.
2. Only the user's ACTIVE sprint should be shown.
3. Remove the squad filter: the `All squads` pill and the squad pills.
4. Remove `Other Data - Fortune Squad` from the list.
5. The sprint group has a down arrow that cannot be opened, and it does not show the user's active
   sprint backlog.
6. Opening a backlog item shows a modal whose button layout is untidy.
7. `Read more` should take the user to jira.bri.co.id.

## Findings

### Squad scoping, measured against the real instance

The stored PATs were decrypted with a throwaway probe, which was deleted afterwards.

| Lunar account | Jira account | Assigned issues | Sprint values on those issues |
|---|---|---|---|
| hadinata.jenta@sharingvision.co.id | Hadinata Jenta | 56 | Sprint 44 - Fortune Squad, Sprint 43 - Fortune Squad |
| 90175548@bri.co.id | Ahmad Dani Alfaza | 74 | Sprint 44 - Fortune Squad, Sprint 43 - Fortune Squad |
| gilar.pangestu@indocyber.id | Gilar Aji Pangestu | 26 | **Sprint #36 - Pangestu Squad, Sprint #35 - Pangestu Squad** |

The third account belongs to Pangestu Squad and Jira holds that fact. Three defects hide it.

**a. The sprint field is never requested.** `SearchMyIssues` asks for
`customfield_15310`, which on this instance is **Acceptance Criteria**, null on all 156 issues
checked. The sprint field is `customfield_10104`. The two identifiers are adjacent, which reads as
an off-by-one. Because the field is absent, `JiraIssue.SprintName` is always empty and
`detectSquadFromIssues` returns nothing for every user.

**b. Empty detection widens to everything.** `buildBacklogResponse` falls back to
`targetSquad = "all"`, and `MatchSquad` returns true for `"all"`, so the page shows every sprint
that survived the board filter with no sign that detection failed.

**c. The board is hardcoded.** `JIRA_BACKLOG_BOARD_ID = 2646` appears in the sprint request URL and
in `filterBacklogSprints`, which drops any sprint whose `originBoardId` differs. Jira reports board
2646 as `03 - Fortune`, and the instance exposes **2746 boards**. Only one squad can ever appear.
The filter also discards sprints genuinely present on the board: board 2646 returns
`Sprint Azzuri #5` with `originBoardId 3946`, which the filter removes.

Jira returns the sprint as a Java toString that carries the owning board:

```
...Sprint@1f2748[id=10983,rapidViewId=2646,state=ACTIVE,name=Sprint 44 - Fortune Squad,...]
```

`rapidViewId` is the board that owns the sprint, so the user's own issue tells us which board to
read. `ExtractSquadName` already parses both observed title forms correctly:
`Sprint 44 - Fortune Squad` and `Sprint #36 - Pangestu Squad`.

### Sprint states on board 2646

```
id=10103  board=3946  active  Sprint Azzuri #5
id=10983  board=2646  active  Sprint 44 - Fortune Squad
id=8827   board=2646  future  Other Data - Fortune Squad
```

`Other Data - Fortune Squad` is a **future** sprint, so restricting the request to active sprints
removes it, and it never matched a squad name anyway: `ExtractSquadName` returns the whole string
because neither side of the dash starts with `Sprint`.

### Why the arrow does nothing

`BacklogList.vue` renders a chevron inside `.backlog-sprint-title` with no click handler, no
`aria-expanded` and no collapse state. The issue list below it is always rendered, so the arrow
promises a control that does not exist.

### Why the modal footer is untidy

The template uses `<div class="modal-footer">` containing a ghost button, a `flex: 1` spacer and a
primary button, but **`.modal-footer` has no rule in the scoped stylesheet**. The buttons are
therefore block-level siblings stacked vertically, with an empty div between them, instead of one
footer row. Two hardcoded colours in the same component also ignore the theme:
`#bfc6cf` on `.description-text` and `rgba(255,255,255,0.08)` on the scrollbar thumb.

## Scope

### Backend

1. Request `customfield_10104` in `SearchMyIssues`; drop `customfield_15310`.
2. Parse `name` and `rapidViewId` out of the sprint value, tolerating a missing or malformed value.
3. Ask for **active sprints only**, dropping `future` from the request.
4. Read the sprint list for the detected board, keeping 2646 only as a last resort.
5. Delete the `originBoardId` filter.
6. Keep the squad filter parameters working so the endpoint contract does not break, but stop
   widening to `"all"` when detection fails: report that no squad was detected instead.

### Frontend

1. Remove the squad filter bar and its pills, along with the now-unused props and event.
2. Give the sprint header a real expand and collapse: a button with `aria-expanded`, a rotating
   chevron, and a collapsed default for anything that is not the active sprint.
3. Add the missing `.modal-footer` layout so the actions sit in one row, and replace the hardcoded
   colours with theme tokens.
4. Point `Read more` at the issue in Jira and open it in a new tab.
5. Show the user's own squad in the backlog header so it is clear whose sprint is on screen.

### Out of scope

`JiraPage.vue` at 690 lines and `IssueDetailModal.vue` at 681 lines both exceed the 300 line rule.
Splitting them is worthwhile but is not part of this repair and would bury the actual fixes.

## Verification

- Each of the three real accounts returns the squad its own issues belong to.
- Only active sprints are listed, so `Other Data - Fortune Squad` is gone.
- The squad filter bar is absent and no `All squads` control remains.
- The sprint header toggles, reports `aria-expanded`, and is operable by keyboard.
- The modal footer renders its actions on one row in both themes.
- `Read more` opens the issue on jira.bri.co.id in a new tab.
- Existing Jira backend and e2e tests still pass.

## Risks

- The sprint value is a Java toString whose shape can vary by Jira version. The parser must fall back
  rather than fail the request.
- `rapidViewId` is the board a sprint belongs to, which matches the squad board on this instance but
  is not guaranteed by Jira's model.
- Restricting to active sprints means a user with no active sprint sees an empty state. That is the
  requested behaviour and must read as deliberate, not broken.

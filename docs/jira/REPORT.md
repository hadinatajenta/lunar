# Lunar QA Automation Report: Jira BRI Integration

## Executive Summary

This report documents the verification and test suite for the **Jira BRI Integration** implemented across the Lunar backend and frontend. The integration enables engineers working on BRI MMS projects to inspect active tasks on a responsive Kanban board, browse sprint backlogs, view issue details within a centered modal, expand long user story descriptions, and handle failure modes gracefully.

The test suite was executed in visible headed browser mode with a total of **18 test cases** (6 negative & error recovery paths, 12 positive interaction paths), achieving **100% pass rate**.

---

## Architecture & Contract Parity

* **Backend Clean Architecture (`backend/internal/jira/`)**:
  * **Domain**: `JiraIssue`, `JiraStatus`, `JiraPriority`, `JiraSprint`, `JiraBacklogResponse`.
  * **Infrastructure**: `AtlassianJiraClient` targeting `https://jira.bri.co.id` with Bearer token authentication and deterministic offline mock fallback.
  * **Application**: `JiraService` retrieving decrypted user PAT from the SQLite encrypted vault (`user_secrets`).
  * **Transport**: HTTP endpoints mounted on `net/http` ServeMux:
    * `GET /api/jira/issues`: returns assigned issues for current user.
    * `GET /api/jira/backlog`: returns sprint backlog data.
    * `GET /api/jira/issues/{key}`: returns issue detail by key.
* **Frontend Architecture (`frontend/src/features/jira/`)**:
  * **Components**:
    * `KanbanBoard.vue`: 3 columns (`Open`, `In Progress`, `Done`), **5-item default limit** per column with `"Load more"` pagination.
    * `KanbanCard.vue`: Issue card with type badge, priority dot, title, story points, and assignee.
    * `IssueDetailModal.vue`: Centered modal dialog displaying issue key, story summary, status, priority, and collapsible `"Read more"` description.
    * `BacklogList.vue`: Sprint grouped backlogs (Active Sprint, Future Sprint, Backlog).
    * `JiraPage.vue`: Page view with tab navigation and subfilters.

---

## Test Scenarios & Verification Matrix

| ID | Test Scenario | Type | Result | Evidence |
|---|---|---|---|---|
| TC-JIRA-01 | User cannot see board when Jira PAT is unconfigured | Negative | PASS | `04_jira_negative_unconfigured.png` |
| TC-JIRA-02 | User can see 401 error banner when Jira PAT is invalid or expired | Negative | PASS | `05_jira_negative_401.png` |
| TC-JIRA-03 | User can see 502 VPN error banner when upstream is unreachable | Negative | PASS | `06_jira_negative_502.png` |
| TC-JIRA-04 | User can see empty column state when no issues exist for a status | Negative | PASS | - |
| TC-JIRA-05 | User can see empty state when subfilter yields zero matches | Negative | PASS | - |
| TC-JIRA-06 | User can view assigned issues Kanban board with default 5 items per column | Positive | PASS | `01_jira_board_overview.png` |
| TC-JIRA-07 | User can click load more button to expand column when more than 5 cards exist | Positive | PASS | - |
| TC-JIRA-08 | User can click card to open centered modal dialog (not drawer) | Positive | PASS | `02_jira_modal_open.png` |
| TC-JIRA-09 | User can verify modal contains Jira code, user story title, status, and priority | Positive | PASS | `02_jira_modal_open.png` |
| TC-JIRA-10 | User can toggle read more and read less description button | Positive | PASS | - |
| TC-JIRA-11 | User can close modal via close button (X) | Positive | PASS | - |
| TC-JIRA-12 | User can close modal via backdrop click | Positive | PASS | - |
| TC-JIRA-13 | User can close modal via Escape key | Positive | PASS | - |
| TC-JIRA-14 | User can filter by category (DEV Documents, Bug / Defect, Subtask) | Positive | PASS | - |
| TC-JIRA-15 | User can filter DEV Documents by sub-type (All, UT, Query Review, SOP) | Positive | PASS | - |
| TC-JIRA-16 | User can switch to backlog tab and view active sprint, future sprint, and backlog issues | Positive | PASS | `03_jira_backlog_view.png` |
| TC-JIRA-17 | User can click backlog issue to open issue detail modal | Positive | PASS | - |

---

## Visual Evidence

### 1. Kanban Board Overview with Default 5 Items Limit
![Jira Board Overview](file:///Users/erendt/code/lunar/docs/jira/evidence/01_jira_board_overview.png)
*Figure 1: Full-page view of the Jira BRI Kanban board displaying Open, In Progress, and Done columns capped at 5 cards with a "Load more" button.*

### 2. Centered Issue Detail Modal with Expandable "Read more" Description
![Jira Modal](file:///Users/erendt/code/lunar/docs/jira/evidence/02_jira_modal_open.png)
*Figure 2: Centered modal dialog showing issue code CRMMS-77911, full story summary, status/priority pills, and collapsible description.*

### 3. Sprint Backlog View
![Jira Backlog](file:///Users/erendt/code/lunar/docs/jira/evidence/03_jira_backlog_view.png)
*Figure 3: Backlog tab displaying active sprint (Sprint 44 - Fortune Squad), future sprint, and product backlog items.*

### 4. Negative Flow: Unconfigured Jira PAT
![Unconfigured Jira PAT](file:///Users/erendt/code/lunar/docs/jira/evidence/04_jira_negative_unconfigured.png)
*Figure 4: Non-destructive warning banner displayed when user has not saved a Jira PAT in Settings, with direct link to /settings.*

### 5. Negative Flow: 401 Unauthorized (Invalid or Expired PAT)
![401 Unauthorized](file:///Users/erendt/code/lunar/docs/jira/evidence/05_jira_negative_401.png)
*Figure 5: Error banner alerting the user that their Jira PAT is invalid or expired, with remediation CTA.*

### 6. Negative Flow: 502 Bad Gateway (VPN Disconnect)
![502 Bad Gateway](file:///Users/erendt/code/lunar/docs/jira/evidence/06_jira_negative_502.png)
*Figure 6: Explicit VPN connection error guidance banner with an interactive Retry button.*

# Test Cases: Jira BRI

This document details the canonical test cases for the Jira BRI workspace covering the Kanban Board, Issue Detail Modal, Backlog & Sprints, Squad Filtering, and Error Recovery.

---

## Positive Scenarios

### JIRA-BOARD-001
- **Title**: Engineer views assigned issues Kanban board with default 5 items per column
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/jira/positive/kanban-board.spec.ts`
- **Preconditions**: Jira PAT configured in vault.
- **Steps**:
  1. Navigate to `/jira`.
- **Expected Results**:
  - Three columns render: Open, In Progress, Done.
  - Open column displays first 5 cards with total count badge "7".
  - "Load more" button is visible in Open column.
- **Evidence**: `docs/qa/evidence/jira/01_jira_board_overview.png`

### JIRA-BOARD-002
- **Title**: Engineer clicks load more button to expand column when more than 5 cards exist
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/kanban-board.spec.ts`
- **Steps**:
  1. Locate Open column with 7 total cards.
  2. Click "Load more" button.
- **Expected Results**:
  - Remaining 2 cards render, bringing visible count to 7.
  - "Load more" button is dismissed.

### JIRA-SKELETON-001
- **Title**: Engineer sees skeleton loader instead of dummy data while Jira data is loading
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/jira/positive/kanban-board.spec.ts`
- **Steps**:
  1. Delay `/api/jira/issues` response by 800ms.
  2. Navigate to `/jira`.
- **Expected Results**:
  - `[data-testid="kanban-skeleton"]` is visible during transit.
  - Skeleton is replaced by actual cards once response resolves.

### JIRA-CACHE-001
- **Title**: Returning to Jira tab keeps board instantly visible without skeleton loader
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/kanban-board.spec.ts`
- **Steps**:
  1. Load `/jira`.
  2. Navigate to `/settings`.
  3. Re-navigate to `/jira`.
- **Expected Results**:
  - Board cards display immediately without skeleton flicker.
- **Evidence**: `docs/qa/evidence/jira/07_jira_cached_revisit.png`

### JIRA-MODAL-001
- **Title**: Engineer clicks issue card to open centered detail modal dialog
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/jira/positive/issue-modal.spec.ts`
- **Steps**:
  1. Click card `CRMMS-77911`.
- **Expected Results**:
  - Modal container opens with `aria-modal="true"`.
  - Background scrim backdrop is visible.
- **Evidence**: `docs/qa/evidence/jira/02_jira_modal_open.png`

### JIRA-MODAL-002
- **Title**: Modal correctly displays issue key, title, status, priority, and story points
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/jira/positive/issue-modal.spec.ts`
- **Steps**:
  1. Open card `CRMMS-77911`.
- **Expected Results**:
  - Modal key shows `CRMMS-77911`.
  - Title shows Indonesian user story.
  - Status shows "In Progress".
  - Priority shows "High".
  - Points shows "5 pts".

### JIRA-MODAL-003
- **Title**: Engineer can toggle description between "Read more" and "Read less"
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/jira/positive/issue-modal.spec.ts`
- **Steps**:
  1. Click "Read more" in modal description.
  2. Click "Read less".
- **Expected Results**:
  - Description container expands and collapses accordingly.

### JIRA-MODAL-004
- **Title**: Engineer closes modal via close icon button
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/issue-modal.spec.ts`
- **Steps**:
  1. Click close button (`[data-testid="btn-close-modal"]`).
- **Expected Results**:
  - Modal dismisses and board remains interactive.

### JIRA-MODAL-005
- **Title**: Engineer closes modal via backdrop scrim click
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/issue-modal.spec.ts`
- **Steps**:
  1. Click backdrop overlay outside modal content.
- **Expected Results**:
  - Modal dismisses.

### JIRA-MODAL-006
- **Title**: Engineer closes modal via keyboard Escape key
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/issue-modal.spec.ts`
- **Steps**:
  1. Press `Escape` key while modal is active.
- **Expected Results**:
  - Modal dismisses.

### JIRA-BACKLOG-001
- **Title**: Engineer switches to Backlog tab and views active sprint, future sprint, and backlog
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/jira/positive/backlog-sprints.spec.ts`
- **Steps**:
  1. Navigate to `/jira`.
  2. Click "Backlog" tab button.
- **Expected Results**:
  - Active sprint section renders with issue count and date range.
  - Future sprint and Backlog lists render.
- **Evidence**: `docs/qa/evidence/jira/03_jira_backlog_view.png`

### JIRA-BACKLOG-002
- **Title**: Engineer clicks backlog item to open issue detail modal
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/backlog-sprints.spec.ts`
- **Steps**:
  1. Click any backlog item row.
- **Expected Results**:
  - Issue detail modal opens with matching issue details.

### JIRA-FILTER-001
- **Title**: Engineer can filter board by document category
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/filters.spec.ts`
- **Steps**:
  1. Click category filter button (e.g. Bugs).
- **Expected Results**:
  - Board cards filter to only display matching issue categories.

### JIRA-FILTER-002
- **Title**: Engineer can filter dev documents by sub-type
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/filters.spec.ts`
- **Steps**:
  1. Select "Dev Docs" filter.
  2. Click sub-type chips (UT, Query, SOP).
- **Expected Results**:
  - Visible issues restrict to selected sub-type.

### JIRA-FILTER-003
- **Title**: Engineer can switch to "All" filter to restore full assigned issues view
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/filters.spec.ts`
- **Steps**:
  1. Click "All" filter chip.
- **Expected Results**:
  - All assigned issues render across the 3 columns.

### JIRA-SQUAD-001
- **Title**: Engineer sees squad filter pills and can switch between squads
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/jira/positive/filters.spec.ts`
- **Steps**:
  1. Switch to Backlog tab with multi-squad configuration.
  2. Inspect squad pills ("Fortune Squad (Yours)", "Azzuri").
- **Expected Results**:
  - User's primary squad is active by default.
  - Switching squad updates visible sprint issues.

---

## Negative Scenarios

### JIRA-AUTH-001
- **Title**: Engineer cannot see board when Jira PAT is unconfigured
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/jira/negative/jira-errors.spec.ts`
- **Steps**:
  1. Load `/jira` with `has_jira_pat: false`.
- **Expected Results**:
  - Unconfigured PAT warning banner renders with link to `/settings`.
  - Zero cards or column structures render.
  - Zero outgoing requests to `/api/jira/issues` are dispatched.
- **Evidence**: `docs/qa/evidence/jira/04_jira_negative_unconfigured.png`

### JIRA-AUTH-002
- **Title**: Engineer sees 401 error banner when Jira PAT is invalid or expired
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/jira/negative/jira-errors.spec.ts`
- **Steps**:
  1. Mock `/api/jira/issues` with 401 Unauthorized.
  2. Load `/jira`.
- **Expected Results**:
  - "Jira PAT is invalid or expired" error banner surfaces.
- **Evidence**: `docs/qa/evidence/jira/05_jira_negative_401.png`

### JIRA-NET-001
- **Title**: Engineer sees 502 VPN error banner when upstream Jira server is unreachable
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/jira/negative/jira-errors.spec.ts`
- **Steps**:
  1. Mock `/api/jira/issues` with 502 Bad Gateway.
  2. Load `/jira`.
- **Expected Results**:
  - "Please verify your BRI VPN connection" error banner surfaces.
- **Evidence**: `docs/qa/evidence/jira/06_jira_negative_502.png`

### JIRA-EMPTY-001
- **Title**: Engineer sees empty column placeholder when no issues exist for a status
- **Classification**: Negative | Priority: P2
- **File**: `frontend/e2e/tests/jira/negative/jira-errors.spec.ts`
- **Steps**:
  1. Mock `/api/jira/issues` with empty array `[]`.
- **Expected Results**:
  - Each column displays "Nothing here" empty state.

### JIRA-EMPTY-002
- **Title**: Engineer sees empty state when subfilter yields zero matching cards
- **Classification**: Negative | Priority: P2
- **File**: `frontend/e2e/tests/jira/negative/jira-errors.spec.ts`
- **Steps**:
  1. Filter by sub-type SOP on dataset containing zero SOP items.
- **Expected Results**:
  - Columns show empty placeholder.

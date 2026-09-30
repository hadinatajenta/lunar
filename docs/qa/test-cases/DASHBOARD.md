# Test Cases: Dashboard

This document details the canonical test cases for the central Developer Dashboard, quick navigation tiles, API contract synchronization, and unauthenticated boundary protection.

---

## Positive Scenarios

### DASH-API-001
- **Title**: Summary endpoint response matches schema contract shape with valid token
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/dashboard/positive/overview.spec.ts`
- **Steps**:
  1. Authenticate session.
  2. Request `GET /api/dashboard/summary`.
- **Expected Results**:
  - Response status is 200 OK.
  - JSON payload contains `issues_assigned`, `active_prs`, `unreviewed_docs`, `last_synced_at`, and `recent_activities`.

### DASH-VIEW-001
- **Title**: Four metric cards render accurately matching the summary API response
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/dashboard/positive/overview.spec.ts`
- **Steps**:
  1. Navigate to `/dashboard`.
  2. Intercept summary API with known metric counts.
- **Expected Results**:
  - Assigned Issues card renders "7".
  - Active Pull Requests card renders "3".
  - Unreviewed Documents card renders "2".
  - Copilot Interactions card renders count.
- **Evidence**: `docs/qa/evidence/dashboard/01_dashboard_metrics.png`

### DASH-SYNC-001
- **Title**: Sync button transitions to spinning state, restores, and updates timestamp
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/positive/overview.spec.ts`
- **Steps**:
  1. Navigate to `/dashboard`.
  2. Click "Sync" action button.
- **Expected Results**:
  - Button exhibits loading spinner during request execution.
  - Button returns to enabled idle state upon completion.
  - "Last synced" timestamp text updates.
- **Evidence**: `docs/qa/evidence/dashboard/02_dashboard_synced.png`

### DASH-NAV-001
- **Title**: Four quick action tiles navigate to their respective application routes
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/positive/overview.spec.ts`
- **Steps**:
  1. Click Jira card -> verify `/jira`.
  2. Click Bitbucket card -> verify `/bitbucket`.
  3. Click Confluence card -> verify `/confluence`.
  4. Click Copilot tile -> verify `/copilot`.
- **Expected Results**:
  - Each action tile redirects to the correct application view.
- **Evidence**: `docs/qa/evidence/dashboard/03_dashboard_quick_actions.png`

### DASH-CRED-001
- **Title**: Integration credential badges display configured or needs PAT status
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/positive/overview.spec.ts`
- **Steps**:
  1. Load dashboard with Jira configured and Bitbucket unconfigured.
- **Expected Results**:
  - Jira integration badge renders green "Configured".
  - Bitbucket integration badge renders warning "Needs PAT".
- **Evidence**: `docs/qa/evidence/dashboard/08_credential_badges.png`

---

## Negative Scenarios

### DASH-AUTH-001
- **Title**: Unauthenticated visitor is redirected to login and never sees dashboard
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/dashboard/negative/unauthenticated-redirect.spec.ts`
- **Steps**:
  1. Open new unauthenticated browser context.
  2. Attempt direct navigation to `/dashboard`.
- **Expected Results**:
  - Browser immediately redirects to `/login`.
  - Metric cards and protected workspace layout are never rendered.
- **Evidence**: `docs/qa/evidence/dashboard/04_unauthenticated_redirect.png`

### DASH-AUTH-002
- **Title**: Unauthenticated direct deep links redirect to login with return path
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/negative/unauthenticated-redirect.spec.ts`
- **Steps**:
  1. Navigate directly to `/jira`, `/bitbucket`, `/confluence`, or `/settings` without session.
- **Expected Results**:
  - In each case, redirected to `/login`.

### DASH-API-002
- **Title**: API summary request with forged or empty bearer token returns 401
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/dashboard/negative/unauthenticated-redirect.spec.ts`
- **Steps**:
  1. Send `GET /api/dashboard/summary` with invalid authorization header.
- **Expected Results**:
  - Response status is 401 Unauthorized.

### DASH-API-003
- **Title**: API summary request without authorization header returns 401
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/dashboard/negative/unauthenticated-redirect.spec.ts`
- **Steps**:
  1. Send `GET /api/dashboard/summary` without header.
- **Expected Results**:
  - Response status is 401 Unauthorized.

### DASH-SUMMARY-001
- **Title**: Summary endpoint returning 500 displays error banner and allows retry
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/negative/summary-failures.spec.ts`
- **Steps**:
  1. Mock `/api/dashboard/summary` with 500 Internal Server Error.
  2. Navigate to `/dashboard`.
  3. Click retry button after restoring healthy mock.
- **Expected Results**:
  - Error banner surfaces on dashboard.
  - Clicking retry re-requests summary and restores metric cards.
- **Evidence**: `docs/qa/evidence/dashboard/05_summary_error_banner.png`

### DASH-SUMMARY-002
- **Title**: Aborted or network-dropped summary request surfaces error banner
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/negative/summary-failures.spec.ts`
- **Steps**:
  1. Abort route `/api/dashboard/summary` on client network level.
  2. Navigate to `/dashboard`.
- **Expected Results**:
  - Network error banner surfaces.
- **Evidence**: `docs/qa/evidence/dashboard/06_summary_network_error.png`

### DASH-SYNC-002
- **Title**: Failed manual sync restores button state and displays failure toast
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/dashboard/negative/summary-failures.spec.ts`
- **Steps**:
  1. Trigger sync button while sync API endpoint returns 500 error.
- **Expected Results**:
  - Spinner dismisses and button re-enables.
  - Global error toast displays sync failure notice.
- **Evidence**: `docs/qa/evidence/dashboard/07_sync_failure_toast.png`

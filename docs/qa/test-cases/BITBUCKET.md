# Test Cases: Bitbucket BRI

This document details the canonical test cases for the Bitbucket BRI workspace covering Recent Pushes, Pull Request Review, Automated AI Code Review, PR Creation, and Error Handling.

---

## Positive Scenarios

### BB-VIEW-001
- **Title**: Engineer views Bitbucket workspace with push cards, PR table, and AI review indicators
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/bitbucket/positive/overview.spec.ts`
- **Preconditions**: Bitbucket PAT configured in vault.
- **Steps**:
  1. Navigate to `/bitbucket`.
- **Expected Results**:
  - Recent Pushes feed renders branch names, commit hashes, and status tags.
  - Active Pull Requests table renders with columns for PR #, repo, author, status, and actions.
  - AI review status pills (e.g. "Review Ready", "Flagged") render accurately.
- **Evidence**: `docs/qa/evidence/bitbucket/01_bitbucket_overview.png`

### BB-PUSH-001
- **Title**: Engineer filters recent pushes by "Ready" status
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/filters.spec.ts`
- **Steps**:
  1. Click "Ready" push filter button.
- **Expected Results**:
  - Only pushes in "ready" state render.
- **Evidence**: `docs/qa/evidence/bitbucket/02_pushes_filter_ready.png`

### BB-PUSH-002
- **Title**: Engineer filters recent pushes by "Stale" status
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/filters.spec.ts`
- **Steps**:
  1. Click "Stale" push filter button.
- **Expected Results**:
  - Only pushes exceeding threshold render.

### BB-PUSH-003
- **Title**: Engineer restores all pushes via "All" filter
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/filters.spec.ts`
- **Steps**:
  1. Click "All" push filter button.
- **Expected Results**:
  - Full set of recent pushes displays.

### BB-PR-001
- **Title**: Engineer filters PR table by "AI Flagged" pull requests
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/filters.spec.ts`
- **Steps**:
  1. Click "AI Flagged" filter chip.
- **Expected Results**:
  - Table filters to PRs containing flagged automated findings.
- **Evidence**: `docs/qa/evidence/bitbucket/03_prs_filter_ai_flagged.png`

### BB-PR-002
- **Title**: Engineer filters PR table by "Assigned to Me"
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/filters.spec.ts`
- **Steps**:
  1. Click "Assigned to Me" filter chip.
- **Expected Results**:
  - Table filters to PRs assigned to the active user.

### BB-PR-003
- **Title**: Engineer restores all pull requests via "All" filter
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/filters.spec.ts`
- **Steps**:
  1. Click "All" PR filter chip.
- **Expected Results**:
  - Full PR list is restored.

### BB-PR-004
- **Title**: Engineer creates a pull request from a push card and sees success confirmation
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/bitbucket/positive/create-pr.spec.ts`
- **Steps**:
  1. Click "Create PR" on push card `push-2`.
  2. Modal opens with pre-filled source branch.
  3. Fill title, target branch, and description.
  4. Submit "Create Pull Request".
- **Expected Results**:
  - Success toast surfaces with link to PR.
- **Evidence**: `docs/qa/evidence/bitbucket/04_create_pr_modal.png`, `docs/qa/evidence/bitbucket/05_create_pr_toast.png`

### BB-REV-001
- **Title**: Engineer opens PR review modal and inspects model selector options
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/pr-review.spec.ts`
- **Steps**:
  1. Click "Review" button on PR #184.
  2. Inspect AI model dropdown in review header.
- **Expected Results**:
  - Configured models are available (e.g. DeepSeek-V4 Pro, DeepSeek Flash).
  - Selected model option updates successfully.

### BB-REV-002
- **Title**: PR review modal displays warning and disables generation when no AI keys configured
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/pr-review.spec.ts`
- **Steps**:
  1. Open review modal with `has_ai_keys: false`.
- **Expected Results**:
  - "AI Key Required for Automated Code Review" warning renders.
  - "Generate AI Review" button is disabled.

### BB-REV-003
- **Title**: Engineer inspects diff, generates AI review, and verifies findings in review textarea
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/bitbucket/positive/pr-review.spec.ts`
- **Steps**:
  1. Open review modal for PR #184.
  2. Inspect diff viewer container.
  3. Click "Generate AI Review".
- **Expected Results**:
  - AI review findings populate with security and lint analysis.
  - Formatted findings automatically copy into the editable review comment box.
- **Evidence**: `docs/qa/evidence/bitbucket/06_pr_review_modal_empty.png`, `docs/qa/evidence/bitbucket/07_pr_review_ai_generated.png`

### BB-REV-006
- **Title**: Engineer approves pull request and sees status confirmation toast
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/pr-review.spec.ts`
- **Steps**:
  1. Open review modal for PR #188.
  2. Click "Approve" action button.
- **Expected Results**:
  - Confirmation toast informs user the PR has been approved.
- **Evidence**: `docs/qa/evidence/bitbucket/08_pr_review_status_action.png`

### BB-REV-007
- **Title**: Engineer posts review comment and sees confirmation toast
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/positive/pr-review.spec.ts`
- **Steps**:
  1. Open review modal for PR #191.
  2. Type custom comment in textarea.
  3. Click "Post Comment".
- **Expected Results**:
  - Review comment is saved to Bitbucket PR.
  - Confirmation toast surfaces.
- **Evidence**: `docs/qa/evidence/bitbucket/09_pr_review_comment_posted.png`

---

## Negative Scenarios

### BB-AUTH-001
- **Title**: Engineer sees 401 error banner when Bitbucket PAT is invalid or expired
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/bitbucket/negative/bitbucket-errors.spec.ts`
- **Steps**:
  1. Mock `/api/bitbucket/pushes` with 401 Unauthorized.
  2. Load `/bitbucket`.
- **Expected Results**:
  - Error banner displays "Bitbucket PAT is invalid or expired".
- **Evidence**: `docs/qa/evidence/bitbucket/11_bitbucket_invalid_pat_negative.png`

### BB-NET-001
- **Title**: Engineer sees 502 VPN error banner when upstream Bitbucket server is unreachable
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/bitbucket/negative/bitbucket-errors.spec.ts`
- **Steps**:
  1. Mock `/api/bitbucket/pushes` with 502 Bad Gateway.
  2. Load `/bitbucket`.
- **Expected Results**:
  - Error banner displays "Please verify your BRI VPN connection".
- **Evidence**: `docs/qa/evidence/bitbucket/12_bitbucket_vpn_disconnect_negative.png`

### BB-AUTH-002
- **Title**: Engineer sees 403 forbidden error banner when user lacks repository permissions
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/negative/bitbucket-errors.spec.ts`
- **Steps**:
  1. Mock `/api/bitbucket/pushes` with 403 Forbidden.
  2. Load `/bitbucket`.
- **Expected Results**:
  - Error banner informs user that their credentials lack required permissions.
- **Evidence**: `docs/qa/evidence/bitbucket/13_bitbucket_access_forbidden_negative.png`

### BB-AUTH-003
- **Title**: Engineer sees unconfigured PAT warning banner when Bitbucket PAT is missing
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/bitbucket/negative/bitbucket-errors.spec.ts`
- **Steps**:
  1. Load `/bitbucket` with `has_bitbucket_pat: false`.
- **Expected Results**:
  - Warning banner displays link to `/settings`.
  - Zero push cards or PR rows render.
- **Evidence**: `docs/qa/evidence/bitbucket/14_bitbucket_no_pat_configured.png`

### BB-REV-004
- **Title**: Review modal displays error banner when diff endpoint returns 500 error
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/negative/bitbucket-errors.spec.ts`
- **Steps**:
  1. Mock `/api/bitbucket/prs/184/diff` with 500.
  2. Open review modal.
- **Expected Results**:
  - Diff loading failure message is displayed in modal.

### BB-REV-005
- **Title**: Review modal displays error banner when AI review endpoint fails
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/bitbucket/negative/bitbucket-errors.spec.ts`
- **Steps**:
  1. Mock `/api/bitbucket/prs/184/review` with 500.
  2. Click "Generate AI Review".
- **Expected Results**:
  - AI generation failure alert is displayed.

---

## Edge Scenarios

### BB-VIEW-002
- **Title**: Engineer sees empty states when no pushes or pull requests exist
- **Classification**: Edge | Priority: P2
- **File**: `frontend/e2e/tests/bitbucket/edge/empty-states.spec.ts`
- **Steps**:
  1. Mock pushes and PRs with empty arrays.
  2. Navigate to `/bitbucket`.
- **Expected Results**:
  - Push section renders "No recent pushes found".
  - PR table renders empty state row.
- **Evidence**: `docs/qa/evidence/bitbucket/10_bitbucket_empty_state.png`

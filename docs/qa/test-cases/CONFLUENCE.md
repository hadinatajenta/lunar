# Test Cases: Confluence BRI

This document details the canonical test cases for the Confluence BRI workspace covering Document Browsing, Search, Filters, Detail View, Document Actions, and Network Error Handling.

---

## Positive Scenarios

### CONF-VIEW-001
- **Title**: Engineer views Confluence workspace with statistics grid, cards, and counts
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/confluence/positive/document-list.spec.ts`
- **Preconditions**: Confluence PAT configured in vault.
- **Steps**:
  1. Navigate to `/confluence`.
- **Expected Results**:
  - Stat cards display counts for UT, QR, SOP, and All documents.
  - Initial 6 document cards render.
  - "Showing 6 of 6" count text visible.

### CONF-PAGE-001
- **Title**: Show more button reveals additional cards when dataset exceeds 6 items
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-list.spec.ts`
- **Steps**:
  1. Load Confluence with 14 documents.
  2. Click "Show more" button.
- **Expected Results**:
  - Card count expands from 6 to 12.
  - Secondary click expands from 12 to 14.
  - "Show more" button dismisses when all cards are visible.

### CONF-SEARCH-001
- **Title**: Search input dynamically filters document cards by title
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-list.spec.ts`
- **Steps**:
  1. Type "UT-101" into search bar.
  2. Clear search bar.
- **Expected Results**:
  - Filtered results display only 1 matching card (`UT-101`).
  - Clearing search restores all 6 cards.

### CONF-FILTER-001
- **Title**: Filter chips switch document types and show correct counts
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-list.spec.ts`
- **Steps**:
  1. Click "Query" filter chip.
  2. Click "SOP" filter chip.
  3. Click "All" filter chip.
- **Expected Results**:
  - Filter chips highlight active state (`aria-current="true"`).
  - Cards update according to selected document type.

### CONF-DETAIL-001
- **Title**: Clicking document card navigates to detail view with badges and metadata
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/confluence/positive/document-detail.spec.ts`
- **Steps**:
  1. Click document card `UT-101`.
- **Expected Results**:
  - URL updates to `/confluence/UT-101`.
  - Detail title, type badge ("UT"), status pill ("Open"), and description render.
  - Metadata shows space "Engineering / Auth" and owner "Eren".

### CONF-NAV-001
- **Title**: SOP document hides Instant Apply and Copy buttons
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-detail.spec.ts`
- **Steps**:
  1. Open document `SOP-301`.
- **Expected Results**:
  - "Instant apply" button is hidden.
  - "Copy description" button is hidden.
  - "Generate SOP with AI" button is visible.

### CONF-NAV-002
- **Title**: Non-SOP document shows Instant Apply and Copy buttons
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-detail.spec.ts`
- **Steps**:
  1. Open document `UT-101`.
- **Expected Results**:
  - "Instant apply" and "Copy description" buttons are visible.

### CONF-EXT-001
- **Title**: Open in Confluence BRI button links to valid document URL
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-detail.spec.ts`
- **Steps**:
  1. Open document `UT-101`.
  2. Inspect "Open in Confluence" button.
- **Expected Results**:
  - Link has `href="https://confluence.bri.co.id/pages/viewpage.action?pageId=UT-101"`.
  - Link has `target="_blank"` and `rel="noopener noreferrer"`.

### CONF-CACHE-001
- **Title**: Revisit list does not refetch from backend when documents already cached
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/confluence/positive/document-detail.spec.ts`
- **Steps**:
  1. Load `/confluence`.
  2. Navigate to `/dashboard` and return to `/confluence`.
- **Expected Results**:
  - No new outgoing network calls dispatched to `/api/confluence/documents`.

### CONF-ACT-001
- **Title**: New page button triggers placeholder toast
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/confluence/positive/document-actions.spec.ts`
- **Steps**:
  1. Click "New page" button in document toolbar.
- **Expected Results**:
  - Toast notification displays "New page is ready for backend wiring."

### CONF-ACT-002
- **Title**: AI generate button triggers action toast for document type
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/confluence/positive/document-actions.spec.ts`
- **Steps**:
  1. Open document detail view.
  2. Click "Generate with AI" button.
- **Expected Results**:
  - Toast notification displays "AI generation is ready for backend wiring."

### CONF-ACT-003
- **Title**: Instant apply, Edit, Share, and More buttons trigger placeholder toasts
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/confluence/positive/document-actions.spec.ts`
- **Steps**:
  1. Click Edit, Share, More, and Instant Apply buttons.
- **Expected Results**:
  - Each action triggers its corresponding placeholder notification.

### CONF-ACT-004
- **Title**: Commenting action triggers toast and validates input
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/confluence/positive/document-actions.spec.ts`
- **Steps**:
  1. Type comment text and click Post.
  2. Click Post with empty input.
- **Expected Results**:
  - Non-empty post displays placeholder toast.
  - Empty post focuses input textarea.

### CONF-CLIP-001
- **Title**: Copy button copies description to clipboard on non-SOP doc
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/confluence/positive/document-actions.spec.ts`
- **Steps**:
  1. Grant clipboard permissions.
  2. Click "Copy" on non-SOP document.
- **Expected Results**:
  - Toast displays "Description copied to clipboard."

---

## Negative Scenarios

### CONF-AUTH-001
- **Title**: Engineer cannot see documents when Confluence PAT is unconfigured
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/confluence/negative/confluence-errors.spec.ts`
- **Steps**:
  1. Load `/confluence` with `has_confluence_pat: false`.
- **Expected Results**:
  - Unconfigured PAT warning banner renders.
  - Zero cards render and zero requests to `/api/confluence/documents` dispatched.

### CONF-AUTH-002
- **Title**: Engineer sees 401 error banner when Confluence PAT is invalid or expired
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/confluence/negative/confluence-errors.spec.ts`
- **Steps**:
  1. Mock `/api/confluence/documents` with 401.
  2. Load `/confluence`.
- **Expected Results**:
  - Error banner displays invalid PAT notice.

### CONF-NET-001
- **Title**: Engineer sees 502 VPN error banner when upstream Confluence is unreachable
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/confluence/negative/confluence-errors.spec.ts`
- **Steps**:
  1. Mock `/api/confluence/documents` with 502.
  2. Load `/confluence`.
- **Expected Results**:
  - Error banner displays BRI VPN connection notice.

### CONF-DETAIL-002
- **Title**: Detail deep link shows back navigation when API returns 404
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/confluence/negative/confluence-errors.spec.ts`
- **Steps**:
  1. Navigate to `/confluence/NONEXISTENT`.
  2. Mock detail endpoint with 404.
- **Expected Results**:
  - Back button is visible with link to `/confluence`.

### CONF-EXT-002
- **Title**: Open in Confluence BRI shows toast when document URL is missing
- **Classification**: Negative | Priority: P2
- **File**: `frontend/e2e/tests/confluence/negative/confluence-errors.spec.ts`
- **Steps**:
  1. Open document `DOC-401` which has empty URL.
  2. Click "Open in Confluence".
- **Expected Results**:
  - Toast surfaces: "This document has no Confluence URL yet."

---

## Edge Scenarios

### CONF-VIEW-002
- **Title**: Engineer sees empty state when no documents exist
- **Classification**: Edge | Priority: P2
- **File**: `frontend/e2e/tests/confluence/edge/search-empty.spec.ts`
- **Steps**:
  1. Mock documents endpoint with `[]`.
- **Expected Results**:
  - Empty state container renders and stat card for All displays "0".

### CONF-SEARCH-002
- **Title**: Engineer sees empty state when search query yields zero matches
- **Classification**: Edge | Priority: P2
- **File**: `frontend/e2e/tests/confluence/edge/search-empty.spec.ts`
- **Steps**:
  1. Type non-matching string into search bar.
- **Expected Results**:
  - Empty state placeholder renders.

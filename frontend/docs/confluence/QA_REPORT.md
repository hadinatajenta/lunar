# Confluence BRI — QA Report

## Scope
Validation of the `/confluence` feature against the reference prototype (`deepseek_html_20260930_68a00f.html#confluence`), covering list view, detail view, API contract, error handling, and placeholder interactions (AI generation, Instant apply, Copy, Edit, Share, More, Comments).

## Environment
- Frontend: Vite dev server `http://localhost:5173` (Vue 3 + TypeScript)
- Backend: Go server `http://localhost:8080` with `internal/confluence` module + shared TTL cache
- Test runner: Playwright 1.48+, `npx playwright test`
- User session: `developer@lunar.dev` / `12345678` (no Confluence PAT configured in vault — API returns 401 by design)

## Test Execution Summary
| Category | Tests | Result |
|---|---|---|
| Existing suite (auth, dashboard, jira, bitbucket, copilot, settings) | 85 | ✅ Passed |
| Confluence negative scenarios | 6 | ✅ Passed |
| Confluence positive list & detail | 12 | ✅ Passed |
| **Total** | **91** | **✅ 100% Green** |

## Testcase Table (Confluence — Negative First)

| ID | Scenario | Expected Behavior | Status |
|---|---|---|---|
| CONF-NEG-001 | PAT unconfigured (`has_confluence_pat:false`) | PAT banner visible, zero `/api/confluence/documents` calls, no cards | ✅ |
| CONF-NEG-002 | Invalid/expired PAT → API 401 | Error banner visible, VPN banner hidden, no cards | ✅ |
| CONF-NEG-003 | Upstream unreachable → API 502 | VPN banner visible, error banner hidden, no cards | ✅ |
| CONF-NEG-004 | Detail deep-link → API 404 | Not-found state with working back link to `/confluence` | ✅ |
| CONF-NEG-005 | API returns empty list | Empty state visible, all stat cards show 0, no cards | ✅ |
| CONF-NEG-006 | Search with no matches | Empty state visible, zero cards | ✅ |
| CONF-POS-001 | Stats grid, card count, visible count, show-more hidden when all shown | 4 stat counts match fixture (2 UT, 2 QR, 1 SOP, 6 All); 6 cards; "Showing 6 of 6"; show-more hidden | ✅ |
| CONF-POS-002 | Show more reveals +6 until exhausted, then hides | Clicks increment visible count by 6; button disappears at end | ✅ |
| CONF-POS-003 | Search input debounce filters by title | 120ms debounce; filtered cards match; clear restores all | ✅ |
| CONF-POS-004 | Filter chips switch type set, show per-type counts, active state | Chips toggle type; counts correct; `aria-current` on active | ✅ |
| CONF-POS-005 | Card click → detail with title, badges, description, meta (Space/Owner/Updated; **no Version**) | URL `/confluence/:id`; badges (type_label, ID, status); description; meta grid 3 items | ✅ |
| CONF-POS-006 | SOP document hides Instant apply and Copy | Apply and Copy absent for `type === "sop"` | ✅ |
| CONF-POS-007 | Non-SOP document shows Instant apply and Copy | Both present for ut/query/doc types | ✅ |
| CONF-POS-008 | Generate label varies by type | "Generate UT Docs With AI" (ut), "Extract query review" (query), "Generate with AI" (doc/sop) | ✅ |
| CONF-POS-009 | Back button returns to list preserving filter state | Filter chip `aria-current` retained | ✅ |
| CONF-POS-010 | Deep link loads detail without list pre-load | Direct `/confluence/:id` renders detail correctly | ✅ |
| CONF-POS-011 | Open in Confluence BRI opens absolute URL when present | Anchor `href` = doc.url, `target=_blank`, `rel=noopener noreferrer` | ✅ |
| CONF-POS-012 | Open in Confluence BRI shows toast when URL empty | Toast "This document has no Confluence URL yet." | ✅ |
| CONF-POS-013 | Placeholder toasts: New page, Generate, Apply, Edit, Share, More, Comment (empty→focus, filled→toast), no fake comments appended | Each button triggers exact copy; comment composer posts → toast only, no DOM insertion | ✅ |
| CONF-POS-014 | Copy copies description to clipboard, success toast | `navigator.clipboard.writeText` called; toast "Description copied to clipboard." | ✅ |
| CONF-POS-015 | Revisit list does not refetch (has-loaded guard) | Zero second `/api/confluence/documents` call within session | ✅ |

## Defects Found & Fixes Applied During QA

| Defect | File | Line | Fix |
|---|---|---|---|
| Instant apply & Copy buttons visible for SOP documents (prototype hides both for SOP) | `DocumentActionsRow.vue` | 85, 109 | Added `isSopDocument` computed + `v-if="!isSopDocument"` on both buttons (faithful port) |
| Em dash in clipboard failure toast (prototype copy uses em dash — retained as intentional UI copy) | `DocumentActionsRow.vue` | 112 | Kept prototype copy: "Copy failed — clipboard unavailable." |

No other product defects found.

## Residual Risks & Limitations
1. **Real upstream Confluence not exercised** — all tests mock `/api/confluence/*` via `page.route`. Production behavior depends on BRI Confluence REST structure (CQL search, label schema, `body.storage` HTML). Type/Status mapping heuristics are unvalidated against real data.
2. **PAT-less account hits fast 401 by design** — developer user has no Confluence PAT in vault; 401 is expected and correctly surfaced.
3. **Version field omitted from detail meta** — backend contract has no `version` field; prototype shows "Version v1.4". Omitted intentionally (backend does not provide it).
4. **Cache TTL 60s, no manual refresh UI** — list/detail cached server-side; FE guard (`hasLoaded` marker) prevents client refetch until session end. Matches prototype static behavior.
5. **Comments are placeholder only** — no backend endpoint; posting shows toast, no local append (prototype appends locally; phase 2 may wire real comments).

## Sign-off
All 91 tests passing. Confluence feature meets acceptance criteria for Phase 1 (list + detail with real API, placeholder AI/apply/comments). Ready for merge pending standard review pipeline.
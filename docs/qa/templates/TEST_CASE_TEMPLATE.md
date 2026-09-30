# [TEST-ID] Test Case Specification

## 1. Metadata
- **Test Case ID**: `[FEATURE]-[AREA]-[NUMBER]`
- **Title**: Descriptive title of test scenario
- **Feature**: Auth | Dashboard | Copilot | Jira | Bitbucket | Confluence | Settings
- **Classification**: Positive | Negative | Edge
- **Priority**: P0 (Blocker) | P1 (Critical) | P2 (Normal)
- **Automation Status**: Automated | Manual | Pending
- **Automation File**: `frontend/e2e/tests/[feature]/[classification]/[spec].spec.ts`

## 2. Description & Objective
Concise explanation of what user behavior or system invariant is being verified.

## 3. Preconditions
- State of user authentication (unauthenticated vs active session).
- Mock route states or credentials configured in credential vault.
- Initial navigation endpoint.

## 4. Execution Steps
1. Navigate to target URL.
2. Perform user action (input text, click button, trigger toggle).
3. Assert system reaction (DOM update, URL redirect, toast notification).

## 5. Expected Results
- Explicit behavioral assertions.
- Error banner visibility and text when testing failure scenarios.
- Network API requests and payloads emitted.

## 6. Evidence & Traceability
- **Screenshot Path**: `docs/qa/evidence/[feature]/[filename].png`
- **Trace Path**: `frontend/test-results/[test-slug]/trace.zip`

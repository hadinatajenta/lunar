# Feature QA Execution Report: [Feature Name]

## 1. Executive Summary
- **Feature Area**: [Feature Name]
- **Execution Date**: YYYY-MM-DD
- **Environment**: Local / Chromium Headless (Playwright v1.63.0)
- **Total Scenarios**: Total count
- **Pass Rate**: X% (Passed / Total)
- **Status**: PASSED | BLOCKED | DEGRADED

## 2. Test Execution Breakdown
| Test Case ID | Classification | Scenario Summary | Execution Time | Status | Evidence |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `ID-001` | Positive | Successful happy path | 120ms | PASSED | `01_screen.png` |
| `ID-002` | Negative | Error handling validation | 85ms | PASSED | `02_error.png` |

## 3. Honest Coverage & Reality Assessment
- **UI Interaction vs Backend Verification**: Detail what is verified on the client vs mocked or backend executed.
- **Mock Dependencies**: List mocked API routes.
- **Unverified Behaviors**: Highlight aspects not yet tested by automation (e.g. live token revocation, long network drop).

## 4. Known Gaps & Prioritized Next Steps
- **P0**: Critical missing tests.
- **P1**: Important edge cases.
- **P2**: Non-blocking cosmetic or performance assertions.

# QA Automation Architecture & Test System

The Lunar QA automation suite is a modular, production-grade Playwright end-to-end testing and verification framework. It validates critical user journeys, API contract integrity, failure modes, boundary behaviors, and security protections across the Lunar developer portal.

---

## 1. Directory Structure

```text
lunar/
├── docs/
│   └── qa/
│       ├── README.md                      # Central QA manual and architecture guide
│       ├── templates/                     # Standardized QA specification and reporting templates
│       │   ├── TEST_CASE_TEMPLATE.md
│       │   ├── FEATURE_REPORT_TEMPLATE.md
│       │   └── RELEASE_REPORT_TEMPLATE.md
│       ├── test-cases/                    # Canonical test case specifications per feature
│       │   ├── AUTH.md
│       │   ├── DASHBOARD.md
│       │   ├── COPILOT.md
│       │   ├── JIRA.md
│       │   ├── BITBUCKET.md
│       │   ├── CONFLUENCE.md
│       │   └── SETTINGS.md
│       ├── reports/
│       │   ├── latest/                    # Current test cycle reports and coverage matrix
│       │   │   ├── SUMMARY.md
│       │   │   ├── AUTH.md
│       │   │   ├── DASHBOARD.md
│       │   │   ├── COPILOT.md
│       │   │   ├── JIRA.md
│       │   │   ├── BITBUCKET.md
│       │   │   ├── CONFLUENCE.md
│       │   │   └── SETTINGS.md
│       │   └── releases/                  # Historical release validation reports
│       │       └── 2026-09-30-v1.0/
│       └── evidence/                      # Visual verification screenshots by feature
│           ├── auth/
│           ├── dashboard/
│           ├── copilot/
│           │   └── legacy/
│           ├── jira/
│           ├── bitbucket/
│           ├── confluence/
│           └── settings/
│
└── frontend/
    ├── playwright.config.ts               # Core Playwright configuration
    └── e2e/
        ├── tests/                         # Feature-based, classified test specifications
        │   ├── auth/                      # positive/, negative/
        │   ├── dashboard/                 # positive/, negative/
        │   ├── copilot/                   # positive/, negative/, edge/
        │   ├── jira/                      # positive/, negative/
        │   ├── bitbucket/                 # positive/, negative/, edge/
        │   ├── confluence/                # positive/, negative/, edge/
        │   └── settings/                  # positive/
        ├── pages/                         # Page Object Model abstractions
        ├── components/                    # Component Objects (modals, boards, pickers)
        ├── fixtures/                      # Reusable Playwright test fixtures
        │   ├── authenticated.fixture.ts
        │   └── unauthenticated.fixture.ts
        ├── data/                          # Isolated mock datasets and payload fixtures
        ├── helpers/                       # Route mockers and portable evidence capture
        │   ├── evidence.ts
        │   └── route-mock.ts
        └── setup/                         # Global session authentication setup
            └── auth.setup.ts
```

---

## 2. Test Case ID Convention

All automated test specifications and documentation share a stable, deterministic Test Case ID formatted as:

`<FEATURE>-<AREA>-<NUMBER>`

### Feature Identifiers
- `AUTH`: Authentication, registration, token refresh, and session storage
- `DASH`: Central developer dashboard, quick actions, metric synchronizations
- `CP`: AI Copilot chat workspace, model selection, reasoning effort, tools, and error gates
- `JIRA`: Jira BRI board, issue detail dialog, backlog, sprint filtering, and cache
- `BB`: Bitbucket code review, push events, PR creation, AI automated review, diff viewer
- `CONF`: Confluence BRI documentation browser, search, actions, external deep links
- `SETTINGS`: Integrations vault, Atlassian PATs, AI API keys, security encryption

### Examples
- `AUTH-LOGIN-001`: Engineer submits valid credentials and reaches dashboard
- `CP-CONV-001`: Engineer sends chat prompt and receives AI stream response
- `JIRA-BOARD-001`: Engineer inspects 3-column Kanban board with default card limits
- `BB-REV-003`: Engineer generates automated AI diff review and inspects findings
- `CONF-ACT-001`: Engineer triggers new page creation toast placeholder

---

## 3. Test Classification Taxonomy

Tests are partitioned into three behavioral categories:

1. `positive/`: Valid user workflows, successful integrations, expected state transitions, and responsive visual updates.
2. `negative/`: Authentication rejections, 401 invalid PATs, 403 forbidden states, 500 server crashes, 502 VPN disconnections, timeout aborts, and input validation failures.
3. `edge/`: Boundary inputs, empty collections, rapid toggle switches, zero-match searches, session deletions, and client caching behaviors.

---

## 4. Test Execution Commands

From the `frontend/` directory:

```bash
# Run all end-to-end tests
npm run test:e2e

# Run with interactive Playwright UI
npm run test:e2e:ui

# Run with visible Chromium browser
npm run test:e2e:headed

# Run feature-specific test suites
npm run test:e2e:auth
npm run test:e2e:dashboard
npm run test:e2e:copilot
npm run test:e2e:jira
npm run test:e2e:bitbucket
npm run test:e2e:confluence
npm run test:e2e:settings

# Run classification filters via tags
npm run test:e2e:smoke       # P0 critical path tests
npm run test:e2e:positive    # Positive scenarios only
npm run test:e2e:negative    # Error handling and validation tests
npm run test:e2e:edge        # Boundary and edge case tests
```

---

## 5. Security & Sensitive Data Policy

1. **Zero Hardcoded Secrets**: Real passwords, API keys, and personal access tokens must never be committed to git or hardcoded in test files.
2. **Environment Variable Injection**: Credentials are read exclusively from process environment variables (`E2E_USER_EMAIL`, `E2E_USER_PASSWORD`).
3. **Session State Isolation**: `playwright/.auth/user.json` stores authenticated browser cookies and local storage tokens locally. It is strictly excluded by `.gitignore`.
4. **Deterministic Route Mocking**: Offline suites and negative tests isolate dependencies via `routeSecrets`, `routeSystemConfig`, and mock route responders without issuing unmonitored external network traffic.

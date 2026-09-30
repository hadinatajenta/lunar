# Test Cases: Settings & Credential Vault

This document details the canonical test cases for Settings, Atlassian Personal Access Tokens, AI Provider API Keys, and Security Encryption Status.

---

## Positive Scenarios

### SETTINGS-001
- **Title**: Engineer views Settings workspace with Integrations tab open by default
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/settings/positive/settings.spec.ts`
- **Preconditions**: Authenticated user session.
- **Steps**:
  1. Navigate to `/settings`.
- **Expected Results**:
  - Page heading displays "Settings".
  - Integrations tab is active by default.
  - Jira, Bitbucket, and Confluence PAT input fields are visible.
- **Evidence**: `docs/qa/evidence/settings/01_settings_integrations_tab.png`

### SETTINGS-002
- **Title**: Engineer saves Atlassian PATs and receives success confirmation
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/settings/positive/settings.spec.ts`
- **Steps**:
  1. Navigate to `/settings`.
  2. Input Jira PAT, Bitbucket username, Bitbucket PAT, and Confluence PAT.
  3. Click "Save Atlassian Integrations".
- **Expected Results**:
  - `PUT /api/auth/secrets` request is dispatched with encrypted credentials.
  - Green success banner toast confirms settings saved.
- **Evidence**: `docs/qa/evidence/settings/02_settings_integrations_filled.png`, `docs/qa/evidence/settings/03_settings_integrations_saved.png`

### SETTINGS-003
- **Title**: Engineer views AI providers tab with model labels for each provider
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/settings/positive/settings.spec.ts`
- **Steps**:
  1. Click "AI providers" tab button.
- **Expected Results**:
  - OpenAI row displays subtitle "GPT-6 Astra".
  - Claude row displays subtitle "Claude Opus 5.5".
  - Gemini row displays subtitle "Gemini 3.8 Flash".
  - DeepSeek row displays subtitle "deepseek-v4-pro".
  - Xiaomi MiMo row displays subtitle "mimo-v2.5-pro".
- **Evidence**: `docs/qa/evidence/settings/04_settings_ai_tab.png`

### SETTINGS-004
- **Title**: Engineer saves Gemini API key and receives success toast
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/settings/positive/settings.spec.ts`
- **Steps**:
  1. Switch to AI providers tab.
  2. Enter Google Gemini API key.
  3. Click Save button in Gemini card.
- **Expected Results**:
  - Success toast surfaces confirming Gemini key persistence.
- **Evidence**: `docs/qa/evidence/settings/05_settings_ai_saved.png`

### SETTINGS-005
- **Title**: Engineer saves DeepSeek API key and receives success toast
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/settings/positive/settings.spec.ts`
- **Steps**:
  1. Switch to AI providers tab.
  2. Enter DeepSeek API key.
  3. Click Save button in DeepSeek card.
- **Expected Results**:
  - Success toast surfaces confirming DeepSeek key persistence.

### SETTINGS-006
- **Title**: Engineer navigates to Security tab and inspects AES-256-GCM encryption status
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/settings/positive/settings.spec.ts`
- **Steps**:
  1. Click "Security" tab button.
- **Expected Results**:
  - Security card title displays "Security & Encryption".
  - Zero plain-text credentials displayed.
  - Encryption status badge confirms active cryptographic protection.
- **Evidence**: `docs/qa/evidence/settings/06_settings_security_tab.png`

# Test Cases: AI Copilot

This document details the canonical test cases for the Lunar AI Copilot workspace covering Model Selection, Thinking Controls, Reasoning Effort, Streaming Conversation, Tool Integration, Access Gates, and Session Lifecycle.

---

## Positive Scenarios

### CP-VIEW-001
- **Title**: Engineer sees clean empty chat state with personalized greeting when configured
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/copilot/positive/chat-conversation.spec.ts`
- **Preconditions**: AI provider API keys configured in vault.
- **Steps**:
  1. Navigate to `/copilot`.
- **Expected Results**:
  - Welcome greeting header is displayed.
  - Suggestion prompt chips are visible.
  - Message input textarea is focused.
- **Evidence**: `docs/qa/evidence/copilot/05_copilot_initial.png`

### CP-CONV-001
- **Title**: Engineer sends prompt and receives streaming AI response with thought process and sources
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/copilot/positive/chat-conversation.spec.ts`
- **Steps**:
  1. Navigate to `/copilot`.
  2. Type prompt: "Summarize recent Jira issues for CRMMS project".
  3. Submit prompt via Send button or Enter.
- **Expected Results**:
  - User message bubble renders in timeline.
  - Assistant responds with streamed answer.
  - Collapsible Thought Process box renders.
  - Reference citation sources render below the message.
- **Evidence**: `docs/qa/evidence/copilot/06_copilot_response_received.png`

### CP-CONV-002
- **Title**: Engineer can toggle visibility of AI thought process section
- **Classification**: Positive | Priority: P2
- **File**: `frontend/e2e/tests/copilot/positive/chat-conversation.spec.ts`
- **Steps**:
  1. Locate completed AI message with thought box.
  2. Click "Thought process" collapsible header.
- **Expected Results**:
  - Thought content toggles between expanded and collapsed states.

### CP-MODEL-001
- **Title**: Engineer only sees and can select configured models in model picker
- **Classification**: Positive | Priority: P0
- **File**: `frontend/e2e/tests/copilot/positive/model-selection.spec.ts`
- **Preconditions**: Credential vault configured with DeepSeek and Gemini only.
- **Steps**:
  1. Click model selector dropdown.
- **Expected Results**:
  - Only models from configured providers are listed.
  - Unconfigured provider models (e.g. Claude, OpenAI) are hidden.
  - Selecting a model updates the active model chip.
- **Evidence**: `docs/qa/evidence/copilot/02_copilot_model_picker_filtered.png`

### CP-THINK-001
- **Title**: Engineer can toggle Thinking mode ON and OFF
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/copilot/positive/controls.spec.ts`
- **Steps**:
  1. Inspect thinking toggle in prompt bar (default: ON).
  2. Click toggle button.
- **Expected Results**:
  - Toggle switches to Thinking OFF.
  - Subsequent request payload sets `thinking: false`.

### CP-REASON-001
- **Title**: Engineer can adjust reasoning effort level via dropdown
- **Classification**: Positive | Priority: P1
- **File**: `frontend/e2e/tests/copilot/positive/controls.spec.ts`
- **Steps**:
  1. Click reasoning effort selector (Low, Medium, High).
  2. Select "High".
- **Expected Results**:
  - Active effort badge updates to "High".
  - Subsequent request payload sets `reasoning_effort: "high"`.

---

## Negative Scenarios

### CP-GATE-001
- **Title**: Access gate surfaces warning banner and link to Settings when no AI keys configured
- **Classification**: Negative | Priority: P0
- **File**: `frontend/e2e/tests/copilot/negative/access-gate.spec.ts`
- **Preconditions**: User has no AI provider API keys configured.
- **Steps**:
  1. Navigate to `/copilot`.
- **Expected Results**:
  - Chat input is disabled.
  - "Configure AI Provider" gate banner is visible with link to `/settings`.
- **Evidence**: `docs/qa/evidence/copilot/01_copilot_gate_unconfigured.png`

### CP-ERR-001
- **Title**: Engineer receives clear error message when upstream AI returns 401 Invalid Key
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/copilot/negative/upstream-errors.spec.ts`
- **Steps**:
  1. Mock Copilot chat stream with 401 Unauthorized.
  2. Send prompt.
- **Expected Results**:
  - In-chat error alert notifies engineer that the provider API key is invalid or revoked.
- **Evidence**: `docs/qa/evidence/copilot/03_copilot_error_401_invalid_key.png`

### CP-ERR-002
- **Title**: Engineer receives clear error message when upstream AI returns 502 Bad Gateway
- **Classification**: Negative | Priority: P1
- **File**: `frontend/e2e/tests/copilot/negative/upstream-errors.spec.ts`
- **Steps**:
  1. Mock Copilot chat stream with 502 Bad Gateway.
  2. Send prompt.
- **Expected Results**:
  - In-chat error alert notifies engineer that the upstream AI service is temporarily unavailable.
- **Evidence**: `docs/qa/evidence/copilot/04_copilot_error_502_timeout.png`

---

## Edge Scenarios

### CP-SESSION-001
- **Title**: Engineer can delete active chat session from sidebar history
- **Classification**: Edge | Priority: P1
- **File**: `frontend/e2e/tests/copilot/edge/session-management.spec.ts`
- **Steps**:
  1. Hover over existing session item in chat sidebar.
  2. Click delete icon.
  3. Confirm deletion in dialog.
- **Expected Results**:
  - Session item is removed from sidebar.
  - Chat view resets to clean empty state.
- **Evidence**: `docs/qa/evidence/copilot/08_copilot_session_deleted.png`

### CP-TOOL-001
- **Title**: Engineer can toggle internal tools modal and adjust enabled tool selections
- **Classification**: Edge | Priority: P2
- **File**: `frontend/e2e/tests/copilot/edge/tool-toggles.spec.ts`
- **Steps**:
  1. Click Tools icon in prompt action bar.
  2. Modal opens displaying available tools (Jira, Bitbucket, Confluence).
  3. Toggle Jira tool checkbox.
  4. Close modal.
- **Expected Results**:
  - Enabled tools count badge updates in prompt bar.
  - Active tool set persists in component state.
- **Evidence**: `docs/qa/evidence/copilot/03_copilot_tool_settings_toggled.png`

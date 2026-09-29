<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useCopilot } from "../composables/useCopilot"
import type { ReasoningEffort } from "../types"

const {
  messages,
  sessions,
  activeSessionId,
  selectedModel,
  thinkingMode,
  reasoningEffort,
  isLoading,
  sendMessage
} = useCopilot()

const inputPrompt = ref("")
const isModelMenuOpen = ref(false)
const isEffortMenuOpen = ref(false)
const messagesContainer = ref<HTMLDivElement | null>(null)
const expandedThoughts = ref<Record<string, boolean>>({})

const availableModels = [
  "GPT-6 Astra (Reasoning)",
  "GPT-6 Sol",
  "Claude Opus 5.5 (Adaptive Thinking)",
  "Claude Sonnet 5.5",
  "Claude Fable 5.1 (Deep Reasoning)",
  "Gemini 3.8 Flash (Extended Thinking)",
  "Gemini 3.1 Pro",
  "DeepSeek-V4 Pro (Thinking)",
  "DeepSeek Flash",
  "Xiaomi MiMo-V2.5 Pro"
]

const availableEfforts: { id: ReasoningEffort; label: string; desc: string }[] = [
  { id: "low", label: "Low effort", desc: "Fast thinking, minimal latency" },
  { id: "medium", label: "Medium effort", desc: "Balanced reasoning depth" },
  { id: "high", label: "High effort", desc: "Exhaustive chain-of-thought analysis" }
]

const currentSession = computed(() => {
  return sessions.value.find((s) => s.id === activeSessionId.value) || {
    title: "Copilot Workspace",
    model: selectedModel.value
  }
})

const isThoughtExpanded = (msgId: string) => {
  return expandedThoughts.value[msgId] ?? true
}

const toggleThought = (msgId: string) => {
  const current = isThoughtExpanded(msgId)
  expandedThoughts.value[msgId] = !current
}

const handleSend = async () => {
  const text = inputPrompt.value.trim()
  if (!text || isLoading.value) return
  inputPrompt.value = ""
  await sendMessage(text)
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

const selectModel = (model: string) => {
  selectedModel.value = model
  isModelMenuOpen.value = false
  if (model.includes("Thinking") || model.includes("Reasoning") || model.includes("Astra") || model.includes("Opus") || model.includes("Fable") || model.includes("Pro")) {
    thinkingMode.value = true
  }
}

const selectEffort = (effort: ReasoningEffort) => {
  reasoningEffort.value = effort
  isEffortMenuOpen.value = false
}
</script>

<template>
  <div class="chat-main">
    <header class="chat-header">
      <div class="header-left">
        <span class="chat-header-title">{{ currentSession.title }}</span>
        <span class="chat-header-meta">{{ selectedModel }}</span>
      </div>

      <div class="header-right">
        <div v-if="thinkingMode" class="thinking-badge active">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 2a6 6 0 0 0-6 6c0 2.22 1.21 4.16 3 5.2V17a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-3.8c1.79-1.04 3-2.98 3-5.2a6 6 0 0 0-6-6z"></path>
            <path d="M9 21h6"></path>
          </svg>
          <span>Thinking ({{ reasoningEffort }})</span>
        </div>
      </div>
    </header>

    <div ref="messagesContainer" class="chat-messages">
      <div class="chat-messages-inner">
        <div v-if="messages.length === 0" class="empty-chat">
          <div class="empty-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M6 8a6 6 0 0 1 12 0v6a4 4 0 0 1-4 4H10a4 4 0 0 1-4-4z"></path>
              <path d="M9 11h.01"></path>
              <path d="M15 11h.01"></path>
              <path d="M9 15c2 1 4 1 6 0"></path>
            </svg>
          </div>
          <div class="empty-title">How can I help you today?</div>
          <div class="empty-desc">Ask about Jira tickets, Bitbucket PR reviews, Confluence architecture specs, or database queries.</div>
        </div>

        <div
          v-for="msg in messages"
          :key="msg.id"
          class="message"
          :class="msg.role"
        >
          <div class="message-meta">
            {{ msg.role === "user" ? "You" : "Lunar Copilot" }}
            <span class="message-time">{{ msg.createdAt }}</span>
          </div>

          <div class="message-bubble">
            <div v-if="msg.reasoning" class="thought-section">
              <button
                class="thought-toggle"
                type="button"
                @click="toggleThought(msg.id)"
              >
                <div class="thought-label-wrap">
                  <svg class="brain-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 2a6 6 0 0 0-6 6c0 2.22 1.21 4.16 3 5.2V17a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-3.8c1.79-1.04 3-2.98 3-5.2a6 6 0 0 0-6-6z"></path>
                    <path d="M9 21h6"></path>
                  </svg>
                  <span class="thought-title">Thought process</span>
                  <span v-if="msg.thinkingDurationMs" class="thought-duration">({{ (msg.thinkingDurationMs / 1000).toFixed(1) }}s)</span>
                </div>

                <svg class="chevron-icon" :class="{ 'is-open': isThoughtExpanded(msg.id) }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="6 9 12 15 18 9"></polyline>
                </svg>
              </button>

              <div v-if="isThoughtExpanded(msg.id)" class="thought-content">
                <pre>{{ msg.reasoning }}</pre>
              </div>
            </div>

            <div class="message-text">{{ msg.content }}</div>

            <div v-if="msg.sources && msg.sources.length > 0" class="source-row">
              <span
                v-for="(source, idx) in msg.sources"
                :key="idx"
                class="source-chip"
              >
                {{ source.label }}
              </span>
            </div>
          </div>
        </div>

        <div v-if="isLoading" class="message assistant">
          <div class="message-meta">Lunar Copilot</div>
          <div class="message-bubble typing">
            <div class="typing-indicator-wrap">
              <span v-if="thinkingMode" class="thinking-text">Thinking with {{ reasoningEffort }} reasoning...</span>
              <div class="dots-wrap">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="chat-input-area">
      <form class="chat-composer" @submit.prevent="handleSend">
        <textarea
          v-model="inputPrompt"
          rows="1"
          placeholder="Message Lunar Copilot with tool integrations..."
          @keydown="handleKeydown"
        ></textarea>

        <div class="composer-actions">
          <div class="action-controls-left">
            <button
              class="control-pill thinking-toggle"
              :class="{ 'is-active': thinkingMode }"
              type="button"
              :aria-pressed="thinkingMode"
              @click="thinkingMode = !thinkingMode"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 2a6 6 0 0 0-6 6c0 2.22 1.21 4.16 3 5.2V17a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-3.8c1.79-1.04 3-2.98 3-5.2a6 6 0 0 0-6-6z"></path>
                <path d="M9 21h6"></path>
              </svg>
              <span>{{ thinkingMode ? "Thinking ON" : "Thinking OFF" }}</span>
            </button>

            <div v-if="thinkingMode" class="effort-menu-wrap">
              <button
                class="control-pill effort-trigger"
                type="button"
                @click="isEffortMenuOpen = !isEffortMenuOpen"
              >
                <span>Effort: {{ reasoningEffort }}</span>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="m6 15 6-6 6 6"></path>
                </svg>
              </button>

              <div v-if="isEffortMenuOpen" class="effort-menu">
                <button
                  v-for="effort in availableEfforts"
                  :key="effort.id"
                  class="effort-option"
                  :class="{ 'is-selected': reasoningEffort === effort.id }"
                  type="button"
                  @click="selectEffort(effort.id)"
                >
                  <div class="effort-opt-text">
                    <span class="effort-opt-label">{{ effort.label }}</span>
                    <span class="effort-opt-desc">{{ effort.desc }}</span>
                  </div>
                </button>
              </div>
            </div>
          </div>

          <div class="action-controls-right">
            <div class="model-trigger-wrap">
              <button
                class="model-trigger"
                type="button"
                @click="isModelMenuOpen = !isModelMenuOpen"
              >
                <span class="model-trigger-label">{{ selectedModel }}</span>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="m6 15 6-6 6 6"></path>
                </svg>
              </button>

              <div v-if="isModelMenuOpen" class="model-menu">
                <button
                  v-for="model in availableModels"
                  :key="model"
                  class="model-option"
                  :class="{ 'is-selected': selectedModel === model }"
                  type="button"
                  @click="selectModel(model)"
                >
                  <span>{{ model }}</span>
                  <svg v-if="selectedModel === model" class="check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="m5 13 4 4L19 7"></path>
                  </svg>
                </button>
              </div>
            </div>

            <button class="send-btn" type="submit" :disabled="!inputPrompt.trim() || isLoading" aria-label="Send">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 19V5"></path>
                <path d="m5 12 7-7 7 7"></path>
              </svg>
            </button>
          </div>
        </div>
      </form>

      <div class="chat-hint">
        <kbd>Enter</kbd> to send · <kbd>Shift</kbd> + <kbd>Enter</kbd> for new line
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-main {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 64px);
  min-width: 0;
  background: var(--bg);
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 52px;
  padding: 0 24px;
  border-bottom: 1px solid var(--border);
  background: rgba(16, 18, 23, 0.6);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chat-header-title {
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
}

.chat-header-meta {
  color: var(--subtle);
  font-size: 11px;
}

.thinking-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid rgba(159, 182, 166, 0.3);
  background: rgba(159, 182, 166, 0.1);
  color: var(--positive);
}

.thinking-badge svg {
  width: 12px;
  height: 12px;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.chat-messages-inner {
  max-width: 820px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.empty-chat {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
}

.empty-icon {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-raised);
  color: var(--muted);
  margin-bottom: 16px;
}

.empty-icon svg {
  width: 24px;
  height: 24px;
}

.empty-title {
  color: var(--text);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.empty-desc {
  max-width: 440px;
  margin-top: 8px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}

.message {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.message.user {
  align-items: flex-end;
}

.message.assistant {
  align-items: flex-start;
}

.message-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--subtle);
  font-size: 11px;
  font-weight: 500;
}

.message-time {
  font-variant-numeric: tabular-nums;
  opacity: 0.7;
}

.message-bubble {
  max-width: 85%;
  padding: 14px 18px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.6;
}

.message.user .message-bubble {
  background: var(--surface-raised);
  border: 1px solid var(--border-strong);
  color: var(--text);
  white-space: pre-wrap;
}

.message.assistant .message-bubble {
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid var(--border);
  color: #e5e9f0;
}

.thought-section {
  margin-bottom: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.25);
  overflow: hidden;
}

.thought-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border: 0;
  background: transparent;
  color: var(--subtle);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease;
}

.thought-toggle:hover {
  background: rgba(255, 255, 255, 0.03);
  color: var(--text);
}

.thought-label-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
}

.brain-icon {
  width: 13px;
  height: 13px;
  stroke: var(--positive);
}

.thought-title {
  color: var(--muted);
  font-weight: 600;
}

.thought-duration {
  color: var(--subtle);
  font-variant-numeric: tabular-nums;
}

.chevron-icon {
  width: 12px;
  height: 12px;
  transition: transform 180ms ease;
}

.chevron-icon.is-open {
  transform: rotate(180deg);
}

.thought-content {
  padding: 10px 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(0, 0, 0, 0.15);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  color: #9aa2ae;
  line-height: 1.5;
  white-space: pre-wrap;
}

.thought-content pre {
  margin: 0;
  font-family: inherit;
}

.message-text {
  white-space: pre-wrap;
}

.source-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
}

.source-chip {
  padding: 2px 8px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 5px;
  background: rgba(255, 255, 255, 0.03);
  color: var(--muted);
  font-size: 11px;
}

.typing {
  padding: 12px 16px;
}

.typing-indicator-wrap {
  display: flex;
  align-items: center;
  gap: 10px;
}

.thinking-text {
  color: var(--subtle);
  font-size: 11px;
  font-style: italic;
}

.dots-wrap {
  display: flex;
  gap: 4px;
}

.dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--muted);
  animation: pulse 1s infinite alternate;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes pulse {
  0% { opacity: 0.2; transform: scale(0.8); }
  100% { opacity: 1; transform: scale(1.1); }
}

.chat-input-area {
  padding: 16px 24px 20px;
  border-top: 1px solid var(--border);
  background: var(--bg);
}

.chat-composer {
  max-width: 820px;
  margin: 0 auto;
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  background: var(--surface-raised);
  padding: 12px 14px;
}

.chat-composer textarea {
  width: 100%;
  background: transparent;
  border: 0;
  outline: none;
  color: var(--text);
  font-size: 14px;
  line-height: 1.5;
  resize: none;
}

.composer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
}

.action-controls-left,
.action-controls-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.control-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--subtle);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 180ms ease;
}

.control-pill:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text);
}

.control-pill svg {
  width: 12px;
  height: 12px;
}

.thinking-toggle.is-active {
  border-color: rgba(159, 182, 166, 0.35);
  background: rgba(159, 182, 166, 0.1);
  color: var(--positive);
}

.effort-menu-wrap {
  position: relative;
}

.effort-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  width: 220px;
  padding: 4px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: #151820;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
  display: grid;
  gap: 2px;
  z-index: 30;
}

.effort-option {
  display: flex;
  align-items: flex-start;
  width: 100%;
  padding: 7px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background-color 160ms ease;
}

.effort-option:hover {
  background: rgba(255, 255, 255, 0.04);
}

.effort-option.is-selected {
  background: rgba(159, 182, 166, 0.1);
}

.effort-opt-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.effort-opt-label {
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
}

.effort-opt-desc {
  color: var(--subtle);
  font-size: 10px;
}

.model-trigger-wrap {
  position: relative;
}

.model-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 180ms ease;
}

.model-trigger:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text);
}

.model-trigger svg {
  width: 12px;
  height: 12px;
}

.model-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  right: 0;
  width: 240px;
  padding: 4px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: #151820;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
  display: grid;
  gap: 2px;
  z-index: 30;
}

.model-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 7px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
}

.model-option:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text);
}

.model-option.is-selected {
  color: var(--text);
  font-weight: 600;
}

.model-option .check {
  width: 12px;
  height: 12px;
  stroke: var(--positive);
}

.send-btn {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 6px;
  background: var(--accent);
  color: #0b0c0f;
  cursor: pointer;
  transition: opacity 180ms ease, transform 180ms ease;
}

.send-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.send-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.send-btn svg {
  width: 14px;
  height: 14px;
}

.chat-hint {
  max-width: 820px;
  margin: 8px auto 0;
  color: var(--subtle);
  font-size: 11px;
  text-align: right;
}

.chat-hint kbd {
  padding: 1px 4px;
  border: 1px solid var(--border);
  border-radius: 4px;
  font-size: 10px;
}
</style>

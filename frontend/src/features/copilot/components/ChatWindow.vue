<script setup lang="ts">
import { nextTick, ref } from "vue"
import { useCopilot } from "../composables/useCopilot"
import { useCodeAgent } from "../../code-context/composables/useCodeAgent"
import ChatMessage from "./ChatMessage.vue"
import ChatStreamingMessage from "./ChatStreamingMessage.vue"
import ChatComposer from "./ChatComposer.vue"
import CodeIndexStatus from "./CodeIndexStatus.vue"
import ThinkingControl from "./ThinkingControl.vue"
import EffortControl from "./EffortControl.vue"
import type { ReasoningEffort } from "../types"

const { messages, isLoading, thinkingMode, reasoningEffort, sendMessage, regenerateMessage } = useCopilot()

const {
  streamingText,
  streamingReasoning,
  streamingSources,
  streamingToolCalls,
  isStreaming,
  indexHint
} = useCodeAgent()

const messagesContainer = ref<HTMLDivElement | null>(null)

const handleSend = async (text: string): Promise<void> => {
  await sendMessage(text)
  await nextTick()
  if (messagesContainer.value !== null) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const toggleThinkingMode = (): void => {
  thinkingMode.value = !thinkingMode.value
}

const selectEffort = (effort: ReasoningEffort): void => {
  reasoningEffort.value = effort
}
</script>

<template>
  <div class="chat-main">
    <div ref="messagesContainer" class="chat-messages">
      <div class="chat-messages-inner">
        <div v-if="messages.length === 0" class="empty-chat">
          <div class="empty-icon">
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M6 8a6 6 0 0 1 12 0v6a4 4 0 0 1-4 4H10a4 4 0 0 1-4-4z"></path>
              <path d="M9 11h.01"></path>
              <path d="M15 11h.01"></path>
              <path d="M9 15c2 1 4 1 6 0"></path>
            </svg>
          </div>
          <div class="empty-title">How can I help you today?</div>
          <div class="empty-desc">Ask about Jira tickets, Bitbucket PR reviews, Confluence architecture specs, or database queries.</div>
        </div>

        <ChatMessage
          v-for="msg in messages"
          :key="msg.id"
          :message="msg"
          :disabled="isLoading"
          @regenerate="regenerateMessage(msg.id)"
        />

        <ChatStreamingMessage
          v-if="isLoading"
          :is-streaming="isStreaming"
          :text="streamingText"
          :reasoning="streamingReasoning"
          :sources="streamingSources"
          :tool-calls="streamingToolCalls"
          :thinking-mode="thinkingMode"
          :reasoning-effort="reasoningEffort"
        />
      </div>
    </div>

    <div class="chat-input-area">
      <div class="control-bar">
        <div class="control-group">
          <ThinkingControl :is-enabled="thinkingMode" @toggle="toggleThinkingMode" />
          <EffortControl :effort="reasoningEffort" :is-thinking-enabled="thinkingMode" @select="selectEffort" />
        </div>

        <CodeIndexStatus />
      </div>

      <div v-if="indexHint" class="index-hint" role="status">{{ indexHint }}</div>

      <ChatComposer :is-loading="isLoading" @send="handleSend" />
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

.chat-input-area {
  padding: 16px 24px 20px;
  border-top: 1px solid var(--border);
  background: var(--bg);
}

.control-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px 16px;
  max-width: 820px;
  margin: 0 auto 10px;
}

.control-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.index-hint {
  max-width: 820px;
  margin: 0 auto 10px;
  padding: 8px 12px;
  border: 1px solid color-mix(in srgb, var(--warning) 35%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--warning) 8%, transparent);
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
}
</style>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from "vue"
import { useCopilot } from "../composables/useCopilot"
import { useCodeAgent } from "../../code-context/composables/useCodeAgent"
import ChatMessage from "./ChatMessage.vue"
import ChatStreamingMessage from "./ChatStreamingMessage.vue"
import ChatComposer from "./ChatComposer.vue"

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
const isPinnedToBottom = ref(true)

const SCROLL_BOTTOM_THRESHOLD = 48

const updatePinnedState = (): void => {
  const container = messagesContainer.value
  if (container === null) {
    return
  }
  const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight
  isPinnedToBottom.value = distanceFromBottom <= SCROLL_BOTTOM_THRESHOLD
}

const scrollToLatest = async (): Promise<void> => {
  await nextTick()
  const container = messagesContainer.value
  if (container !== null) {
    container.scrollTop = container.scrollHeight
  }
  isPinnedToBottom.value = true
}

const handleSend = async (text: string): Promise<void> => {
  await sendMessage(text)
  await scrollToLatest()
}

onMounted(updatePinnedState)

watch([() => messages.value.length, streamingText], () => {
  if (isPinnedToBottom.value) {
    void scrollToLatest()
  }
})
</script>

<template>
  <div class="chat-main">
    <div ref="messagesContainer" class="chat-messages" @scroll="updatePinnedState">
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

    <button
      v-if="!isPinnedToBottom && messages.length > 0"
      class="scroll-latest"
      type="button"
      aria-label="Scroll to the latest message"
      data-testid="scroll-latest"
      @click="scrollToLatest"
    >
      <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 5v14"></path>
        <path d="m6 13 6 6 6-6"></path>
      </svg>
    </button>

    <div class="chat-input-area">
      <div v-if="indexHint" class="index-hint" role="status">{{ indexHint }}</div>

      <ChatComposer :is-loading="isLoading" @send="handleSend" />
    </div>
  </div>
</template>

<style scoped>
.chat-main {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
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

.scroll-latest {
  position: absolute;
  left: 50%;
  bottom: 96px;
  transform: translateX(-50%);
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--border-strong);
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--text);
  cursor: pointer;
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.22);
  transition: background-color 160ms ease, border-color 160ms ease;
}

.scroll-latest:hover {
  background: var(--surface-hover);
  border-color: var(--accent);
}

.scroll-latest:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.scroll-latest svg {
  width: 16px;
  height: 16px;
}

.chat-input-area {
  padding: 16px 24px 20px;
  border-top: 1px solid var(--border);
  background: var(--bg);
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

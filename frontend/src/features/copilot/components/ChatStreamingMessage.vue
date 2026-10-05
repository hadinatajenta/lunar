<script setup lang="ts">
import { onUnmounted, ref, watch } from "vue"
import type { AgentSource } from "../../code-context/types"
import type { ActiveToolCall } from "../../code-context/agent-types"
import ChatMessageContent from "./ChatMessageContent.vue"
import ChatReasoningPanel from "./ChatReasoningPanel.vue"
import ChatStreamingIndicator from "./ChatStreamingIndicator.vue"
import ChatToolCalls from "./ChatToolCalls.vue"

const props = defineProps<{
  isStreaming: boolean
  text: string
  reasoning: string
  sources: AgentSource[]
  toolCalls: ActiveToolCall[]
  thinkingMode: boolean
  reasoningEffort: string
}>()

const elapsedMs = ref(0)
let startedAt = 0
let timer: ReturnType<typeof setInterval> | null = null

const stopTimer = (): void => {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
}

const startTimer = (): void => {
  stopTimer()
  startedAt = Date.now()
  elapsedMs.value = 0
  timer = setInterval(() => {
    elapsedMs.value = Date.now() - startedAt
  }, 100)
}

watch(
  () => props.isStreaming,
  (streaming) => {
    if (streaming) {
      startTimer()
      return
    }
    stopTimer()
  },
  { immediate: true }
)

onUnmounted(stopTimer)
</script>

<template>
  <div class="message assistant">
    <div class="message-avatar assistant-avatar" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 3l1.9 5.1L19 10l-5.1 1.9L12 17l-1.9-5.1L5 10l5.1-1.9z"></path>
      </svg>
    </div>

    <div class="message-body">
      <template v-if="props.isStreaming">
        <ChatReasoningPanel
          v-if="props.reasoning"
          :reasoning="props.reasoning"
          :is-reasoning="props.text.length === 0"
        />

        <ChatToolCalls v-if="props.toolCalls.length > 0" :calls="props.toolCalls" :is-streaming="true" />

        <ChatMessageContent
          v-if="props.text.length > 0"
          class="message-text"
          :content="props.text"
          :is-streaming="true"
        />

        <ChatStreamingIndicator
          v-else
          :label="props.thinkingMode ? `Thinking with ${props.reasoningEffort} reasoning` : 'Preparing response'"
          :elapsed-ms="elapsedMs"
        />

      </template>

      <template v-else>
        <span v-if="props.thinkingMode" class="thinking-text">Thinking with {{ props.reasoningEffort }} reasoning...</span>
        <div class="dots-wrap" role="status" aria-live="polite">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.message {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.message-avatar {
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 2px;
}

.message-avatar svg {
  width: 16px;
  height: 16px;
}

.assistant-avatar {
  border-radius: 50%;
  background: color-mix(in srgb, var(--accent) 14%, transparent);
  color: var(--accent);
}

.message-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}

.message-text {
  white-space: normal;
  color: var(--text);
}

.thinking-text {
  color: var(--muted);
  font-size: 12px;
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

@media (prefers-reduced-motion: reduce) {
  .dot {
    animation: none;
    opacity: 0.6;
  }
}
</style>

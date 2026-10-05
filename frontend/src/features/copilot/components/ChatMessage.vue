<script setup lang="ts">
import type { ChatMessage } from "../types"
import ChatMessageContent from "./ChatMessageContent.vue"
import ChatMessageActions from "./ChatMessageActions.vue"
import ChatReasoningPanel from "./ChatReasoningPanel.vue"

const props = defineProps<{
  message: ChatMessage
  disabled?: boolean
}>()

const emit = defineEmits<{
  (event: "regenerate"): void
}>()
</script>

<template>
  <div class="message" :class="props.message.role">
    <div v-if="props.message.role === 'assistant'" class="message-avatar assistant-avatar" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 3l1.9 5.1L19 10l-5.1 1.9L12 17l-1.9-5.1L5 10l5.1-1.9z"></path>
      </svg>
    </div>

    <div class="message-body">
      <div v-if="props.message.role === 'user'" class="user-bubble">{{ props.message.content }}</div>

      <template v-else>
        <ChatReasoningPanel
          v-if="props.message.reasoning"
          :reasoning="props.message.reasoning"
          :is-reasoning="false"
          :duration-ms="props.message.thinkingDurationMs"
        />

        <ChatMessageContent class="message-text" :content="props.message.content" />


        <div v-if="props.message.sources && props.message.sources.length > 0" class="source-row">
          <span v-for="(source, idx) in props.message.sources" :key="idx" class="source-chip">
            {{ source.label }}
          </span>
        </div>

        <ChatMessageActions :text="props.message.content" :disabled="props.disabled" @regenerate="emit('regenerate')" />
      </template>

      <div v-if="props.message.role === 'user'" class="message-time">{{ props.message.createdAt }}</div>
    </div>

    <div v-if="props.message.role === 'user'" class="message-avatar user-avatar" aria-hidden="true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
        <circle cx="12" cy="7" r="4"></circle>
      </svg>
    </div>
  </div>
</template>

<style scoped>
.message {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.message.user {
  justify-content: flex-end;
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

.user-avatar {
  border-radius: 10px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  color: var(--muted);
}

.message-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.message.user .message-body {
  flex: 0 1 auto;
  align-items: flex-end;
  max-width: min(560px, 84%);
}

.user-bubble {
  padding: 11px 15px;
  border-radius: 16px 6px 16px 16px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.message-time {
  font-size: 11px;
  color: var(--subtle);
  font-variant-numeric: tabular-nums;
}

.message-text {
  white-space: normal;
  color: var(--text);
}

.source-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px solid var(--border);
}

.source-chip {
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 11px;
}
</style>

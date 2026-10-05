<script setup lang="ts">
import { computed, ref } from "vue"
import { useCopilot } from "../composables/useCopilot"
import ConfirmDialog from "../../../components/ui/ConfirmDialog.vue"

const { sessions, activeSessionId, isDomainModalOpen, startNewChat, selectSession, deleteSession } = useCopilot()

const pendingDeletionId = ref<string | null>(null)

const pendingDeletion = computed(
  () => sessions.value.find((session) => session.id === pendingDeletionId.value) ?? null
)

const requestDeletion = (sessionId: string): void => {
  pendingDeletionId.value = sessionId
}

const cancelDeletion = (): void => {
  pendingDeletionId.value = null
}

const confirmDeletion = (): void => {
  if (pendingDeletionId.value !== null) deleteSession(pendingDeletionId.value)
  pendingDeletionId.value = null
}
</script>

<template>
  <aside class="chat-history">
    <div class="chat-history-header">
      <button
        class="new-chat-btn"
        type="button"
        aria-keyshortcuts="Meta+Shift+O Control+Shift+O"
        @click="startNewChat"
      >
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 20h9"></path>
          <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"></path>
        </svg>
        <span class="new-chat-label">New chat</span>
        <kbd class="new-chat-shortcut">⌘⇧O</kbd>
      </button>

      <button
        class="history-settings-btn"
        type="button"
        title="Copilot active tools configuration"
        aria-label="Chat tool settings"
        @click="isDomainModalOpen = true"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06-.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
      </button>
    </div>

    <div class="chat-history-list">
      <div class="history-section-label">Conversations</div>

      <div v-if="sessions.length === 0" class="empty-history-notice">
        No conversation history.
      </div>

      <div
        v-for="session in sessions"
        :key="session.id"
        class="history-item"
        :class="{ 'is-active': activeSessionId === session.id }"
        @click="selectSession(session.id)"
      >
        <div class="session-text-wrap">
          <span class="session-title">{{ session.title }}</span>
          <span class="session-meta">{{ session.model }}</span>
        </div>

        <button
          class="history-delete-btn"
          type="button"
          aria-label="Delete conversation"
          title="Delete conversation"
          @click.stop="requestDeletion(session.id)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 6h18"></path>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"></path>
            <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
          </svg>
        </button>
      </div>
    </div>

    <ConfirmDialog
      :open="pendingDeletion !== null"
      tone="danger"
      title="Delete conversation?"
      :message="`\u201c${pendingDeletion?.title ?? ''}\u201d and its messages will be removed permanently. This cannot be undone.`"
      confirm-label="Delete"
      cancel-label="Keep it"
      @confirm="confirmDeletion"
      @cancel="cancelDeletion"
    />
  </aside>
</template>
<style scoped src="./chat-history.css"></style>

<script setup lang="ts">
import { useCopilot } from "../composables/useCopilot"

const { sessions, activeSessionId, isDomainModalOpen, startNewChat, selectSession, deleteSession } = useCopilot()
</script>

<template>
  <aside class="chat-history">
    <div class="chat-history-header">
      <button class="new-chat-btn" type="button" @click="startNewChat">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <span>New chat</span>
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
          @click.stop="deleteSession(session.id)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 6h18"></path>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"></path>
            <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
          </svg>
        </button>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.chat-history {
  display: flex;
  flex-direction: column;
  height: 100%;
  border-right: 1px solid var(--border);
  background: rgba(13, 15, 19, 0.95);
}

.chat-history-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  border-bottom: 1px solid var(--border);
}

.new-chat-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  color: var(--text);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 180ms ease, border-color 180ms ease;
}

.new-chat-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  border-color: var(--border-strong);
}

.new-chat-btn svg {
  width: 14px;
  height: 14px;
}

.history-settings-btn {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.03);
  color: var(--muted);
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease, border-color 180ms ease;
}

.history-settings-btn:hover {
  color: var(--text);
  background: rgba(255, 255, 255, 0.06);
  border-color: var(--border-strong);
}

.history-settings-btn svg {
  width: 15px;
  height: 15px;
}

.chat-history-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.history-section-label {
  padding: 8px 10px 4px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.empty-history-notice {
  padding: 18px 10px;
  color: var(--subtle);
  font-size: 12px;
  text-align: center;
}

.history-item {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 8px 10px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--muted);
  text-align: left;
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease;
}

.history-item:hover {
  background: rgba(255, 255, 255, 0.035);
  color: var(--text);
}

.history-item.is-active {
  background: rgba(255, 255, 255, 0.065);
  color: var(--text);
}

.session-text-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.session-title {
  width: 100%;
  font-size: 13px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-meta {
  color: var(--subtle);
  font-size: 10px;
}

.history-delete-btn {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--subtle);
  opacity: 0;
  pointer-events: none;
  cursor: pointer;
  transition: opacity 160ms ease, color 160ms ease, background-color 160ms ease;
  margin-left: 6px;
  flex-shrink: 0;
}

.history-item:hover .history-delete-btn {
  opacity: 1;
  pointer-events: auto;
}

.history-delete-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.history-delete-btn svg {
  width: 13px;
  height: 13px;
}
</style>

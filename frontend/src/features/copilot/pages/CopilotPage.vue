<script setup lang="ts">
import { onMounted } from "vue"
import PageLayout from "../../../components/layout/PageLayout.vue"
import ChatSidebar from "../components/ChatSidebar.vue"
import ChatWindow from "../components/ChatWindow.vue"
import DomainToggleModal from "../components/DomainToggleModal.vue"
import { useCopilot } from "../composables/useCopilot"
import { useCopilotShortcuts } from "../composables/useCopilotShortcuts"

const { hasConfiguredAI, isCheckingProviders, refreshConfiguredProviders } = useCopilot()

useCopilotShortcuts()

onMounted(async () => {
  await refreshConfiguredProviders()
})
</script>

<template>
  <PageLayout>
    <div class="copilot-container">
      <div v-if="!hasConfiguredAI && !isCheckingProviders" class="unconfigured-gate" data-testid="banner-no-ai-keys">
        <div class="gate-card">
          <div class="gate-icon-wrap" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            </svg>
          </div>
          <h2 class="gate-title">AI Provider API Key Required</h2>
          <p class="gate-description">
            You have not configured an API key for any AI provider yet. To use Lunar Copilot, please configure at least one API key in Settings (e.g. DeepSeek, Google Gemini, OpenAI, Anthropic Claude, or Xiaomi MiMo).
          </p>
          <div class="gate-actions">
            <RouterLink to="/settings" class="gate-btn-primary" data-testid="btn-go-to-settings">
              Configure in Settings →
            </RouterLink>
          </div>
        </div>
      </div>

      <div v-else class="copilot-grid">
        <ChatSidebar class="copilot-sidebar-col" />
        <ChatWindow class="copilot-main-col" />
      </div>
      <DomainToggleModal />
    </div>
  </PageLayout>
</template>

<style scoped>
.copilot-container {
  height: calc(100vh - 64px);
  overflow: hidden;
  position: relative;
}

.unconfigured-gate {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.gate-card {
  max-width: 520px;
  width: 100%;
  background: var(--surface, #121316);
  border: 1px solid var(--border-subtle, #272a31);
  border-radius: 12px;
  padding: 36px 32px;
  text-align: center;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.4);
}

.gate-icon-wrap {
  width: 56px;
  height: 56px;
  margin: 0 auto 20px;
  border-radius: 50%;
  background: rgba(234, 179, 8, 0.12);
  color: #eab308;
  display: flex;
  align-items: center;
  justify-content: center;
}

.gate-icon-wrap svg {
  width: 28px;
  height: 28px;
}

.gate-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary, #f3f4f6);
  margin: 0 0 12px;
}

.gate-description {
  font-size: 0.9rem;
  line-height: 1.5;
  color: var(--text-secondary, #9ca3af);
  margin: 0 0 24px;
}

.gate-actions {
  display: flex;
  justify-content: center;
}

.gate-btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  background: var(--accent);
  color: var(--accent-contrast);
  font-size: 0.9rem;
  font-weight: 500;
  border-radius: 6px;
  text-decoration: none;
  transition: background 0.15s ease;
}

.gate-btn-primary:hover {
  background: #2563eb;
}

.copilot-grid {
  display: grid;
  grid-template-columns: 240px 1fr;
  height: 100%;
}

@media (max-width: 768px) {
  .copilot-grid {
    grid-template-columns: 1fr;
  }
  .copilot-sidebar-col {
    display: none;
  }
}
</style>

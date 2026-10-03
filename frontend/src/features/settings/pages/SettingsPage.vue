<script setup lang="ts">
import { onMounted, ref } from "vue"
import PageLayout from "../../../components/layout/PageLayout.vue"
import IntegrationsSettings from "../components/IntegrationsSettings.vue"
import WorkspaceSettings from "../components/WorkspaceSettings.vue"
import AiProvidersSettings from "../components/AiProvidersSettings.vue"
import SecuritySettings from "../components/SecuritySettings.vue"
import { useSettings } from "../composables/useSettings"

const activeTab = ref<"integrations" | "workspace" | "ai" | "security">("integrations")
const { fetchSettings, isLoading } = useSettings()

onMounted(() => {
  fetchSettings()
})
</script>

<template>
  <PageLayout>
    <div class="content">
      <div class="page-header">
        <p class="eyebrow">Workspace configuration</p>
        <h1>Settings</h1>
        <p class="page-copy">
          Manage integrations and AI provider credentials used by your Lunar workspace.
        </p>
      </div>

      <div class="settings-layout">
        <nav class="settings-nav" aria-label="Settings navigation">
          <button
            class="settings-tab"
            :class="{ 'is-active': activeTab === 'integrations' }"
            type="button"
            @click="activeTab = 'integrations'"
          >
            Integrations
          </button>
          <button
            class="settings-tab"
            :class="{ 'is-active': activeTab === 'workspace' }"
            type="button"
            @click="activeTab = 'workspace'"
          >
            Workspace
          </button>
          <button
            class="settings-tab"
            :class="{ 'is-active': activeTab === 'ai' }"
            type="button"
            @click="activeTab = 'ai'"
          >
            AI providers
          </button>
          <button
            class="settings-tab"
            :class="{ 'is-active': activeTab === 'security' }"
            type="button"
            @click="activeTab = 'security'"
          >
            Security
          </button>
        </nav>

        <div class="settings-body">
          <div v-if="isLoading" class="loading-state">
            Loading settings...
          </div>
          <div v-else>
            <IntegrationsSettings v-if="activeTab === 'integrations'" />
            <WorkspaceSettings v-else-if="activeTab === 'workspace'" />
            <AiProvidersSettings v-else-if="activeTab === 'ai'" />
            <SecuritySettings v-else-if="activeTab === 'security'" />
          </div>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 34px 30px 48px;
}

.page-header {
  margin-bottom: 30px;
}

.eyebrow {
  margin: 0 0 9px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: clamp(28px, 3vw, 36px);
  font-weight: 600;
  line-height: 1.1;
  letter-spacing: -0.04em;
}

.page-copy {
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.settings-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 32px;
  align-items: start;
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.settings-tab {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 38px;
  padding: 0 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--muted);
  font-size: 13px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease;
}

.settings-tab:hover {
  background: var(--surface-hover);
  color: var(--text);
  color: var(--text);
}

.settings-tab.is-active {
  background: var(--surface-raised);
  color: var(--text);
  font-weight: 600;
  color: var(--text);
  font-weight: 600;
}

.settings-body {
  min-width: 0;
}

.loading-state {
  padding: 40px;
  text-align: center;
  color: var(--subtle);
  font-size: 13px;
}

@media (max-width: 768px) {
  .settings-layout {
    grid-template-columns: 1fr;
    gap: 20px;
  }
  .settings-nav {
    flex-direction: row;
    border-bottom: 1px solid var(--border);
    padding-bottom: 8px;
  }
}
</style>

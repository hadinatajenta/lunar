<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"

const route = useRoute()

const pageTitle = computed(() => {
  if (route.path.startsWith("/jira")) return "Jira BRI"
  if (route.path.startsWith("/bitbucket")) return "Bitbucket BRI"
  if (route.path.startsWith("/confluence")) return "Confluence BRI"
  if (route.path.startsWith("/copilot")) return "AI Copilot"
  if (route.path.startsWith("/settings")) return "Settings"
  return "Dashboard"
})
</script>

<template>
  <header class="topbar">
    <div class="breadcrumb">
      <span>Workspace</span>
      <span class="sep">/</span>
      <strong>{{ pageTitle }}</strong>
    </div>

    <div class="command">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <span>Search issues, PRs, docs...</span>
      <kbd class="shortcut">⌘K</kbd>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 64px;
  padding: 0 30px;
  border-bottom: 1px solid var(--border);
  background: rgba(11, 12, 15, 0.87);
  backdrop-filter: blur(18px);
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--subtle);
  font-size: 12px;
}

.breadcrumb .sep {
  color: var(--subtle);
  opacity: 0.5;
}

.breadcrumb strong {
  color: var(--text);
  font-weight: 500;
}

.command {
  display: flex;
  align-items: center;
  gap: 10px;
  width: min(360px, 36vw);
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.022);
  color: var(--subtle);
}

.command svg {
  width: 15px;
  height: 15px;
  stroke: currentColor;
}

.command span {
  flex: 1;
  color: var(--subtle);
  font-size: 12px;
}

.shortcut {
  padding: 3px 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  color: #747c87;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  background: rgba(255, 255, 255, 0.02);
}
</style>

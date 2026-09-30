<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import { useTheme } from "../../composables/useTheme"

const route = useRoute()
const { isDark, toggleTheme } = useTheme()

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

    <div class="header-actions">
      <div class="command">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <span>Search issues, PRs, docs...</span>
        <kbd class="shortcut">⌘K</kbd>
      </div>

      <button
        class="theme-toggle"
        type="button"
        data-testid="theme-toggle"
        :aria-label="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
        @click="toggleTheme"
      >
        <svg v-if="isDark" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="4"></circle>
          <path d="M12 2v2"></path>
          <path d="M12 20v2"></path>
          <path d="m4.93 4.93 1.41 1.41"></path>
          <path d="m17.66 17.66 1.41 1.41"></path>
          <path d="M2 12h2"></path>
          <path d="M20 12h2"></path>
          <path d="m6.34 17.66-1.41 1.41"></path>
          <path d="m19.07 4.93-1.41 1.41"></path>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"></path>
        </svg>
      </button>
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
  background: color-mix(in srgb, var(--bg) 87%, transparent);
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

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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
  background: var(--surface-raised);
  color: var(--muted);
}

.command svg {
  width: 15px;
  height: 15px;
  stroke: currentColor;
}

.command span {
  flex: 1;
  color: var(--muted);
  font-size: 12px;
}

.shortcut {
  padding: 3px 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--muted);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  background: var(--surface);
}

.theme-toggle {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    border-color 150ms ease;
}

.theme-toggle:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
  color: var(--text);
}

.theme-toggle svg {
  width: 16px;
  height: 16px;
}
</style>

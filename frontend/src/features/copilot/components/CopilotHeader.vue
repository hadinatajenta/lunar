<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue"
import { useCopilot } from "../composables/useCopilot"
import { useMainSidebar } from "../../../components/layout/useMainSidebar"

const { startNewChat } = useCopilot()
const { isMobileMenuOpen, openMobileMenu, registerMenuTrigger } = useMainSidebar()

const narrowViewportQuery = window.matchMedia("(max-width: 768px)")

const isNarrowViewport = ref(narrowViewportQuery.matches)
const menuTriggerElement = ref<HTMLButtonElement | null>(null)

const handleViewportChange = (event: MediaQueryListEvent): void => {
  isNarrowViewport.value = event.matches
}

watch(menuTriggerElement, (element) => registerMenuTrigger(element), { immediate: true })

narrowViewportQuery.addEventListener("change", handleViewportChange)

onBeforeUnmount(() => {
  narrowViewportQuery.removeEventListener("change", handleViewportChange)
  registerMenuTrigger(null)
})
</script>

<template>
  <header class="copilot-header" data-testid="copilot-header">
    <div class="header-start">
      <button
        v-if="isNarrowViewport"
        ref="menuTriggerElement"
        class="menu-trigger"
        type="button"
        aria-label="Open main menu"
        aria-controls="main-menu"
        :aria-expanded="isMobileMenuOpen"
        @click="openMobileMenu"
      >
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 6h16"></path>
          <path d="M4 12h16"></path>
          <path d="M4 18h16"></path>
        </svg>
      </button>

      <div class="breadcrumb">
        <span>Workspace</span>
        <span class="sep">/</span>
        <strong>AI Copilot</strong>
      </div>
    </div>

    <button
      class="new-chat-btn"
      type="button"
      data-testid="copilot-header-new-chat"
      aria-label="New chat"
      @click="startNewChat"
    >
      <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 4v16m8-8H4"></path>
      </svg>
    </button>
  </header>
</template>

<style scoped>
.copilot-header {
  position: sticky;
  top: 0;
  z-index: 10;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 0 24px;
  border-bottom: 1px solid var(--border);
  background: transparent;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
}

.header-start {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.menu-trigger {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 150ms ease, background-color 150ms ease, border-color 150ms ease;
}

.menu-trigger:hover {
  color: var(--text);
  background: var(--surface-hover);
  border-color: var(--border-strong);
}

.menu-trigger:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.menu-trigger svg {
  width: 18px;
  height: 18px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
  color: var(--subtle);
  font-size: 12px;
  white-space: nowrap;
}

.breadcrumb .sep {
  color: var(--subtle);
  opacity: 0.5;
}

.breadcrumb strong {
  color: var(--muted);
  font-weight: 500;
}

.new-chat-btn {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 150ms ease, background-color 150ms ease, border-color 150ms ease;
}

.new-chat-btn:hover {
  color: var(--text);
  background: var(--surface-hover);
  border-color: var(--border-strong);
}

.new-chat-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.new-chat-btn svg {
  width: 16px;
  height: 16px;
}

@media (max-width: 768px) {
  .copilot-header {
    padding: 0 16px;
  }

  .breadcrumb {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .menu-trigger,
  .new-chat-btn {
    transition: none;
  }
}
</style>

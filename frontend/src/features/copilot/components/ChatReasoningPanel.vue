<script setup lang="ts">
import { computed, ref, useId } from "vue"

const props = defineProps<{
  reasoning: string
  isReasoning: boolean
  durationMs?: number
}>()

const contentId = useId()
const isExpanded = ref(false)

const hasReasoning = computed(() => props.reasoning.length > 0)

const durationLabel = computed(() => {
  const durationMs = props.durationMs
  if (durationMs === undefined || durationMs <= 0) {
    return null
  }
  return `(${(durationMs / 1000).toFixed(1)}s)`
})

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
}
</script>

<template>
  <div v-if="hasReasoning" class="reasoning-panel">
    <button
      class="reasoning-toggle"
      type="button"
      :aria-expanded="isExpanded"
      :aria-controls="contentId"
      @click="toggleExpanded"
    >
      <svg
        class="reasoning-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M9.5 18h5"></path>
        <path d="M10 21h4"></path>
        <path d="M12 3a6 6 0 0 0-3.6 10.8c.7.5 1.1 1.2 1.4 2l.1.4h4.2l.1-.4c.3-.8.7-1.5 1.4-2A6 6 0 0 0 12 3z"></path>
      </svg>

      <span class="reasoning-title">Thinking Process</span>

      <template v-if="isReasoning">
        <span class="reasoning-pulse" aria-hidden="true"></span>
        <span class="reasoning-status">thinking...</span>
      </template>
      <span v-else-if="durationLabel" class="reasoning-duration">{{ durationLabel }}</span>

      <svg
        class="reasoning-chevron"
        :class="{ 'is-open': isExpanded }"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <polyline points="6 9 12 15 18 9"></polyline>
      </svg>
    </button>

    <div v-if="isExpanded" class="reasoning-body">
      <pre :id="contentId" class="reasoning-text" tabindex="0">{{ reasoning }}<span
        v-if="isReasoning"
        class="reasoning-caret"
        aria-hidden="true"
      ></span></pre>
    </div>
  </div>
</template>

<style scoped>
.reasoning-panel {
  max-width: 42rem;
  overflow: hidden;
  border: 1px dashed var(--border-strong);
  border-radius: var(--radius);
  background: color-mix(in srgb, var(--subtle) 12%, transparent);
  color: var(--muted);
  font-size: 12px;
}

.reasoning-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 9px 13px;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color 180ms ease;
}

.reasoning-toggle:hover {
  background: var(--surface-hover);
}

.reasoning-toggle:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.reasoning-icon {
  flex-shrink: 0;
  width: 15px;
  height: 15px;
  color: var(--warning);
}

.reasoning-title {
  font-weight: 500;
}

.reasoning-pulse {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--warning);
  animation: reasoning-pulse 1s ease-in-out infinite alternate;
}

.reasoning-status {
  color: var(--warning);
  font-size: 11px;
}

.reasoning-duration {
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.reasoning-chevron {
  flex-shrink: 0;
  margin-left: auto;
  width: 12px;
  height: 12px;
  color: var(--muted);
  transition: transform 180ms ease;
}

.reasoning-chevron.is-open {
  transform: rotate(180deg);
}

.reasoning-body {
  border-top: 1px dashed var(--border-strong);
}

.reasoning-text {
  max-height: 16rem;
  margin: 0;
  padding: 10px 13px 12px;
  overflow-y: auto;
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.reasoning-text:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.reasoning-caret {
  display: inline-block;
  width: 6px;
  height: 11px;
  margin-left: 2px;
  border-radius: 2px;
  background: var(--warning);
  vertical-align: text-bottom;
  animation: reasoning-caret-blink 1s steps(2, start) infinite;
}

@keyframes reasoning-pulse {
  from {
    opacity: 0.35;
    transform: scale(0.85);
  }

  to {
    opacity: 1;
    transform: scale(1.1);
  }
}

@keyframes reasoning-caret-blink {
  50% {
    opacity: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .reasoning-pulse,
  .reasoning-caret {
    animation: none;
  }

  .reasoning-toggle,
  .reasoning-chevron {
    transition: none;
  }
}
</style>

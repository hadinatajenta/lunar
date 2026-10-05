<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
  label: string
  elapsedMs: number
}>()

const elapsedLabel = computed(() => `${(props.elapsedMs / 1000).toFixed(1)}s`)
</script>

<template>
  <div class="streaming-pill" role="status" aria-live="polite">
    <span class="streaming-spinner" aria-hidden="true"></span>
    <span class="streaming-label">{{ label }}</span>
    <span class="streaming-separator" aria-hidden="true">·</span>
    <span class="streaming-elapsed">{{ elapsedLabel }}</span>
  </div>
</template>

<style scoped>
.streaming-pill {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  align-self: flex-start;
  padding: 6px 13px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface);
  color: var(--muted);
  font-size: 12px;
  line-height: 1.4;
}

.streaming-spinner {
  flex-shrink: 0;
  width: 13px;
  height: 13px;
  border: 2px solid var(--border-strong);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: streaming-spin 900ms linear infinite;
}

.streaming-label {
  min-width: 0;
  letter-spacing: -0.005em;
}

.streaming-separator {
  flex-shrink: 0;
  color: var(--border-strong);
}

.streaming-elapsed {
  flex-shrink: 0;
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

@keyframes streaming-spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .streaming-spinner {
    animation: none;
  }
}
</style>

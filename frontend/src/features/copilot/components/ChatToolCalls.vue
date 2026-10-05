<script setup lang="ts">
import { computed, ref, useId, watch } from "vue"
import type { ActiveToolCall } from "../../code-context/agent-types"

const props = defineProps<{
  calls: ActiveToolCall[]
  isStreaming: boolean
}>()

const contentId = useId()
const isExpanded = ref(false)

watch(
  () => props.isStreaming,
  (isActive) => {
    isExpanded.value = isActive
  },
  { immediate: true }
)

const hasCalls = computed(() => props.calls.length > 0)

const callCountLabel = computed(() => {
  return props.calls.length === 1 ? "1 tool call" : `${props.calls.length} tool calls`
})

const roundCountLabel = computed(() => {
  const roundCount = new Set(props.calls.map((call) => call.round)).size
  return roundCount === 1 ? "1 round" : `${roundCount} rounds`
})

const summaryLabel = computed(() => {
  const total = props.calls.length
  if (total === 0) {
    return ""
  }
  if (!props.isStreaming) {
    return `${callCountLabel.value} completed`
  }
  const latestCall = props.calls[total - 1]
  if (latestCall.action) {
    return `Running: ${latestCall.action}...`
  }
  return `Running ${callCountLabel.value}...`
})

const resolveCallKey = (call: ActiveToolCall, index: number): string => {
  return `${call.round}-${call.tool}-${call.action}-${index}`
}

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
}
</script>

<template>
  <div v-if="hasCalls" class="tool-calls">
    <button
      class="tool-header"
      type="button"
      :aria-expanded="isExpanded"
      :aria-controls="contentId"
      @click="toggleExpanded"
    >
      <span class="tool-leading">
        <span v-if="isStreaming" class="tool-spinner" aria-hidden="true"></span>
        <svg
          v-else
          class="tool-check"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="9"></circle>
          <polyline points="8.4 12.4 11 15 15.6 9.4"></polyline>
        </svg>
      </span>

      <span class="tool-summary">{{ summaryLabel }}</span>
      <span class="tool-rounds">{{ roundCountLabel }}</span>

      <span class="tool-toggle-hint">{{ isExpanded ? "Hide details" : "View details" }}</span>
      <svg
        class="tool-chevron"
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

    <ul v-if="isExpanded" :id="contentId" class="tool-rows">
      <li
        v-for="(call, index) in calls"
        :key="resolveCallKey(call, index)"
        class="tool-row"
        :class="{ 'is-active': isStreaming }"
      >
        <span v-if="isStreaming" class="tool-row-pulse" aria-hidden="true"></span>
        <span v-else class="tool-row-dot" aria-hidden="true"></span>

        <span class="tool-row-round">#{{ call.round }}</span>
        <span class="tool-row-tool">{{ call.tool }}</span>
        <span class="tool-row-action">{{ call.action }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.tool-calls {
  max-width: 42rem;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  font-size: 12px;
}

.tool-header {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 9px 13px;
  border: 0;
  background: transparent;
  color: var(--muted);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color 180ms ease;
}

.tool-header:hover {
  background: var(--surface-hover);
}

.tool-header:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.tool-leading {
  display: inline-flex;
  flex-shrink: 0;
}

.tool-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border-strong);
  border-top-color: var(--warning);
  border-radius: 50%;
  animation: tool-spin 900ms linear infinite;
}

.tool-check {
  width: 15px;
  height: 15px;
  color: var(--positive);
}

.tool-summary {
  min-width: 0;
  overflow: hidden;
  color: var(--muted);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-rounds {
  flex-shrink: 0;
  color: var(--subtle);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.tool-toggle-hint {
  flex-shrink: 0;
  margin-left: auto;
  color: var(--muted);
  font-size: 11px;
}

.tool-chevron {
  flex-shrink: 0;
  width: 12px;
  height: 12px;
  color: var(--muted);
  transition: transform 180ms ease;
}

.tool-chevron.is-open {
  transform: rotate(180deg);
}

.tool-rows {
  display: grid;
  gap: 6px;
  margin: 0;
  padding: 7px 10px 10px;
  border-top: 1px solid var(--border);
  list-style: none;
}

.tool-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 9px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: color-mix(in srgb, var(--subtle) 12%, transparent);
}

.tool-row.is-active {
  border-color: color-mix(in srgb, var(--warning) 32%, transparent);
}

.tool-row-pulse {
  flex-shrink: 0;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--warning);
  animation: tool-pulse 1s ease-in-out infinite alternate;
}

.tool-row-dot {
  flex-shrink: 0;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--positive);
}

.tool-row-round {
  flex-shrink: 0;
  color: var(--subtle);
  font-variant-numeric: tabular-nums;
}

.tool-row-tool {
  flex-shrink: 0;
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.tool-row-action {
  min-width: 0;
  overflow: hidden;
  color: var(--muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@keyframes tool-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@keyframes tool-pulse {
  from { opacity: 0.35; transform: scale(0.85); }
  to { opacity: 1; transform: scale(1.1); }
}

@media (prefers-reduced-motion: reduce) {
  .tool-spinner,
  .tool-row-pulse { animation: none; }
  .tool-header,
  .tool-chevron { transition: none; }
}
</style>

<script setup lang="ts">
import type { ReasoningEffort } from "../types"

const props = defineProps<{
  effort: ReasoningEffort
  isThinkingEnabled: boolean
}>()

const emit = defineEmits<{
  (event: "select", effort: ReasoningEffort): void
}>()

const EFFORT_OPTIONS: { id: ReasoningEffort; label: string; description: string }[] = [
  { id: "low", label: "Low", description: "Fast thinking, minimal latency" },
  { id: "medium", label: "Medium", description: "Balanced reasoning depth" },
  { id: "high", label: "High", description: "Exhaustive chain-of-thought analysis" }
]

const selectEffort = (effort: ReasoningEffort): void => {
  if (!props.isThinkingEnabled) return
  emit("select", effort)
}
</script>

<template>
  <div class="effort-control">
    <span id="reasoning-effort-label" class="effort-label">Reasoning effort</span>

    <div
      class="effort-options"
      :class="{ 'is-inactive': !props.isThinkingEnabled }"
      role="group"
      aria-labelledby="reasoning-effort-label"
      :aria-disabled="!props.isThinkingEnabled"
    >
      <button
        v-for="option in EFFORT_OPTIONS"
        :key="option.id"
        class="effort-option"
        :class="{ 'is-selected': props.effort === option.id }"
        type="button"
        :data-testid="`effort-option-${option.id}`"
        :aria-pressed="props.effort === option.id"
        :aria-disabled="!props.isThinkingEnabled"
        :title="option.description"
        @click="selectEffort(option.id)"
      >
        {{ option.label }}
      </button>
    </div>

    <p class="effort-requirement">Applies only while thinking is on</p>
  </div>
</template>

<style scoped>
.effort-control {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.effort-label {
  color: var(--text);
  font-size: 11px;
  font-weight: 600;
  white-space: nowrap;
}

.effort-options {
  display: flex;
  align-items: center;
  gap: 4px;
  transition: opacity 180ms ease;
}

.effort-options.is-inactive {
  opacity: 0.5;
}

.effort-options.is-inactive .effort-option {
  cursor: not-allowed;
}

.effort-option {
  height: 26px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease, border-color 180ms ease;
}

.effort-options:not(.is-inactive) .effort-option:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
  color: var(--text);
}

.effort-option:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.effort-option.is-selected {
  border-color: var(--border-strong);
  background: var(--surface-hover);
  color: var(--text);
  font-weight: 600;
}

.effort-requirement {
  margin: 0;
  color: var(--muted);
  font-size: 10px;
  line-height: 1.4;
}

@media (prefers-reduced-motion: reduce) {
  .effort-options {
    transition: none;
  }
}
</style>

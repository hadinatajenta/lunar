<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    message: string
    tone?: "neutral" | "warning" | "danger"
    actionLabel?: string
    actionTo?: string
    secondaryActionLabel?: string
    secondaryActionTo?: string
  }>(),
  {
    tone: "neutral",
    actionLabel: "",
    actionTo: "",
    secondaryActionLabel: "",
    secondaryActionTo: ""
  }
)

const emit = defineEmits<{
  (event: "action"): void
  (event: "secondary-action"): void
}>()
</script>

<template>
  <div class="state-panel" :class="`is-${tone}`" data-testid="workspace-state-panel">
    <span class="state-icon" aria-hidden="true">
      <svg v-if="tone === 'danger'" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="9"></circle>
        <path d="M12 8v5"></path>
        <path d="M12 16h.01"></path>
      </svg>
      <svg v-else-if="tone === 'warning'" viewBox="0 0 24 24">
        <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"></path>
        <path d="M12 9v4"></path>
        <path d="M12 17h.01"></path>
      </svg>
      <svg v-else viewBox="0 0 24 24">
        <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
      </svg>
    </span>

    <h2 class="state-title">{{ title }}</h2>
    <p class="state-message">{{ message }}</p>

    <div v-if="actionLabel || secondaryActionLabel" class="state-actions">
      <RouterLink v-if="actionLabel && actionTo" class="state-action" :to="actionTo">
        {{ actionLabel }}
      </RouterLink>
      <button v-else-if="actionLabel" class="state-action" type="button" @click="emit('action')">
        {{ actionLabel }}
      </button>

      <RouterLink
        v-if="secondaryActionLabel && secondaryActionTo"
        class="state-action is-secondary"
        :to="secondaryActionTo"
      >
        {{ secondaryActionLabel }}
      </RouterLink>
      <button
        v-else-if="secondaryActionLabel"
        class="state-action is-secondary"
        type="button"
        @click="emit('secondary-action')"
      >
        {{ secondaryActionLabel }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.state-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 54px 28px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  text-align: center;
}

.state-icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  margin-bottom: 8px;
  border: 1px solid var(--border-strong);
  border-radius: 11px;
  background: var(--surface-raised);
  color: var(--muted);
}

.state-panel.is-warning .state-icon {
  border-color: color-mix(in srgb, var(--warning) 32%, transparent);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
  color: var(--warning);
}

.state-panel.is-danger .state-icon {
  border-color: color-mix(in srgb, var(--danger) 32%, transparent);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
  color: var(--danger);
}

.state-icon svg {
  width: 18px;
  height: 18px;
  stroke: currentColor;
  stroke-width: 1.7;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.state-title {
  margin: 0;
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.state-message {
  max-width: 560px;
  margin: 0;
  color: var(--muted);
  font-size: 12.5px;
  line-height: 1.65;
  overflow-wrap: anywhere;
}

.state-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
  flex-wrap: wrap;
  justify-content: center;
}

.state-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 34px;
  padding: 0 14px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--accent);
  color: var(--accent-contrast);
  font-size: 11.5px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 160ms ease, border-color 160ms ease, background-color 160ms ease;
}

.state-action:hover {
  opacity: 0.9;
}

.state-action.is-secondary {
  border-color: var(--border);
  background: transparent;
  color: var(--text);
}

.state-action.is-secondary:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
  opacity: 1;
}
</style>

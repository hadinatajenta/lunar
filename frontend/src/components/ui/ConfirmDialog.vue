<script setup lang="ts">
import { nextTick, ref, watch } from "vue"

const props = withDefaults(
  defineProps<{
    open: boolean
    title: string
    message: string
    confirmLabel?: string
    cancelLabel?: string
    tone?: "danger" | "default"
  }>(),
  {
    confirmLabel: "Confirm",
    cancelLabel: "Cancel",
    tone: "default"
  }
)

const emit = defineEmits<{
  (event: "confirm"): void
  (event: "cancel"): void
}>()

const cancelButton = ref<HTMLButtonElement | null>(null)

watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return
    await nextTick()
    cancelButton.value?.focus()
  }
)
</script>

<template>
  <div
    v-if="props.open"
    class="confirm-backdrop"
    @click.self="emit('cancel')"
    @keydown.esc="emit('cancel')"
  >
    <div
      class="confirm-card"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="confirm-title"
      aria-describedby="confirm-message"
    >
      <div class="confirm-icon" :class="{ 'is-danger': props.tone === 'danger' }" aria-hidden="true">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
          <line x1="12" y1="9" x2="12" y2="13"></line>
          <line x1="12" y1="17" x2="12.01" y2="17"></line>
        </svg>
      </div>

      <h2 id="confirm-title" class="confirm-title">{{ props.title }}</h2>
      <p id="confirm-message" class="confirm-message">{{ props.message }}</p>

      <div class="confirm-actions">
        <button ref="cancelButton" class="confirm-button is-secondary" type="button" @click="emit('cancel')">
          {{ props.cancelLabel }}
        </button>
        <button
          class="confirm-button"
          :class="props.tone === 'danger' ? 'is-danger' : 'is-primary'"
          type="button"
          data-testid="confirm-accept"
          @click="emit('confirm')"
        >
          {{ props.confirmLabel }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.confirm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.72);
  backdrop-filter: blur(8px);
}

.confirm-card {
  width: 100%;
  max-width: 400px;
  padding: 22px 24px 20px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.6);
  text-align: center;
}

.confirm-icon {
  width: 38px;
  height: 38px;
  margin: 0 auto 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--surface-raised);
  color: var(--muted);
}

.confirm-icon.is-danger {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.confirm-icon svg {
  width: 19px;
  height: 19px;
}

.confirm-title {
  margin: 0 0 8px;
  font-size: 15px;
  font-weight: 650;
  color: var(--text);
}

.confirm-message {
  margin: 0 0 20px;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--muted);
}

.confirm-actions {
  display: flex;
  gap: 9px;
}

.confirm-button {
  flex: 1;
  padding: 9px 14px;
  border-radius: 9px;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.confirm-button.is-secondary {
  border: 1px solid var(--border);
  background: var(--surface-raised);
  color: var(--text);
}

.confirm-button.is-secondary:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
}

.confirm-button.is-primary {
  border: 1px solid transparent;
  background: var(--accent);
  color: var(--accent-contrast);
}

.confirm-button.is-danger {
  border: 1px solid color-mix(in srgb, var(--danger) 40%, transparent);
  background: color-mix(in srgb, var(--danger) 16%, transparent);
  color: var(--danger);
}

.confirm-button.is-danger:hover {
  background: color-mix(in srgb, var(--danger) 24%, transparent);
}

.confirm-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
</style>

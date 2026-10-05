<script setup lang="ts">
const props = defineProps<{
  isEnabled: boolean
}>()

const emit = defineEmits<{
  (event: "toggle"): void
}>()
</script>

<template>
  <button
    class="thinking-toggle"
    :class="{ 'is-active': props.isEnabled }"
    type="button"
    data-testid="thinking-toggle"
    :aria-pressed="props.isEnabled"
    aria-label="Thinking mode"
    title="Thinking mode"
    @click="emit('toggle')"
  >
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
      <path d="M12 2a6 6 0 0 0-6 6c0 2.22 1.21 4.16 3 5.2V17a1 1 0 0 0 1 1h4a1 1 0 0 0 1-1v-3.8c1.79-1.04 3-2.98 3-5.2a6 6 0 0 0-6-6z"></path>
      <path d="M9 21h6"></path>
    </svg>
    <span v-if="props.isEnabled" class="thinking-marker" aria-hidden="true"></span>
  </button>
</template>

<style scoped>
.thinking-toggle {
  --thinking-gold: color-mix(in srgb, var(--warning) 65%, var(--text));
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  color: var(--muted);
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease, border-color 180ms ease, transform 180ms ease;
}

.thinking-toggle svg {
  width: 16px;
  height: 16px;
}

.thinking-toggle:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
  color: var(--text);
}

.thinking-toggle:active {
  transform: scale(0.96);
}

.thinking-toggle:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.thinking-toggle.is-active {
  border-color: color-mix(in srgb, var(--thinking-gold) 45%, transparent);
  background: color-mix(in srgb, var(--thinking-gold) 12%, var(--surface-raised));
  color: var(--thinking-gold);
}

.thinking-toggle.is-active:hover {
  background: color-mix(in srgb, var(--thinking-gold) 20%, var(--surface-raised));
}

.thinking-marker {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

@media (prefers-reduced-motion: reduce) {
  .thinking-toggle {
    transition: none;
  }

  .thinking-toggle:active {
    transform: none;
  }
}
</style>

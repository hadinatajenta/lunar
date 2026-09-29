<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading"
    :aria-label="ariaLabel"
    :class="['btn-base', variantClass]"
  >
    <svg
      v-if="loading"
      class="btn-spinner"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
    <span :class="{ 'opacity-80': loading }">
      <slot />
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from "vue"

interface ButtonProps {
  variant?: "primary" | "secondary"
  type?: "button" | "submit" | "reset"
  disabled?: boolean
  loading?: boolean
  ariaLabel?: string
}

const props = withDefaults(defineProps<ButtonProps>(), {
  variant: "primary",
  type: "button",
  disabled: false,
  loading: false,
  ariaLabel: undefined
})

const variantClass = computed(() => {
  return props.variant === "secondary" ? "btn-secondary" : "btn-primary"
})
</script>

<style scoped>
.btn-base {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  font-family: inherit;
  outline: none;
  transition:
    transform 180ms ease,
    opacity 180ms ease,
    border-color 180ms ease,
    background-color 180ms ease;
}

.btn-base:focus-visible {
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.15);
}

.btn-primary {
  height: 48px;
  border: 0;
  border-radius: 10px;
  background: var(--accent);
  color: #0b0c0f;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
}

.btn-primary:active:not(:disabled) {
  transform: translateY(0);
}

.btn-secondary {
  min-height: 46px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: transparent;
  color: #d9dee6;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}

.btn-secondary:hover:not(:disabled) {
  border-color: var(--border-strong);
  background: rgba(255, 255, 255, 0.025);
}

.btn-base:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-spinner {
  width: 18px;
  height: 18px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>

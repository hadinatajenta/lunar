<script setup lang="ts">
import { onUnmounted, ref } from "vue"

const props = defineProps<{
  text: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  (event: "regenerate"): void
}>()

const COPY_FEEDBACK_MS = 2000

const isCopied = ref(false)
const copyError = ref("")
let copiedResetTimer: ReturnType<typeof setTimeout> | null = null

const clearCopiedResetTimer = () => {
  if (copiedResetTimer === null) return
  clearTimeout(copiedResetTimer)
  copiedResetTimer = null
}

const resolveCopyFailureMessage = (error: unknown): string => {
  if (error instanceof Error && error.message) {
    return `Copy failed: ${error.message}`
  }
  return "Copy failed: the browser rejected clipboard access."
}

const copyResponse = async () => {
  copyError.value = ""
  if (!navigator.clipboard?.writeText) {
    isCopied.value = false
    copyError.value = "Copy failed: the clipboard is unavailable in this browser."
    return
  }
  try {
    await navigator.clipboard.writeText(props.text)
  } catch (error) {
    isCopied.value = false
    copyError.value = resolveCopyFailureMessage(error)
    return
  }
  clearCopiedResetTimer()
  isCopied.value = true
  copiedResetTimer = setTimeout(() => {
    isCopied.value = false
    copiedResetTimer = null
  }, COPY_FEEDBACK_MS)
}

onUnmounted(clearCopiedResetTimer)
</script>

<template>
  <div class="message-actions" data-testid="message-actions">
    <button
      class="action-btn"
      type="button"
      title="Copy response"
      aria-label="Copy response"
      data-testid="copy-btn"
      @click="copyResponse"
    >
      <svg
        v-if="isCopied"
        class="action-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
        data-testid="copied-icon"
      >
        <path d="M8 5.5H7a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-11a2 2 0 0 0-2-2h-1"></path>
        <path d="M8 3.5h8v4H8z"></path>
        <path d="m9.5 14 2 2 4-4"></path>
      </svg>

      <svg
        v-else
        class="action-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
        data-testid="copy-icon"
      >
        <path d="M8 5.5H7a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-11a2 2 0 0 0-2-2h-1"></path>
        <path d="M8 3.5h8v4H8z"></path>
      </svg>

      <span class="action-label">{{ isCopied ? "Copied" : "Copy" }}</span>
    </button>

    <button
      class="action-btn"
      type="button"
      title="Regenerate response"
      aria-label="Regenerate response"
      data-testid="regenerate-btn"
      :disabled="disabled"
      @click="emit('regenerate')"
    >
      <svg
        class="action-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M20 12a8 8 0 1 1-2.34-5.66"></path>
        <path d="M20 4v4h-4"></path>
      </svg>
      <span class="action-label">Regenerate</span>
    </button>

    <span v-if="copyError" class="action-error" role="alert" data-testid="copy-error">
      {{ copyError }}
    </span>
  </div>
</template>

<style scoped>
.message-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  padding-top: 4px;
  color: var(--muted);
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-family: inherit;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    color 160ms ease,
    background-color 160ms ease,
    border-color 160ms ease;
}

.action-btn:hover:not(:disabled) {
  border-color: var(--border);
  background: var(--surface-hover);
  color: var(--text);
}

.action-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}

.action-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.action-icon {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
}

.action-error {
  color: var(--danger);
  font-size: 11px;
}

@media (prefers-reduced-motion: reduce) {
  .action-btn {
    transition: none;
  }
}
</style>

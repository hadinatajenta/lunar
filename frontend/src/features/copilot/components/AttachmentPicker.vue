<script setup lang="ts">
import { ATTACH_CONTROL_LABEL, ATTACHMENT_ACCEPT_ATTRIBUTE } from "../utils/attachment-rules"

const emit = defineEmits<{
  (event: "select", files: File[]): void
}>()

const inputId = "copilot-attachment-input"

const handleChange = (event: Event): void => {
  const target = event.target as HTMLInputElement
  const selectedFiles = target.files === null ? [] : Array.from(target.files)
  target.value = ""
  if (selectedFiles.length === 0) return
  emit("select", selectedFiles)
}
</script>

<template>
  <span class="attachment-picker">
    <input
      :id="inputId"
      class="attachment-input"
      type="file"
      multiple
      :accept="ATTACHMENT_ACCEPT_ATTRIBUTE"
      :aria-label="ATTACH_CONTROL_LABEL"
      :title="ATTACH_CONTROL_LABEL"
      data-testid="attachment-input"
      @change="handleChange"
    />

    <label
      class="attachment-trigger"
      :for="inputId"
      :title="ATTACH_CONTROL_LABEL"
      data-testid="attachment-trigger"
    >
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57a4 4 0 1 1 5.66 5.66l-8.57 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"></path>
      </svg>
    </label>
  </span>
</template>

<style scoped>
.attachment-picker {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.attachment-input {
  position: absolute;
  top: 0;
  left: 0;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: 0;
  border: 0;
  opacity: 0;
  overflow: hidden;
  white-space: nowrap;
}

.attachment-trigger {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  color: var(--muted);
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease, border-color 180ms ease, transform 180ms ease;
}

.attachment-trigger:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
  color: var(--text);
}

.attachment-trigger:active {
  transform: scale(0.96);
}

.attachment-input:focus-visible + .attachment-trigger {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.attachment-trigger svg {
  width: 16px;
  height: 16px;
}

@media (prefers-reduced-motion: reduce) {
  .attachment-trigger {
    transition: none;
  }

  .attachment-trigger:active {
    transform: none;
  }
}
</style>

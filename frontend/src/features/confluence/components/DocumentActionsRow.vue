<script setup lang="ts">
import { computed } from "vue"
import type { ConfluenceDocument } from "../types"
import { useToast } from "../../../composables/useToast"

interface Props {
  document: ConfluenceDocument
  isGenerating: boolean
  isApplying: boolean
  hasResult: boolean
  canGenerate: boolean
  canGenerateSop: boolean
  isSopGenerating: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (event: "generate"): void
  (event: "apply"): void
  (event: "sop-generate"): void
}>()

const { showToast } = useToast()

const generateLabel = computed(() => {
  if (props.document.type === "ut") {
    return "Generate UT Docs With AI"
  }
  if (props.document.type === "query") {
    return "Extract query review"
  }
  return "Generate with AI"
})

const busyLabel = computed(() => {
  if (props.document.type === "ut") {
    return "Generating UT docs…"
  }
  if (props.document.type === "query") {
    return "Extracting query review…"
  }
  return "Generating…"
})

const isGenerateDisabled = computed(() => !props.canGenerate || props.isGenerating)

const isSopDocument = computed(() => props.document.type === "sop")

const handleOpenConfluence = () => {
  if (!props.document.url) {
    showToast("This document has no Confluence URL yet.")
    return
  }
  window.open(props.document.url, "_blank", "noopener,noreferrer")
}

const handleGenerate = () => {
  if (isGenerateDisabled.value) {
    return
  }
  emit("generate")
}

const isApplyDisabled = computed(() => !props.hasResult || props.isApplying)

const handleApply = () => {
  if (isApplyDisabled.value) {
    return
  }
  emit("apply")
}

const handleSopGenerate = () => {
  if (props.isSopGenerating) {
    return
  }
  emit("sop-generate")
}

const handleCopy = async () => {
  const text = props.document.description || props.document.title || ""
  if (!navigator.clipboard?.writeText) {
    showToast("Copy failed: clipboard unavailable.")
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    showToast("Description copied to clipboard.")
  } catch {
    showToast("Copy failed: clipboard unavailable.")
  }
}

const handleEdit = () => {
  showToast("Edit is ready for backend wiring.")
}

const handleShare = () => {
  showToast("Share is ready for backend wiring.")
}

const handleMore = () => {
  showToast("More actions are ready for backend wiring.")
}
</script>

<template>
  <div class="detail-actions">
    <button
      class="btn btn-primary"
      type="button"
      data-testid="detail-open-confluence"
      @click="handleOpenConfluence"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M7 17 17 7"></path>
        <path d="M8 7h9v9"></path>
      </svg>
      Open in Confluence BRI
    </button>

    <button
      class="btn btn-ai"
      type="button"
      data-testid="detail-generate"
      :disabled="isGenerateDisabled"
      :aria-busy="props.isGenerating"
      @click="handleGenerate"
    >
      <span v-if="props.isGenerating" class="btn-spinner" aria-hidden="true"></span>
      <svg v-else viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 3v3"></path>
        <path d="M12 18v3"></path>
        <path d="m5.6 5.6 2.1 2.1"></path>
        <path d="m16.3 16.3 2.1 2.1"></path>
        <path d="M3 12h3"></path>
        <path d="M18 12h3"></path>
        <path d="m5.6 18.4 2.1-2.1"></path>
        <path d="m16.3 7.7 2.1-2.1"></path>
        <circle cx="12" cy="12" r="3.5"></circle>
      </svg>
      {{ props.isGenerating ? busyLabel : generateLabel }}
    </button>

    <button
      v-if="props.canGenerateSop"
      class="btn btn-ghost"
      type="button"
      data-testid="detail-generate-sop"
      :disabled="props.isSopGenerating"
      :aria-busy="props.isSopGenerating"
      @click="handleSopGenerate"
    >
      <span v-if="props.isSopGenerating" class="btn-spinner" aria-hidden="true"></span>
      <svg v-else viewBox="0 0 24 24" aria-hidden="true">
        <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8Z"></path>
        <path d="M14 3v5h5"></path>
        <path d="M9 13h6"></path>
        <path d="M9 17h6"></path>
      </svg>
      {{ props.isSopGenerating ? "Generating SOP…" : "Generate SOP Deploy & Operasional" }}
    </button>

    <button
      v-if="!isSopDocument"
      class="btn btn-ghost"
      type="button"
      data-testid="detail-apply"
      :disabled="isApplyDisabled"
      :aria-busy="props.isApplying"
      @click="handleApply"
    >
      <span v-if="props.isApplying" class="btn-spinner" aria-hidden="true"></span>
      <svg v-else viewBox="0 0 24 24" aria-hidden="true">
        <path d="M20 6 9 17l-5-5"></path>
      </svg>
      Instant apply
    </button>

    <button
      v-if="!isSopDocument"
      class="btn btn-ghost"
      type="button"
      data-testid="detail-copy"
      :disabled="!props.hasResult"
      @click="handleCopy"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <rect x="9" y="9" width="11" height="11" rx="2"></rect>
        <path d="M5 15V5a2 2 0 0 1 2-2h10"></path>
      </svg>
      Copy
    </button>

    <div class="spacer"></div>

    <button class="icon-btn" type="button" aria-label="Edit" @click="handleEdit">
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 20h9"></path>
        <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4Z"></path>
      </svg>
    </button>

    <button class="icon-btn" type="button" aria-label="Share" @click="handleShare">
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="18" cy="5" r="3"></circle>
        <circle cx="6" cy="12" r="3"></circle>
        <circle cx="18" cy="19" r="3"></circle>
        <path d="m8.59 13.51 6.83 3.98"></path>
        <path d="m15.41 6.51-6.82 3.98"></path>
      </svg>
    </button>

    <button class="icon-btn" type="button" aria-label="More actions" @click="handleMore">
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="1.5"></circle>
        <circle cx="19" cy="12" r="1.5"></circle>
        <circle cx="5" cy="12" r="1.5"></circle>
      </svg>
    </button>
  </div>
</template>

<style scoped>
.detail-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.detail-actions .spacer {
  flex: 1;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms cubic-bezier(0.2, 0.7, 0.3, 1),
    border-color 160ms cubic-bezier(0.2, 0.7, 0.3, 1),
    color 160ms cubic-bezier(0.2, 0.7, 0.3, 1),
    opacity 160ms cubic-bezier(0.2, 0.7, 0.3, 1);
}

.btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.btn svg {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.btn-spinner {
  width: 12px;
  height: 12px;
  flex: 0 0 auto;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: btn-spin 700ms linear infinite;
}

@keyframes btn-spin {
  to {
    transform: rotate(360deg);
  }
}

.btn-primary {
  background: var(--accent);
  color: var(--accent-contrast);
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-ghost {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
}

.btn-ghost:hover:not(:disabled) {
  border-color: var(--border-strong);
  background: var(--surface-hover);
  color: var(--text);
}

.btn:disabled {
  opacity: 0.32;
  cursor: not-allowed;
  border-color: var(--border);
  background: transparent;
  color: var(--subtle);
  pointer-events: auto;
}

.btn:disabled:hover {
  background: transparent;
  border-color: var(--border);
  color: var(--subtle);
}

.btn-ai {
  border-color: color-mix(in srgb, var(--positive) 45%, transparent);
  background: color-mix(in srgb, var(--positive) 12%, transparent);
  color: var(--positive);
}

.btn-ai:hover:not(:disabled) {
  background: color-mix(in srgb, var(--positive) 20%, transparent);
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 150ms cubic-bezier(0.2, 0.7, 0.3, 1),
    color 150ms cubic-bezier(0.2, 0.7, 0.3, 1),
    border-color 150ms cubic-bezier(0.2, 0.7, 0.3, 1);
}

.icon-btn:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
  color: var(--text);
}

.icon-btn:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.icon-btn svg {
  width: 15px;
  height: 15px;
  stroke: currentColor;
  stroke-width: 1.7;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

@media (max-width: 620px) {
  .detail-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .detail-actions .btn {
    justify-content: center;
  }

  .detail-actions .spacer {
    display: none;
  }

  .detail-actions .icon-btn {
    width: 100%;
  }
}
</style>

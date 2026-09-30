<script setup lang="ts">
import { computed } from "vue"
import type { ConfluenceDocument } from "../types"
import { useToast } from "../../../composables/useToast"

interface Props {
  document: ConfluenceDocument
}

const props = defineProps<Props>()

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

const isSopDocument = computed(() => props.document.type === "sop")

const handleOpenConfluence = () => {
  if (!props.document.url) {
    showToast("This document has no Confluence URL yet.")
    return
  }
  window.open(props.document.url, "_blank", "noopener,noreferrer")
}

const handleGenerate = () => {
  showToast("AI generation is ready for backend wiring.")
}
const handleApply = () => {
  showToast("Instant apply is ready for backend wiring.")
}

const handleCopy = async () => {
  const text = props.document.description || props.document.title || ""
  if (!navigator.clipboard?.writeText) {
    showToast("Copy failed — clipboard unavailable.")
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    showToast("Description copied to clipboard.")
  } catch {
    showToast("Copy failed — clipboard unavailable.")
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
      @click="handleGenerate"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
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
      {{ generateLabel }}
    </button>

    <button
      v-if="!isSopDocument"
      class="btn btn-ghost"
      type="button"
      @click="handleApply"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M20 6 9 17l-5-5"></path>
      </svg>
      Instant apply
    </button>

    <button
      v-if="!isSopDocument"
      class="btn btn-ghost"
      type="button"
      data-testid="detail-copy"
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
  margin-top: 22px;
  flex-wrap: wrap;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--border);
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
    background-color 160ms ease,
    border-color 160ms ease,
    color 160ms ease,
    opacity 160ms ease;
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

.btn-primary {
  background: var(--accent);
  color: #0b0c0f;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-ghost {
  border-color: var(--border);
  background: transparent;
  color: #c8ced5;
}

.btn-ghost:hover {
  border-color: var(--border-strong);
  background: rgba(255, 255, 255, 0.03);
}

.btn-ai {
  border-color: rgba(159, 182, 166, 0.32);
  background: rgba(159, 182, 166, 0.09);
  color: #c1d4c5;
}

.btn-ai:hover {
  background: rgba(159, 182, 166, 0.16);
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    border-color 150ms ease;
}

.icon-btn:hover {
  background: rgba(255, 255, 255, 0.04);
  border-color: var(--border-strong);
  color: var(--text);
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

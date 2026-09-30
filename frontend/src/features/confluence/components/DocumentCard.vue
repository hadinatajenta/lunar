<script setup lang="ts">
import type { ConfluenceDocument } from "../types"
import DocumentTypeBadge from "./DocumentTypeBadge.vue"
import DocumentStatusPill from "./DocumentStatusPill.vue"
import { formatRelativeTime } from "../../../lib/utils"

interface Props {
  document: ConfluenceDocument
}

defineProps<Props>()

const emit = defineEmits<{
  (e: "open", confluenceDocument: ConfluenceDocument): void
}>()
</script>

<template>
  <button
    class="doc-card"
    type="button"
    data-testid="doc-card"
    :data-doc-id="document.id"
    @click="emit('open', document)"
  >
    <div class="doc-card-head">
      <div class="doc-card-head-left">
        <DocumentTypeBadge :type="document.type" :label="document.type_label" />
        <span class="doc-card-id">{{ document.id }}</span>
      </div>
      <DocumentStatusPill :status="document.status" />
    </div>

    <div class="doc-card-title">{{ document.title }}</div>
    <div class="doc-card-desc">{{ document.description }}</div>

    <div class="doc-card-foot">
      <div class="doc-card-meta">
        <span>{{ document.space || "-" }}</span>
        <span class="sep">·</span>
        <span>{{ formatRelativeTime(document.updated) }}</span>
      </div>
      <span class="doc-card-open">
        Open
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M5 12h14"></path>
          <path d="m13 6 6 6-6 6"></path>
        </svg>
      </span>
    </div>
  </button>
</template>

<style scoped>
.doc-card {
  display: flex;
  flex-direction: column;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  text-align: left;
  cursor: pointer;
  transition:
    border-color 160ms ease,
    background-color 160ms ease,
    transform 160ms ease;
}

.doc-card:hover {
  background: var(--surface-hover);
  transform: translateY(-1px);
}

.doc-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.doc-card-head-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.doc-card-id {
  color: var(--subtle);
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.doc-card-title {
  color: var(--text);
  font-size: 13.5px;
  font-weight: 500;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 38px;
}

.doc-card-desc {
  color: var(--subtle);
  font-size: 11px;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.doc-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid var(--border);
}

.doc-card-meta {
  color: var(--muted);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.doc-card-meta .sep {
  opacity: 0.5;
}

.doc-card-open {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: #a9b1bb;
  font-size: 10px;
  font-weight: 500;
  flex: 0 0 auto;
}

.doc-card-open svg {
  width: 11px;
  height: 11px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>

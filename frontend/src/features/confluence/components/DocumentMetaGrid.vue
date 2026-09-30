<script setup lang="ts">
import type { ConfluenceDocument } from "../types"
import { formatRelativeTime } from "../../../lib/utils"

interface Props {
  document: ConfluenceDocument
}

defineProps<Props>()
</script>

<template>
  <div class="detail-meta" data-testid="detail-meta">
    <div class="detail-meta-item">
      <span class="detail-meta-label">Space</span>
      <span class="detail-meta-value">{{ document.space || "-" }}</span>
    </div>
    <div class="detail-meta-item">
      <span class="detail-meta-label">Owner</span>
      <span class="detail-meta-value">{{ document.owner || "-" }}</span>
    </div>
    <div class="detail-meta-item">
      <span class="detail-meta-label">Last Editor</span>
      <span class="detail-meta-value" data-testid="detail-last-editor">{{
        document.last_editor || document.owner || "-"
      }}</span>
    </div>
    <div class="detail-meta-item">
      <span class="detail-meta-label">Updated</span>
      <span class="detail-meta-value">{{ formatRelativeTime(document.updated) }}</span>
    </div>
  </div>
</template>

<style scoped>
.detail-meta {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 18px;
  padding: 20px 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
}

.detail-meta-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-meta-label {
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.detail-meta-value {
  color: var(--text);
  font-size: 12.5px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 640px) {
  .detail-meta {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }
}
</style>

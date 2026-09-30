<script setup lang="ts">
import type { ConfluenceDocument } from "../types"
import { formatRelativeTime } from "../../../lib/utils"

interface Props {
  document: ConfluenceDocument
}

defineProps<Props>()
</script>

<template>
  <div class="detail-meta">
    <div class="detail-meta-item">
      <span class="detail-meta-label">Space</span>
      <span class="detail-meta-value">{{ document.space || "-" }}</span>
    </div>
    <div class="detail-meta-item">
      <span class="detail-meta-label">Owner</span>
      <span class="detail-meta-value">{{ document.owner || "-" }}</span>
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
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
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
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.detail-meta-value {
  color: #dfe3e8;
  font-size: 12.5px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}
</style>

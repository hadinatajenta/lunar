<script setup lang="ts">
import type { ChangedFile } from "../types"
import { resolveFileStatusLabel } from "./service-presentation"

const props = defineProps<{
  files: ChangedFile[]
  isTruncated: boolean
}>()
</script>

<template>
  <div>
    <div v-if="props.files.length === 0" class="file-empty">
      Working tree is clean, there is nothing to review.
    </div>

    <div v-else class="file-list">
      <div v-for="file in props.files" :key="`${file.status}:${file.path}`" class="file-row">
        <span class="file-status" :class="`is-${file.status}`">{{ resolveFileStatusLabel(file.status) }}</span>
        <span class="file-path" :title="file.path"><bdi>{{ file.path }}</bdi></span>
        <span class="file-diff">
          <span class="add">+{{ file.added }}</span>
          <span class="sep">·</span>
          <span class="del">−{{ file.deleted }}</span>
        </span>
      </div>
    </div>

    <p v-if="props.isTruncated" class="truncated-note">
      The changed file list was cut at 200 entries. Open the repository to review the rest.
    </p>
  </div>
</template>

<style scoped>
.file-empty,
.truncated-note {
  margin: 0;
  color: var(--subtle);
  font-size: 11.5px;
  line-height: 1.55;
}

.file-empty {
  padding: 14px 8px;
  text-align: center;
}

.truncated-note {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--border);
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 260px;
  overflow-y: auto;
}

.file-row {
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  padding: 7px 9px;
  border-radius: 7px;
  font-size: 11.5px;
}

.file-row:hover {
  background: var(--surface-hover);
}

.file-status {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border-radius: 5px;
  font-size: 9.5px;
  font-weight: 700;
}

.file-status.is-modified {
  color: var(--warning);
  background: color-mix(in srgb, var(--warning) 14%, transparent);
}

.file-status.is-added {
  color: var(--positive);
  background: color-mix(in srgb, var(--positive) 14%, transparent);
}

.file-status.is-deleted {
  color: var(--danger);
  background: color-mix(in srgb, var(--danger) 14%, transparent);
}

.file-status.is-renamed {
  color: var(--muted);
  background: color-mix(in srgb, var(--muted) 16%, transparent);
}

.file-status.is-untracked {
  color: var(--subtle);
  background: color-mix(in srgb, var(--subtle) 16%, transparent);
}

.file-path {
  overflow: hidden;
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-diff {
  color: var(--subtle);
  font-size: 10.5px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.file-diff .add {
  color: var(--positive);
}

.file-diff .del {
  color: var(--danger);
}

.file-diff .sep {
  margin: 0 4px;
  opacity: 0.5;
}
</style>

<script setup lang="ts">
import type { RepositorySummary } from "../types"
import { formatDirtyLabel } from "./service-presentation"

const props = defineProps<{
  repository: RepositorySummary
  isSelected: boolean
}>()

const emit = defineEmits<{
  (event: "toggle", repositoryName: string, isSelected: boolean): void
}>()

const handleToggle = (event: Event) => {
  const checkbox = event.target as HTMLInputElement
  emit("toggle", props.repository.name, checkbox.checked)
}
</script>

<template>
  <label class="picker-row" :class="{ 'is-disabled': !props.repository.is_git }">
    <input
      class="picker-check"
      type="checkbox"
      :checked="props.isSelected"
      :disabled="!props.repository.is_git"
      :aria-label="`Include ${props.repository.name}`"
      @change="handleToggle"
    />

    <span class="picker-body">
      <span class="picker-name">{{ props.repository.name }}</span>

      <span v-if="props.repository.is_git" class="picker-meta">
        <span v-if="props.repository.branch" class="picker-branch">{{ props.repository.branch }}</span>
        <span class="picker-dirty" :class="{ 'is-dirty': props.repository.dirty_count > 0 }">
          {{ formatDirtyLabel(props.repository.dirty_count) }}
        </span>
        <span v-if="props.repository.updated_relative" class="picker-updated">
          {{ props.repository.updated_relative }}
        </span>
      </span>

      <span v-else class="picker-note">
        This folder is not a git repository, so Lunar cannot read its state and it cannot be tracked.
        <span v-if="props.repository.error" class="picker-note-detail">{{ props.repository.error }}</span>
      </span>
    </span>
  </label>
</template>

<style scoped>
.picker-row {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 11px 13px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
  cursor: pointer;
  transition: border-color 150ms ease, background-color 150ms ease;
}

.picker-row:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
}

.picker-row.is-disabled {
  cursor: not-allowed;
  opacity: 0.75;
}

.picker-check {
  width: 15px;
  height: 15px;
  margin: 2px 0 0;
  flex: 0 0 auto;
  accent-color: var(--text);
  cursor: inherit;
}

.picker-body {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  flex: 1;
}

.picker-name {
  color: var(--text);
  font-size: 12.5px;
  font-weight: 600;
  overflow-wrap: anywhere;
}

.picker-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.picker-branch {
  max-width: 100%;
  padding: 2px 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--muted);
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  overflow-wrap: anywhere;
}

.picker-dirty,
.picker-updated {
  color: var(--subtle);
  font-size: 10.5px;
}

.picker-dirty.is-dirty {
  color: var(--warning);
}

.picker-note {
  color: var(--subtle);
  font-size: 11px;
  line-height: 1.5;
}

.picker-note-detail {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
</style>

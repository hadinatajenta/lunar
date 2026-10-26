<script setup lang="ts">
import { computed, ref, watch } from "vue"
import type { PRDiff } from "../types"

const props = defineProps<{
  diff: PRDiff | null
  loading: boolean
}>()

const collapsedPaths = ref<string[]>([])

watch(
  () => props.diff,
  () => {
    collapsedPaths.value = []
  }
)

const fileList = computed(() => props.diff?.files ?? [])

const isCollapsed = (filePath: string): boolean => collapsedPaths.value.includes(filePath)

const toggleFile = (filePath: string): void => {
  collapsedPaths.value = isCollapsed(filePath)
    ? collapsedPaths.value.filter((path) => path !== filePath)
    : [...collapsedPaths.value, filePath]
}

const statsLabel = (additions: number, deletions: number): string => `+${additions} −${deletions}`

const lineClass = (line: string): string => {
  if (line.startsWith("+")) return "line-add"
  if (line.startsWith("-")) return "line-del"
  return "line-context"
}
</script>

<template>
  <div class="review-diff" data-testid="diff-viewer">
    <div v-if="loading" class="diff-state">
      <span class="diff-spinner" aria-hidden="true"></span>
      <span>Loading changed files…</span>
    </div>

    <div v-else-if="fileList.length === 0" class="diff-state" data-testid="diff-empty">
      No diff changes detected for this pull request.
    </div>

    <div v-else class="diff-file-list" data-testid="diff-file-list">
      <section
        v-for="(file, index) in fileList"
        :key="file.new_path"
        class="diff-file"
        :data-testid="`diff-file-${index}`"
      >
        <h4 class="diff-file-head">
          <button
            class="diff-file-trigger"
            type="button"
            :aria-expanded="!isCollapsed(file.new_path)"
            :data-testid="`diff-file-toggle-${index}`"
            @click="toggleFile(file.new_path)"
          >
            <svg class="chevron" :class="{ 'is-collapsed': isCollapsed(file.new_path) }" viewBox="0 0 24 24" aria-hidden="true">
              <path d="m9 6 6 6-6 6"></path>
            </svg>
            <span class="file-path">{{ file.new_path }}</span>
          </button>
          <span class="file-stats">
            <span class="file-status">{{ file.status }}</span>
            <span class="tabular">{{ statsLabel(file.additions, file.deletions) }}</span>
          </span>
        </h4>

        <div v-show="!isCollapsed(file.new_path)" class="diff-hunks">
          <div v-for="(hunk, hunkIndex) in file.hunks" :key="hunkIndex" class="diff-hunk">
            <div class="hunk-header">{{ hunk.header }}</div>
            <div
              v-for="(line, lineIndex) in hunk.lines"
              :key="lineIndex"
              class="diff-line"
              :class="lineClass(line)"
            >{{ line }}</div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.review-diff {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
  height: 100%;
  overflow-y: auto;
  background: var(--surface-raised);
}

.diff-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px 18px;
  color: var(--subtle);
  font-size: 11.5px;
  text-align: center;
}

.diff-spinner {
  width: 13px;
  height: 13px;
  flex: 0 0 auto;
  border: 2px solid color-mix(in srgb, var(--text) 18%, transparent);
  border-top-color: var(--text);
  border-radius: 50%;
  animation: diff-spin 700ms linear infinite;
}

@keyframes diff-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .diff-spinner {
    animation: none;
  }
}

.diff-file-list {
  display: flex;
  flex-direction: column;
}

.diff-file + .diff-file {
  border-top: 1px solid var(--border);
}

.diff-file-head {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin: 0;
  padding: 8px 12px;
  background: var(--surface-hover);
  border-bottom: 1px solid var(--border);
  font-size: 11px;
  font-weight: 500;
}

.diff-file-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  text-align: left;
  cursor: pointer;
}

.diff-file-trigger:focus-visible {
  outline: 2px solid var(--border-strong);
  outline-offset: 2px;
  border-radius: 4px;
}

.chevron {
  width: 11px;
  height: 11px;
  flex: 0 0 auto;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  transform: rotate(90deg);
  transition: transform 150ms ease-out;
}

.chevron.is-collapsed {
  transform: rotate(0deg);
}

.file-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-stats {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
  color: var(--subtle);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
}

.file-status {
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  text-transform: lowercase;
}

.tabular {
  font-variant-numeric: tabular-nums;
}

.diff-hunks {
  padding-bottom: 6px;
}

.hunk-header {
  padding: 5px 12px;
  background: color-mix(in srgb, var(--text) 4%, transparent);
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
}

.diff-line {
  padding: 1px 12px;
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
  tab-size: 2;
}

.diff-line.line-add {
  background: color-mix(in srgb, var(--positive) 12%, transparent);
  color: var(--positive);
}

.diff-line.line-del {
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
}

.diff-line.line-context {
  color: var(--muted);
}
</style>

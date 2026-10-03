<script setup lang="ts">
import { computed } from "vue"
import type { AgentSource } from "../../code-context/types"

const props = defineProps<{ sources: AgentSource[] }>()

const readLineNumber = (source: AgentSource, key: string): number | null => {
  const entries: Array<[string, unknown]> = Object.entries(source)
  for (const [entryKey, entryValue] of entries) {
    if (entryKey === key && typeof entryValue === "number" && Number.isFinite(entryValue)) {
      return entryValue
    }
  }
  return null
}

const resolveLineLabel = (source: AgentSource): string | null => {
  const startLine = readLineNumber(source, "start_line")
  const endLine = readLineNumber(source, "end_line")
  if (startLine === null) {
    return null
  }
  if (endLine === null || endLine === startLine) {
    return `Line ${startLine}`
  }
  return `Lines ${startLine}-${endLine}`
}

const citations = computed(() =>
  props.sources.map((source, index) => ({
    id: `${source.repo}:${source.file}:${source.type}:${index}`,
    repo: source.repo,
    file: source.file,
    type: source.type,
    snippet: source.snippet.trim(),
    lineLabel: resolveLineLabel(source)
  }))
)
</script>

<template>
  <div class="citation-list">
    <div v-for="citation in citations" :key="citation.id" class="citation-card">
      <div class="citation-head">
        <span class="citation-repo">{{ citation.repo }}</span>
        <span v-if="citation.type" class="citation-type">{{ citation.type }}</span>
      </div>

      <div class="citation-location">
        <span class="citation-file">{{ citation.file }}</span>
        <span v-if="citation.lineLabel" class="citation-lines">{{ citation.lineLabel }}</span>
      </div>

      <pre v-if="citation.snippet" class="citation-snippet">{{ citation.snippet }}</pre>
    </div>
  </div>
</template>

<style scoped>
.citation-list {
  display: grid;
  gap: 8px;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.citation-card {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
}

.citation-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.citation-repo {
  color: var(--text);
  font-size: 11px;
  font-weight: 600;
}

.citation-type {
  padding: 1px 6px;
  border: 1px solid var(--border-strong);
  border-radius: 4px;
  color: var(--muted);
  font-size: 9px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.citation-location {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-top: 4px;
}

.citation-file {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.citation-lines {
  flex-shrink: 0;
  color: var(--positive);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}

.citation-snippet {
  max-height: 132px;
  margin: 8px 0 0;
  padding: 8px 10px;
  overflow: auto;
  border-radius: 6px;
  background: var(--surface);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  line-height: 1.5;
  white-space: pre;
}
</style>

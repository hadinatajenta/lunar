<script setup lang="ts">
import type { ReviewFinding, ReviewSeverity } from "../types"

const props = defineProps<{
  severity: ReviewSeverity
  title: string
  hint: string
  findings: ReviewFinding[]
  indexOffset: number
  testId: string
}>()

const locationLabel = (finding: ReviewFinding): string => {
  if (finding.file === "") return ""
  return finding.line > 0 ? `${finding.file}:${finding.line}` : finding.file
}
</script>

<template>
  <section v-if="props.findings.length > 0" class="finding-group" :data-testid="props.testId">
    <h4 class="group-head">
      <span class="group-dot" :class="`is-${props.severity}`" aria-hidden="true"></span>
      <span>{{ props.title }}</span>
      <span class="group-count">{{ props.findings.length }}</span>
    </h4>
    <p class="group-hint">{{ props.hint }}</p>

    <ol class="finding-list">
      <li
        v-for="(finding, index) in props.findings"
        :key="`${finding.file}-${finding.line}-${finding.title}`"
        class="finding"
        :data-testid="`ai-finding-${props.indexOffset + index}`"
      >
        <div class="finding-head">
          <code v-if="locationLabel(finding)" class="finding-location">{{ locationLabel(finding) }}</code>
          <span class="finding-title">{{ finding.title }}</span>
        </div>

        <div class="finding-tags">
          <span v-if="finding.category" class="tag">{{ finding.category }}</span>
          <span v-if="finding.confidence === 'assumption'" class="tag is-assumption">assumption</span>
        </div>

        <p v-if="finding.explanation" class="finding-detail">
          <span class="detail-key">Why</span>{{ finding.explanation }}
        </p>
        <p v-if="finding.suggested_fix" class="finding-detail">
          <span class="detail-key">Fix</span>{{ finding.suggested_fix }}
        </p>
        <p v-if="finding.evidence" class="finding-detail">
          <span class="detail-key">Evidence</span><code class="detail-code">{{ finding.evidence }}</code>
        </p>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.finding-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.group-head {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0;
  color: var(--text);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

.group-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 auto;
  border-radius: 50%;
}

.group-dot.is-blocker {
  background: var(--danger);
}

.group-dot.is-suggestion {
  background: var(--warning);
}

.group-dot.is-nitpick {
  background: var(--positive);
}

.group-count {
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--subtle);
  font-size: 9.5px;
  font-variant-numeric: tabular-nums;
}

.group-hint {
  margin: 0 0 2px;
  color: var(--subtle);
  font-size: 10.5px;
}

.finding-list {
  display: flex;
  flex-direction: column;
  gap: 9px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.finding {
  padding: 9px 11px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface);
}

.finding-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
}

.finding-location {
  padding: 1px 6px;
  border-radius: 5px;
  background: var(--surface-hover);
  color: var(--muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  word-break: break-all;
}

.finding-title {
  color: var(--text);
  font-size: 11.5px;
  font-weight: 600;
  line-height: 1.5;
}

.finding-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 6px;
}

.tag {
  padding: 1px 6px;
  border: 1px solid var(--border);
  border-radius: 5px;
  color: var(--subtle);
  font-size: 9.5px;
  letter-spacing: 0.03em;
  text-transform: lowercase;
}

.tag.is-assumption {
  border-color: color-mix(in srgb, var(--warning) 35%, transparent);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
  color: var(--warning);
}

.finding-detail {
  display: flex;
  gap: 7px;
  margin: 7px 0 0;
  color: var(--muted);
  font-size: 11px;
  line-height: 1.6;
}

.detail-key {
  flex: 0 0 54px;
  color: var(--subtle);
  font-size: 9.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding-top: 2px;
}

.detail-code {
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--surface-hover);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
  word-break: break-word;
}
</style>

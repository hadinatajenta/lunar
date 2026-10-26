<script setup lang="ts">
import { computed } from "vue"
import type { UtRunDone, UtRunStatus, UtTestFile } from "../types"

const props = defineProps<{
  result: UtRunDone
  repository: string
}>()

const STATUS_LABELS: Record<UtRunStatus, string> = {
  passed: "Passed",
  failed: "Failed",
  blocked: "Blocked",
  not_run: "Not run"
}

const statusLabel = computed(() => STATUS_LABELS[props.result.run.status])
const blockedFiles = computed<UtTestFile[]>(() => props.result.blocked_files)

const durationLabel = computed(() => {
  const durationMs = props.result.run.duration_ms
  if (durationMs < 1000) {
    return `${Math.max(durationMs, 0)} ms`
  }
  const totalSeconds = durationMs / 1000
  if (totalSeconds < 60) {
    return `${totalSeconds.toFixed(1)} s`
  }
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = Math.round(totalSeconds % 60)
  return `${minutes} m ${seconds} s`
})
</script>

<template>
  <div class="ai-run-summary" data-testid="ai-ut-summary">
    <div class="ai-run-head">
      <span class="ai-repository-name tabular" data-testid="ai-ut-summary-repository">
        {{ props.repository }}
      </span>
      <span class="ai-status-pill" :class="`is-${props.result.run.status}`" data-testid="ai-ut-status">
        {{ statusLabel }}
      </span>
      <span class="ai-run-counts tabular" data-testid="ai-ut-counts">
        {{ props.result.run.passed }}/{{ props.result.run.total }} passed
        <template v-if="props.result.run.failed > 0"> · {{ props.result.run.failed }} failed</template>
      </span>
      <span class="ai-run-duration tabular" data-testid="ai-ut-duration">{{ durationLabel }}</span>
    </div>

    <div
      v-if="props.result.stashed || props.result.restored"
      class="ai-run-flags"
      data-testid="ai-ut-worktree"
    >
      <span v-if="props.result.stashed" class="ai-run-flag">Local changes stashed</span>
      <span v-if="props.result.restored" class="ai-run-flag">Local changes restored</span>
    </div>

    <div class="ai-run-block">
      <p class="ai-field-label">Executed packages</p>
      <div v-if="props.result.run.executed_packages.length > 0" class="ai-chips" data-testid="ai-ut-packages">
        <code v-for="packagePath in props.result.run.executed_packages" :key="packagePath" class="ai-chip">
          {{ packagePath }}
        </code>
      </div>
      <p v-else class="ai-run-empty">No package was executed.</p>
    </div>

    <div v-if="blockedFiles.length > 0" class="ai-run-block" data-testid="ai-ut-blocked">
      <p class="ai-field-label">Blocked files (not executed)</p>
      <ul class="ai-blocked-list">
        <li v-for="file in blockedFiles" :key="file.path" data-testid="ai-ut-blocked-file">
          <code>{{ file.path }}</code>
          <span class="ai-blocked-reason">{{ file.reason || "A database reset was detected in this file." }}</span>
          <span class="ai-blocked-tag">Not executed</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.ai-run-summary {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.ai-run-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.ai-repository-name {
  color: var(--text);
  font-size: 12.5px;
  font-weight: 600;
}

.ai-status-pill {
  padding: 3px 10px;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.ai-status-pill.is-passed {
  border-color: color-mix(in srgb, var(--positive) 45%, transparent);
  color: var(--positive);
}

.ai-status-pill.is-failed {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  color: var(--danger);
}

.ai-status-pill.is-blocked {
  border-color: color-mix(in srgb, var(--warning) 45%, transparent);
  color: var(--warning);
}

.ai-status-pill.is-not_run {
  color: var(--muted);
}

.ai-run-counts,
.ai-run-duration {
  color: var(--muted);
  font-size: 11.5px;
}

.ai-run-flags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.ai-run-flag {
  padding: 3px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 10.5px;
}

.ai-run-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ai-field-label {
  margin: 0;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-run-empty {
  margin: 0;
  color: var(--subtle);
  font-size: 11.5px;
}

.ai-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.ai-chip {
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10.5px;
}

.ai-blocked-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-blocked-list li {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
  color: var(--text);
  font-size: 11.5px;
}

.ai-blocked-list code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}

.ai-blocked-reason {
  color: var(--muted);
}

.ai-blocked-tag {
  padding: 2px 7px;
  border: 1px solid color-mix(in srgb, var(--warning) 45%, transparent);
  border-radius: 999px;
  color: var(--warning);
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.tabular {
  font-variant-numeric: tabular-nums;
}
</style>

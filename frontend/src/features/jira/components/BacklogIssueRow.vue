<script setup lang="ts">
import type { JiraIssue } from "../types"
import IssueBranchPill from "./IssueBranchPill.vue"

interface Props {
  issue: JiraIssue
}

defineProps<Props>()

const emit = defineEmits<{
  (e: "select-issue", issue: JiraIssue): void
}>()

const typeIcon = (kind: string): string => {
  switch (kind) {
    case "story":
      return "S"
    case "task":
      return "T"
    case "bug":
      return "B"
    case "subtask":
      return "S"
    case "ut":
      return "U"
    case "query":
      return "Q"
    case "sop":
      return "P"
    default:
      return kind ? kind[0].toUpperCase() : "T"
  }
}

const formatStatus = (rawStatus: string): string => {
  const status = (rawStatus || "").toLowerCase()
  if (status === "progress" || status.includes("progress")) {
    return "In progress"
  }
  if (status === "done" || status.includes("done")) {
    return "Done"
  }
  return "Open"
}
</script>

<template>
  <div
    class="backlog-issue"
    :data-testid="`backlog-issue-${issue.id}`"
    role="button"
    tabindex="0"
    @click="emit('select-issue', issue)"
    @keydown.enter="emit('select-issue', issue)"
    @keydown.space.prevent="emit('select-issue', issue)"
  >
    <div class="backlog-issue-inner" data-testid="backlog-issue">
      <span :class="['issue-type', `is-${issue.kind}`]">{{ typeIcon(issue.kind) }}</span>
      <span class="issue-id">{{ issue.id }}</span>
      <span class="issue-title">{{ issue.title }}</span>
      <IssueBranchPill :issue-key="issue.key || issue.id" />
      <span :class="['issue-status', `is-${issue.status}`]">
        {{ formatStatus(issue.status) }}
      </span>
      <span class="issue-points">{{ issue.points }}</span>
    </div>
  </div>
</template>

<style scoped>
.backlog-issue {
  padding: 0;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
  cursor: pointer;
  transition: background-color 150ms ease;
  user-select: none;
}

.backlog-issue:last-child {
  border-bottom: 0;
}

.backlog-issue:hover {
  background: var(--surface-hover);
}

.backlog-issue:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.backlog-issue-inner {
  display: grid;
  grid-template-columns: 24px 100px minmax(0, 1fr) auto auto 40px;
  gap: 12px;
  align-items: center;
  padding: 10px 16px;
  width: 100%;
}

.issue-type {
  display: grid;
  place-items: center;
  width: 20px;
  height: 20px;
  border-radius: 5px;
  font-size: 10px;
  font-weight: 700;
  color: var(--accent-contrast);
  flex: 0 0 auto;
}

.issue-type.is-story,
.issue-type.is-ut {
  background: var(--positive);
}

.issue-type.is-task,
.issue-type.is-subtask,
.issue-type.is-sop {
  background: var(--muted);
}

.issue-type.is-bug {
  background: var(--danger);
}

.issue-type.is-query {
  background: var(--warning);
}

.issue-id {
  color: var(--muted);
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.02em;
}

.issue-title {
  color: var(--text);
  font-size: 12.5px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.issue-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 8px;
  border-radius: 5px;
  font-size: 10px;
  font-weight: 500;
  white-space: nowrap;
}

.issue-status::before {
  content: "";
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.issue-status.is-open {
  color: var(--muted);
  background: var(--surface-raised);
  border: 1px solid var(--border);
}

.issue-status.is-progress {
  color: var(--warning);
  background: color-mix(in srgb, var(--warning) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--warning) 25%, transparent);
}

.issue-status.is-done {
  color: var(--positive);
  background: color-mix(in srgb, var(--positive) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--positive) 25%, transparent);
}

.issue-points {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 620px) {
  .backlog-issue-inner {
    grid-template-columns: 24px minmax(0, 1fr) auto auto 40px;
  }

  .backlog-issue-inner .issue-id {
    display: none;
  }
}
</style>

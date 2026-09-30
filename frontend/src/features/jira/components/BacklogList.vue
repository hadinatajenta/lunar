<script setup lang="ts">
import type { JiraIssue, JiraSprint } from "../types"

interface Props {
  sprints: JiraSprint[]
  availableSquads?: string[]
  selectedSquad?: string
  detectedSquad?: string
}

withDefaults(defineProps<Props>(), {
  availableSquads: () => [],
  selectedSquad: "",
  detectedSquad: ""
})

const emit = defineEmits<{
  (e: "select-issue", issue: JiraIssue): void
  (e: "select-squad", squad: string): void
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
  <div class="backlog-container" data-testid="backlog-container">
    <div
      v-if="availableSquads && availableSquads.length > 1"
      class="squad-filter-bar"
      data-testid="squad-filter-bar"
    >
      <button
        :class="['squad-pill', { 'is-active': selectedSquad === 'all' }]"
        type="button"
        data-testid="squad-pill-all"
        @click="emit('select-squad', 'all')"
      >
        All squads
      </button>
      <button
        v-for="squad in availableSquads"
        :key="squad"
        :class="['squad-pill', { 'is-active': selectedSquad === squad }]"
        type="button"
        :data-testid="`squad-pill-${squad.toLowerCase().replace(/\s+/g, '-')}`"
        @click="emit('select-squad', squad)"
      >
        <span>{{ squad }}</span>
        <span v-if="detectedSquad === squad" class="squad-badge-mine">Yours</span>
      </button>
    </div>

    <div
      v-if="sprints.length === 0"
      class="backlog-empty"
      data-testid="backlog-empty"
    >
      No sprints available for this squad
    </div>

    <div
      v-for="sprint in sprints"
      :key="sprint.id"
      class="backlog-sprint"
      data-testid="backlog-sprint"
    >
      <div class="backlog-sprint-header">
        <div class="backlog-sprint-title">
          <svg
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M6 9l6 6 6-6"></path>
          </svg>
          <span>{{ sprint.name }}</span>
          <span v-if="sprint.is_active" class="backlog-sprint-badge is-active">ACTIVE</span>
        </div>

        <div class="backlog-sprint-meta">
          <span class="backlog-sprint-dates">{{ sprint.dates }}</span>
        </div>
      </div>

      <div class="backlog-issues">
        <div
          v-for="issue in sprint.issues"
          :key="issue.id"
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
            <span :class="['issue-status', `is-${issue.status}`]">
              {{ formatStatus(issue.status) }}
            </span>
            <span class="issue-points">{{ issue.points }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.backlog-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.squad-filter-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 6px 8px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
}

.squad-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--muted);
  font-size: 11.5px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 140ms ease,
    color 140ms ease,
    border-color 140ms ease;
}

.squad-pill:hover {
  color: var(--text);
  background: var(--surface-hover);
}

.squad-pill.is-active {
  color: var(--text);
  background: var(--surface);
  border-color: var(--border);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.squad-badge-mine {
  padding: 1px 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--positive) 15%, transparent);
  border: 1px solid color-mix(in srgb, var(--positive) 25%, transparent);
  color: var(--positive);
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
}

.backlog-empty {
  padding: 48px 24px;
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: 12px;
  color: var(--muted);
  font-size: 13px;
}

.backlog-sprint {
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  overflow: hidden;
}

.backlog-sprint-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 16px;
  background: var(--surface-raised);
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}

.backlog-sprint-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.backlog-sprint-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  border-radius: 5px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.backlog-sprint-badge.is-active {
  color: var(--positive);
  background: color-mix(in srgb, var(--positive) 15%, transparent);
  border: 1px solid color-mix(in srgb, var(--positive) 30%, transparent);
}

.backlog-sprint-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--muted);
  font-size: 11px;
}

.backlog-sprint-dates {
  font-variant-numeric: tabular-nums;
}

.backlog-issues {
  display: flex;
  flex-direction: column;
}

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
  grid-template-columns: 24px 100px 1fr auto 40px;
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

.issue-type.is-story {
  background: var(--positive);
}

.issue-type.is-task {
  background: var(--muted);
}

.issue-type.is-bug {
  background: var(--danger);
}

.issue-type.is-subtask {
  background: var(--muted);
}

.issue-type.is-ut {
  background: var(--positive);
}

.issue-type.is-query {
  background: var(--warning);
}

.issue-type.is-sop {
  background: var(--muted);
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
    grid-template-columns: 24px 1fr auto 40px;
  }

  .backlog-issue-inner .issue-id {
    display: none;
  }
}
</style>

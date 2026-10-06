<script setup lang="ts">
import { ref } from "vue"
import type { JiraIssue, JiraSprint } from "../types"
import BacklogIssueRow from "./BacklogIssueRow.vue"

interface Props {
  sprint: JiraSprint
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: "select-issue", issue: JiraIssue): void
}>()

const isExpanded = ref(props.sprint.is_active === true)

const issuesElementId = `backlog-sprint-issues-${props.sprint.id}`
</script>

<template>
  <div class="backlog-sprint" data-testid="backlog-sprint">
    <button
      class="backlog-sprint-header"
      type="button"
      :aria-expanded="isExpanded"
      :aria-controls="issuesElementId"
      :data-testid="`sprint-header-${sprint.id}`"
      @click="isExpanded = !isExpanded"
    >
      <span class="backlog-sprint-title">
        <svg
          :class="['backlog-sprint-chevron', { 'is-expanded': isExpanded }]"
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
      </span>

      <span class="backlog-sprint-meta">
        <span class="backlog-sprint-dates">{{ sprint.dates }}</span>
      </span>
    </button>

    <div v-show="isExpanded" :id="issuesElementId" class="backlog-issues">
      <BacklogIssueRow
        v-for="issue in sprint.issues"
        :key="issue.id"
        :issue="issue"
        @select-issue="emit('select-issue', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
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
  width: 100%;
  padding: 12px 16px;
  border: 0;
  border-bottom: 1px solid var(--border);
  background: var(--surface-raised);
  color: var(--text);
  font: inherit;
  text-align: left;
  cursor: pointer;
  flex-wrap: wrap;
  transition: background-color 150ms ease;
}

.backlog-sprint-header:hover {
  background: var(--surface-hover);
}

.backlog-sprint-header:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.backlog-sprint-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.backlog-sprint-chevron {
  flex-shrink: 0;
  color: var(--muted);
  transform: rotate(-90deg);
  transition: transform 160ms ease;
}

.backlog-sprint-chevron.is-expanded {
  transform: rotate(0deg);
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
  font-weight: 400;
}

.backlog-sprint-dates {
  font-variant-numeric: tabular-nums;
}

.backlog-issues {
  display: flex;
  flex-direction: column;
}
</style>

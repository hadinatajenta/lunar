<script setup lang="ts">
import { computed } from "vue"
import type { JiraIssue } from "../types"

interface Props {
  item: JiraIssue
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: "click", item: JiraIssue): void
}>()

const typeBadgeLabel = computed(() => {
  if (props.item.label) {
    return props.item.label
  }
  switch (props.item.kind) {
    case "bug":
      return "BUG"
    case "subtask":
      return "SUB"
    case "ut":
      return "UT"
    case "query":
      return "QR"
    case "sop":
      return "SOP"
    case "story":
      return "STORY"
    case "task":
      return "TASK"
    default:
      return (props.item.kind || "TASK").toUpperCase()
  }
})

const assigneeInitials = computed(() => {
  if (typeof props.item.assignee === "string" && props.item.assignee.length > 0) {
    return props.item.assignee.slice(0, 2).toUpperCase()
  }
  if (props.item.assignee && typeof props.item.assignee === "object") {
    if (props.item.assignee.initials) {
      return props.item.assignee.initials
    }
    if (props.item.assignee.display_name) {
      const parts = props.item.assignee.display_name.trim().split(/\s+/)
      if (parts.length >= 2) {
        return (parts[0][0] + parts[1][0]).toUpperCase()
      }
      return parts[0].slice(0, 2).toUpperCase()
    }
  }
  return "ER"
})

const priorityClass = computed(() => {
  const priority = (props.item.priority || "medium").toLowerCase()
  if (priority.includes("high") || priority.includes("crit")) {
    return "is-high"
  }
  if (priority.includes("low")) {
    return "is-low"
  }
  return "is-medium"
})
</script>

<template>
  <button
    class="kanban-card"
    type="button"
    :data-testid="`kanban-card-${item.id}`"
    @click="emit('click', item)"
  >
    <div class="kanban-card-inner" data-testid="kanban-card">
      <div class="kanban-card-top">
        <span :class="['item-type-badge', `is-${item.kind}`]">{{ typeBadgeLabel }}</span>
        <span
          :class="['priority-dot', priorityClass]"
          :title="`${item.priority || 'medium'} priority`"
        ></span>
      </div>
      <div class="kanban-card-title">{{ item.title }}</div>
      <div class="kanban-card-sub">
        {{ item.id }}{{ item.sub ? ' · ' + item.sub : '' }}
      </div>
      <div class="kanban-card-foot">
        <span class="kanban-card-points">{{ item.points }} pts</span>
        <span class="kanban-card-assignee">{{ assigneeInitials }}</span>
      </div>
    </div>
  </button>
</template>

<style scoped>
.kanban-card {
  display: flex;
  flex-direction: column;
  width: 100%;
  padding: 12px 13px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 160ms ease,
    background-color 160ms ease,
    box-shadow 160ms ease,
    transform 160ms ease;
}

.kanban-card:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.kanban-card-inner {
  display: flex;
  flex-direction: column;
  gap: 9px;
  width: 100%;
}

.kanban-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.item-type-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 34px;
  height: 20px;
  padding: 0 7px;
  border: 1px solid var(--border);
  border-radius: 5px;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.item-type-badge.is-bug {
  color: var(--danger);
  border-color: color-mix(in srgb, var(--danger) 30%, transparent);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
}

.item-type-badge.is-subtask {
  color: var(--muted);
  border-color: var(--border);
  background: var(--surface-raised);
}

.item-type-badge.is-ut {
  color: var(--positive);
  border-color: color-mix(in srgb, var(--positive) 30%, transparent);
  background: color-mix(in srgb, var(--positive) 10%, transparent);
}

.item-type-badge.is-query {
  color: var(--warning);
  border-color: color-mix(in srgb, var(--warning) 30%, transparent);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
}

.item-type-badge.is-sop {
  color: var(--muted);
  border-color: var(--border);
  background: var(--surface-raised);
}

.item-type-badge.is-story {
  color: var(--positive);
  border-color: color-mix(in srgb, var(--positive) 30%, transparent);
  background: color-mix(in srgb, var(--positive) 10%, transparent);
}

.item-type-badge.is-task {
  color: var(--muted);
  border-color: var(--border);
  background: var(--surface-raised);
}

.priority-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex: 0 0 auto;
}

.priority-dot.is-high {
  background: var(--danger);
  box-shadow: 0 0 8px color-mix(in srgb, var(--danger) 40%, transparent);
}

.priority-dot.is-medium {
  background: var(--warning);
}

.priority-dot.is-low {
  background: var(--positive);
}

.kanban-card-title {
  color: var(--text);
  font-size: 12.5px;
  font-weight: 500;
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.kanban-card-sub {
  color: var(--muted);
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kanban-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 2px;
}

.kanban-card-points {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 7px;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
}

.kanban-card-assignee {
  width: 20px;
  height: 20px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 9px;
  font-weight: 600;
}
</style>

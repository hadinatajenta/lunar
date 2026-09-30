<script setup lang="ts">
import type { JiraIssue } from "../types"
import KanbanCard from "./KanbanCard.vue"

interface Props {
  columns: {
    open: JiraIssue[]
    progress: JiraIssue[]
    done: JiraIssue[]
  }
  columnLimits?: Record<string, number>
}

const props = withDefaults(defineProps<Props>(), {
  columnLimits: () => ({
    open: 5,
    progress: 5,
    done: 5
  })
})

const emit = defineEmits<{
  (e: "load-more", status: string): void
  (e: "select-issue", issue: JiraIssue): void
}>()

interface ColumnDefinition {
  key: "open" | "progress" | "done"
  title: string
  dotClass: string
}

const columnDefs: ColumnDefinition[] = [
  { key: "open", title: "Open", dotClass: "is-open" },
  { key: "progress", title: "In Progress", dotClass: "is-progress" },
  { key: "done", title: "Done", dotClass: "is-done" }
]

const getVisibleIssues = (colKey: "open" | "progress" | "done"): JiraIssue[] => {
  const issues = props.columns[colKey] || []
  const limit = props.columnLimits[colKey] ?? 5
  return issues.slice(0, limit)
}

const getRemainingCount = (colKey: "open" | "progress" | "done"): number => {
  const total = (props.columns[colKey] || []).length
  const limit = props.columnLimits[colKey] ?? 5
  return Math.max(0, total - limit)
}
</script>

<template>
  <div class="kanban">
    <div
      v-for="col in columnDefs"
      :key="col.key"
      class="kanban-col"
      :data-testid="`kanban-col-${col.key}`"
    >
      <div class="kanban-col-head">
        <span class="kanban-col-title">
          <span :class="['kanban-col-dot', col.dotClass]"></span>
          {{ col.title }}
        </span>
        <span class="kanban-col-count" :data-testid="`col-count-${col.key}`">
          {{ (columns[col.key] || []).length }}
        </span>
      </div>

      <div class="kanban-cards">
        <div
          v-if="(columns[col.key] || []).length === 0"
          class="kanban-empty"
          :data-testid="`kanban-empty-${col.key}`"
        >
          Nothing here
        </div>

        <template v-else>
          <KanbanCard
            v-for="item in getVisibleIssues(col.key)"
            :key="item.id"
            :item="item"
            @click="emit('select-issue', item)"
          />

          <button
            v-if="getRemainingCount(col.key) > 0"
            class="btn-load-more"
            type="button"
            :data-testid="`btn-load-more-${col.key}`"
            @click="emit('load-more', col.key)"
          >
            Load more ({{ getRemainingCount(col.key) }} more)
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kanban {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.kanban-col {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: rgba(16, 18, 23, 0.5);
  min-height: 320px;
}

.kanban-col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.kanban-col-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #dfe3e8;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.kanban-col-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.kanban-col-dot.is-open {
  background: #7d8590;
}

.kanban-col-dot.is-progress {
  background: #bba984;
  box-shadow: 0 0 8px rgba(187, 169, 132, 0.4);
}

.kanban-col-dot.is-done {
  background: #9fb6a6;
}

.kanban-col-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 20px;
  padding: 0 7px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  color: var(--subtle);
  font-size: 10.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.kanban-cards {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}

.kanban-empty {
  padding: 24px 12px;
  text-align: center;
  color: var(--subtle);
  font-size: 11px;
  border: 1px dashed var(--border);
  border-radius: 9px;
  margin-top: 4px;
}

.btn-load-more {
  width: 100%;
  min-height: 32px;
  padding: 6px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 160ms ease,
    color 160ms ease,
    border-color 160ms ease;
  margin-top: 4px;
}

.btn-load-more:hover {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text);
  border-color: var(--border-strong);
}

@media (max-width: 1000px) {
  .kanban {
    grid-template-columns: 1fr;
  }

  .kanban-col {
    min-height: 180px;
  }
}
</style>

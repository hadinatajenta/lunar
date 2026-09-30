<script setup lang="ts">
import type { DocTypeFilter, StatsCounts } from "../types"

interface Props {
  searchValue: string
  currentFilter: DocTypeFilter
  counts: StatsCounts
  visibleSummary: string
}

defineProps<Props>()

const emit = defineEmits<{
  (e: "search", value: string): void
  (e: "select-filter", filter: DocTypeFilter): void
}>()

const filterEntries = [
  { key: "all", label: "All" },
  { key: "ut", label: "UT" },
  { key: "query", label: "Query Review" },
  { key: "sop", label: "SOP" }
] as const satisfies readonly { key: DocTypeFilter; label: string }[]

const handleSearchInput = (event: Event) => {
  const input = event.target as HTMLInputElement
  emit("search", input.value)
}
</script>

<template>
  <div class="toolbar">
    <div class="toolbar-left">
      <div class="search">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="11" cy="11" r="6.5"></circle>
          <path d="m16 16 5 5"></path>
        </svg>
        <input
          type="search"
          data-testid="doc-search"
          placeholder="Search by title, ID, or keyword…"
          :value="searchValue"
          @input="handleSearchInput"
        />
      </div>

      <div class="filter" role="tablist" aria-label="Document type">
        <button
          v-for="entry in filterEntries"
          :key="entry.key"
          :class="{ 'is-active': currentFilter === entry.key }"
          type="button"
          role="tab"
          data-testid="doc-filter"
          :data-filter="entry.key"
          @click="emit('select-filter', entry.key)"
        >
          {{ entry.label }}
          <span class="chip">{{ counts[entry.key] }}</span>
        </button>
      </div>
    </div>

    <div class="toolbar-right">
      <span data-testid="doc-visible-count">{{ visibleSummary }}</span>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.search {
  position: relative;
  flex: 1;
  max-width: 340px;
}

.search svg {
  position: absolute;
  left: 11px;
  top: 50%;
  transform: translateY(-50%);
  width: 14px;
  height: 14px;
  stroke: var(--subtle);
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
}

.search input {
  width: 100%;
  height: 36px;
  padding: 0 12px 0 34px;
  border: 1px solid var(--border);
  border-radius: 9px;
  outline: 0;
  background: rgba(255, 255, 255, 0.022);
  color: #ebeff3;
  font-size: 12px;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease;
}

.search input::placeholder {
  color: #5d6570;
}

.search input:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.03);
}

.filter {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(16, 18, 23, 0.6);
}

.filter button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 6px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease;
}

.filter button:hover {
  background: rgba(255, 255, 255, 0.04);
  color: var(--text);
}

.filter button.is-active {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text);
}

.filter button .chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 16px;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  font-size: 9.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.filter button.is-active .chip {
  background: rgba(255, 255, 255, 0.14);
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--subtle);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 820px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left {
    flex-direction: column;
    align-items: stretch;
  }

  .search {
    max-width: none;
  }

  .filter {
    overflow-x: auto;
  }
}
</style>

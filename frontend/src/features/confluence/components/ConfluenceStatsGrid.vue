<script setup lang="ts">
import type { StatsCounts } from "../types"

interface Props {
  counts: StatsCounts
}

defineProps<Props>()

const statEntries = [
  { key: "all", label: "All documents", sub: "Across all spaces", dotClass: "is-all" },
  { key: "ut", label: "UT Documents", sub: "Unit test specs", dotClass: "is-ut" },
  { key: "query", label: "Query Review", sub: "SQL reviews pending", dotClass: "is-query" },
  { key: "sop", label: "SOP", sub: "Monitoring · Ops · Rollback", dotClass: "is-sop" }
] as const
</script>

<template>
  <div class="stats-grid">
    <div
      v-for="entry in statEntries"
      :key="entry.key"
      class="stat-card"
      data-testid="stat-card"
      :data-stat="entry.key"
    >
      <div class="stat-label">
        <span class="stat-dot" :class="entry.dotClass"></span>
        {{ entry.label }}
      </div>
      <div class="stat-value">{{ counts[entry.key] }}</div>
      <div class="stat-sub">{{ entry.sub }}</div>
    </div>
  </div>
</template>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 22px;
}

.stat-card {
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.stat-label {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.stat-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.stat-dot.is-all {
  color: #a9b1bb;
}

.stat-dot.is-ut {
  color: #b5cbbd;
}

.stat-dot.is-query {
  color: #cbbb97;
}

.stat-dot.is-sop {
  color: #a9b8c9;
}

.stat-value {
  margin-top: 9px;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.stat-sub {
  margin-top: 3px;
  color: var(--subtle);
  font-size: 10px;
}

@media (max-width: 1000px) {
  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>

<script setup lang="ts">
const columns = [
  { key: "open", title: "Open" },
  { key: "progress", title: "In Progress" },
  { key: "done", title: "Done" }
]
</script>

<template>
  <div class="kanban-skeleton" data-testid="kanban-skeleton">
    <div
      v-for="col in columns"
      :key="col.key"
      class="skeleton-col"
    >
      <div class="skeleton-head">
        <div class="skeleton-title-pill"></div>
        <div class="skeleton-count-circle"></div>
      </div>

      <div class="skeleton-cards">
        <div
          v-for="i in 3"
          :key="i"
          class="skeleton-card"
        >
          <div class="skeleton-card-top">
            <div class="skeleton-chip"></div>
            <div class="skeleton-dot"></div>
          </div>
          <div class="skeleton-line is-long"></div>
          <div class="skeleton-line is-short"></div>
          <div class="skeleton-card-bottom">
            <div class="skeleton-key"></div>
            <div class="skeleton-avatar"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kanban-skeleton {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
}

.skeleton-col {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: rgba(16, 18, 23, 0.5);
  min-height: 380px;
}

.skeleton-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.skeleton-title-pill {
  width: 72px;
  height: 12px;
  border-radius: 6px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-count-circle {
  width: 22px;
  height: 18px;
  border-radius: 999px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-cards {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.skeleton-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.skeleton-chip {
  width: 52px;
  height: 18px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-line {
  height: 11px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-line.is-long {
  width: 95%;
}

.skeleton-line.is-short {
  width: 65%;
}

.skeleton-card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 4px;
}

.skeleton-key {
  width: 84px;
  height: 12px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

@keyframes pulse-shimmer {
  0%, 100% {
    opacity: 0.35;
  }
  50% {
    opacity: 0.75;
  }
}

@media (max-width: 900px) {
  .kanban-skeleton {
    grid-template-columns: 1fr;
  }
}
</style>

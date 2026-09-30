<script setup lang="ts">
</script>

<template>
  <div class="backlog-skeleton" data-testid="backlog-skeleton">
    <div
      v-for="s in 2"
      :key="s"
      class="skeleton-sprint"
    >
      <div class="skeleton-sprint-head">
        <div class="skeleton-sprint-title"></div>
        <div class="skeleton-sprint-date"></div>
      </div>

      <div class="skeleton-sprint-rows">
        <div
          v-for="r in 4"
          :key="r"
          class="skeleton-row"
        >
          <div class="skeleton-type-pill"></div>
          <div class="skeleton-issue-key"></div>
          <div class="skeleton-issue-summary"></div>
          <div class="skeleton-issue-status"></div>
          <div class="skeleton-issue-pts"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.backlog-skeleton {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.skeleton-sprint {
  border: 1px solid var(--border);
  border-radius: 12px;
  background: rgba(16, 18, 23, 0.6);
  overflow: hidden;
}

.skeleton-sprint-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid var(--border);
}

.skeleton-sprint-title {
  width: 190px;
  height: 14px;
  border-radius: 6px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-sprint-date {
  width: 140px;
  height: 12px;
  border-radius: 6px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
}

.skeleton-sprint-rows {
  display: flex;
  flex-direction: column;
}

.skeleton-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}

.skeleton-row:last-child {
  border-bottom: 0;
}

.skeleton-type-pill {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
  flex-shrink: 0;
}

.skeleton-issue-key {
  width: 86px;
  height: 12px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
  flex-shrink: 0;
}

.skeleton-issue-summary {
  flex: 1;
  height: 12px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
  min-width: 100px;
}

.skeleton-issue-status {
  width: 64px;
  height: 16px;
  border-radius: 4px;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
  flex-shrink: 0;
}

.skeleton-issue-pts {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--surface-raised);
  animation: pulse-shimmer 1.6s ease-in-out infinite;
  flex-shrink: 0;
}

@keyframes pulse-shimmer {
  0%, 100% {
    opacity: 0.35;
  }
  50% {
    opacity: 0.75;
  }
}

@media (max-width: 768px) {
  .skeleton-issue-summary {
    display: none;
  }
}
</style>

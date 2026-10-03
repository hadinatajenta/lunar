<script setup lang="ts">
const skeletonRows = [1, 2, 3, 4, 5, 6]
</script>

<template>
  <div class="table-wrap" data-testid="services-skeleton" aria-busy="true">
    <table class="table">
      <thead>
        <tr>
          <th>Service</th>
          <th class="col-branch">Branch</th>
          <th>Dirty</th>
          <th class="col-updated">Updated</th>
          <th class="col-action"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="rowNumber in skeletonRows" :key="rowNumber" class="skeleton-row">
          <td>
            <div class="skeleton-service">
              <span class="skeleton-mark"></span>
              <span class="skeleton-text">
                <span class="skeleton-line is-name"></span>
                <span class="skeleton-line is-sub"></span>
              </span>
            </div>
          </td>
          <td class="col-branch"><span class="skeleton-pill is-branch"></span></td>
          <td><span class="skeleton-pill is-dirty"></span></td>
          <td class="col-updated"><span class="skeleton-line is-updated"></span></td>
          <td class="col-action"><span class="skeleton-action"></span></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  overflow: hidden;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table thead th {
  padding: 11px 18px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-raised);
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  text-align: left;
  white-space: nowrap;
}

.table thead th.col-action {
  width: 44px;
}

.skeleton-row td {
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}

.skeleton-service {
  display: flex;
  align-items: center;
  gap: 11px;
}

.skeleton-text {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.skeleton-mark,
.skeleton-line,
.skeleton-pill,
.skeleton-action {
  display: block;
  border-radius: 6px;
  background: var(--surface-hover);
  animation: skeleton-pulse 1.4s ease-in-out infinite;
}

.skeleton-mark {
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
}

.skeleton-line.is-name {
  width: 150px;
  height: 11px;
}

.skeleton-line.is-sub {
  width: 74px;
  height: 8px;
}

.skeleton-line.is-updated {
  width: 86px;
  height: 10px;
}

.skeleton-pill.is-branch {
  width: 130px;
  height: 20px;
}

.skeleton-pill.is-dirty {
  width: 96px;
  height: 20px;
}

.skeleton-action {
  width: 30px;
  height: 30px;
  margin-left: auto;
}

@keyframes skeleton-pulse {
  0%,
  100% {
    opacity: 0.55;
  }
  50% {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-mark,
  .skeleton-line,
  .skeleton-pill,
  .skeleton-action {
    animation: none;
  }
}

@media (max-width: 820px) {
  .table thead th.col-branch,
  .skeleton-row td.col-branch {
    display: none;
  }
}

@media (max-width: 620px) {
  .table thead th.col-updated,
  .skeleton-row td.col-updated {
    display: none;
  }
}
</style>

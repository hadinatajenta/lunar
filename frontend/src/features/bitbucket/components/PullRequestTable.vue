<script setup lang="ts">
import type { PullRequest } from "../types"

defineProps<{
  pullRequests: PullRequest[]
  activeFilter: string
  loading: boolean
}>()

const emit = defineEmits<{
  (e: "filter", filter: string): void
  (e: "review", pr: PullRequest): void
}>()
</script>

<template>
  <section class="bb-section">
    <div class="bb-section-head">
      <div>
        <div class="bb-section-title">Open pull requests</div>
        <div class="bb-section-desc">Click Review to open the pull request and run an AI review.</div>
      </div>
      <div class="bb-filter" data-bb-filter="prs">
        <button
          type="button"
          :class="{ 'is-active': activeFilter === 'all' }"
          data-testid="pr-filter-all"
          @click="emit('filter', 'all')"
        >
          All
        </button>
        <button
          type="button"
          :class="{ 'is-active': activeFilter === 'mine' }"
          data-testid="pr-filter-mine"
          @click="emit('filter', 'mine')"
        >
          Assigned to me
        </button>
        <button
          type="button"
          :class="{ 'is-active': activeFilter === 'ai' }"
          data-testid="pr-filter-ai"
          @click="emit('filter', 'ai')"
        >
          AI flagged
        </button>
      </div>
    </div>

    <div v-if="loading" class="empty-state">
      Loading pull requests…
    </div>

    <div v-else-if="pullRequests.length === 0" class="empty-state">
      No pull requests match this filter.
    </div>

    <div v-else class="pr-table-wrap">
      <table class="pr-table">
        <thead>
          <tr>
            <th>Pull request</th>
            <th class="col-status">Status</th>
            <th class="col-updated">Updated</th>
            <th style="width: 1%"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="pr in pullRequests"
            :key="pr.id"
            :data-testid="`pr-row-${pr.number}`"
          >
            <td>
              <div class="pr-cell-content">
                <div class="pr-cell-meta">
                  <span class="pr-cell-id">{{ pr.id }} · {{ pr.repo }}</span>
                  <span
                    class="branch-pill"
                    :title="pr.target_branch ? `${pr.source_branch} → ${pr.target_branch}` : pr.source_branch"
                  >
                    <svg viewBox="0 0 24 24" aria-hidden="true">
                      <circle cx="6" cy="6" r="2.5"></circle>
                      <circle cx="6" cy="18" r="2.5"></circle>
                      <circle cx="18" cy="12" r="2.5"></circle>
                      <path d="M6 8.5v7"></path>
                      <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
                    </svg>
                    <span class="branch-text">{{ pr.source_branch }}</span>
                  </span>
                  <span v-if="pr.target_branch" class="branch-arrow">→ {{ pr.target_branch }}</span>
                </div>
                <div class="pr-cell-name" :title="pr.title">{{ pr.title }}</div>
              </div>
            </td>
            <td class="col-status">
              <span
                class="status-pill"
                :class="{
                  'is-open': pr.status === 'open' || pr.status === 'approved',
                  'is-draft': pr.status === 'draft',
                  'is-declined': pr.status === 'declined',
                  'is-needs-work': pr.status === 'needs_work',
                }"
              >
                {{ pr.status.replace('_', ' ') }}
              </span>
            </td>
            <td class="col-updated">
              <span class="pr-cell-muted">{{ pr.updated_relative }}</span>
            </td>
            <td class="is-actions">
              <button
                class="review-btn"
                type="button"
                :data-testid="`btn-review-${pr.number}`"
                @click="emit('review', pr)"
              >
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7-11-7-11-7z"></path>
                  <circle cx="12" cy="12" r="3"></circle>
                </svg>
                Review
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<style scoped>
.bb-section {
  margin-bottom: 34px;
}

.bb-section-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.bb-section-title {
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
}

.bb-section-desc {
  margin-top: 3px;
  color: var(--subtle);
  font-size: 12px;
}

.bb-filter {
  display: inline-flex;
  padding: 2px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.02);
}

.bb-filter button {
  padding: 5px 11px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease;
}

.bb-filter button:hover {
  color: var(--text);
}

.bb-filter button.is-active {
  background: rgba(255, 255, 255, 0.075);
  color: var(--text);
}

.pr-table-wrap {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: rgba(16, 18, 23, 0.88);
  overflow-x: auto;
}

.pr-table {
  width: 100%;
  border-collapse: collapse;
}

.pr-table thead th {
  padding: 11px 18px;
  border-bottom: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.014);
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  text-align: left;
  white-space: nowrap;
}

.pr-table thead th:last-child {
  text-align: right;
}

.pr-table tbody td {
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
  color: #c8ced5;
  font-size: 12px;
  vertical-align: middle;
}

.pr-table tbody tr:last-child td {
  border-bottom: 0;
}

.pr-table tbody tr {
  transition: background-color 150ms ease;
}

.pr-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.02);
}

.pr-cell-content {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}

.pr-cell-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.pr-cell-id {
  color: var(--subtle);
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.pr-cell-name {
  color: #e9edf1;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 680px;
}

.pr-cell-muted {
  color: var(--subtle);
  font-size: 11px;
  white-space: nowrap;
}

.branch-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 7px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.025);
  color: #a9b1bb;
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  max-width: 260px;
  white-space: nowrap;
}

.branch-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.branch-arrow {
  color: var(--subtle);
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.branch-pill svg {
  width: 10px;
  height: 10px;
  flex-shrink: 0;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 9px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: #a9b1bb;
  font-size: 10px;
  font-weight: 500;
  text-transform: capitalize;
  white-space: nowrap;
}

.status-pill::before {
  content: "";
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.status-pill.is-open {
  color: #b5cbbd;
  border-color: rgba(159, 182, 166, 0.3);
  background: rgba(159, 182, 166, 0.08);
}

.status-pill.is-draft {
  color: #a9b1bb;
  border-color: var(--border);
  background: rgba(255, 255, 255, 0.025);
}

.status-pill.is-needs-work {
  color: #d1b48c;
  border-color: rgba(209, 180, 140, 0.3);
  background: rgba(209, 180, 140, 0.08);
}

.status-pill.is-declined {
  color: #e09999;
  border-color: rgba(224, 153, 153, 0.3);
  background: rgba(224, 153, 153, 0.08);
}

.pr-table td.is-actions {
  text-align: right;
  white-space: nowrap;
}

.review-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 11px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: #c8ced5;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease, color 150ms ease;
}

.review-btn:hover {
  border-color: var(--border-strong);
  background: rgba(255, 255, 255, 0.035);
  color: var(--text);
}

.review-btn svg {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  stroke-width: 1.7;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.empty-state {
  padding: 30px;
  text-align: center;
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  color: var(--subtle);
  font-size: 12px;
}
</style>

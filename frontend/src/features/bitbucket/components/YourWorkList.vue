<script setup lang="ts">
import type { PushBranch } from "../types"

defineProps<{
  pushes: PushBranch[]
  activeFilter: string
  loading: boolean
}>()

const emit = defineEmits<{
  (e: "filter", filter: string): void
  (e: "create-pr", push: PushBranch): void
}>()
</script>

<template>
  <section class="bb-section">
    <div class="bb-section-head">
      <div>
        <div class="bb-section-title">Your work</div>
        <div class="bb-section-desc">Branches you pushed recently. Create a pull request to start review.</div>
      </div>
      <div class="bb-filter" data-bb-filter="pushes">
        <button
          type="button"
          :class="{ 'is-active': activeFilter === 'all' }"
          data-testid="push-filter-all"
          @click="emit('filter', 'all')"
        >
          All
        </button>
        <button
          type="button"
          :class="{ 'is-active': activeFilter === 'ready' }"
          data-testid="push-filter-ready"
          @click="emit('filter', 'ready')"
        >
          Ready
        </button>
        <button
          type="button"
          :class="{ 'is-active': activeFilter === 'stale' }"
          data-testid="push-filter-stale"
          @click="emit('filter', 'stale')"
        >
          Stale
        </button>
      </div>
    </div>

    <div v-if="loading" class="empty-state">
      Loading pushed branches…
    </div>

    <div v-else-if="pushes.length === 0" class="empty-state">
      No pushed branches match this filter.
    </div>

    <div v-else class="push-list">
      <div
        v-for="push in pushes"
        :key="push.id"
        class="push-card"
        :data-testid="`push-card-${push.id}`"
      >
        <div class="push-id">
          <div class="repo-mark">{{ push.repo_mark }}</div>
          <div style="min-width: 0">
            <div class="push-repo">{{ push.repo }}</div>
            <div class="push-branch-row">
              <span class="branch-pill" :title="push.branch">
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <circle cx="6" cy="6" r="2.5"></circle>
                  <circle cx="6" cy="18" r="2.5"></circle>
                  <circle cx="18" cy="12" r="2.5"></circle>
                  <path d="M6 8.5v7"></path>
                  <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
                </svg>
                <span class="branch-text">{{ push.branch }}</span>
              </span>
              <span class="push-sub">{{ push.commit_count }} new commits · {{ push.relative_time }}</span>
            </div>
          </div>
        </div>

        <div class="push-ai">
          <span class="ai-badge">{{ push.ai_badge }}</span>
          <span class="ai-text">
            {{ push.ai_summary }}
          </span>
        </div>

        <div class="push-actions">
          <button
            class="btn btn-primary"
            type="button"
            :data-testid="`btn-create-pr-${push.id}`"
            @click="emit('create-pr', push)"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 5v14"></path>
              <path d="M5 12h14"></path>
            </svg>
            Create pull request
          </button>
        </div>
      </div>
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

.push-list {
  display: grid;
  gap: 10px;
}

.push-card {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr) auto;
  gap: 18px;
  align-items: center;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: rgba(16, 18, 23, 0.88);
  transition: border-color 180ms ease;
}

.push-card:hover {
  border-color: var(--border-strong);
}

.push-id {
  display: flex;
  align-items: center;
  gap: 11px;
  min-width: 0;
}

.repo-mark {
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  display: grid;
  place-items: center;
  border: 1px solid var(--border-strong);
  border-radius: 9px;
  background: var(--surface-raised);
  color: #d8dde4;
  font-size: 11px;
  font-weight: 700;
}

.push-repo {
  color: #e3e7ec;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.push-branch-row {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 6px;
  flex-wrap: wrap;
}

.branch-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 7px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.025);
  color: #a9b1bb;
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  max-width: 240px;
  white-space: nowrap;
}

.branch-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.push-sub {
  color: var(--subtle);
  font-size: 10px;
}

.push-ai {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.018);
}

.ai-badge {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.07);
  color: #cfd6de;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.ai-text {
  color: #b9c0c9;
  font-size: 11px;
  line-height: 1.6;
}

.push-actions {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 13px;
  border-radius: 8px;
  font-size: 11.5px;
  font-weight: 500;
  cursor: pointer;
  transition: all 160ms ease;
  white-space: nowrap;
}

.btn-primary {
  border: 1px solid rgba(255, 255, 255, 0.16);
  background: #f0f3f6;
  color: #0b0c0f;
}

.btn-primary:hover {
  background: #ffffff;
}

.btn-primary svg {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  stroke-width: 2.2;
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

@media (max-width: 1000px) {
  .push-card {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }
  .push-actions {
    grid-column: 1 / -1;
  }
}

@media (max-width: 640px) {
  .push-card {
    grid-template-columns: 1fr;
  }
}
</style>

<script setup lang="ts">
import { onMounted } from "vue"
import PageLayout from "../../../components/layout/PageLayout.vue"
import { useSettings } from "../../settings/composables/useSettings"
import { useJira } from "../composables/useJira"
import KanbanBoard from "../components/KanbanBoard.vue"
import KanbanSkeleton from "../components/KanbanSkeleton.vue"
import BacklogList from "../components/BacklogList.vue"
import BacklogSkeleton from "../components/BacklogSkeleton.vue"
import IssueDetailModal from "../components/IssueDetailModal.vue"

const { secrets, config } = useSettings()

const {
  sprints,
  activeTab,
  currentCategory,
  currentDocType,
  detectedSquad,
  availableSquads,
  selectedSquad,
  columnLimits,
  selectedIssue,
  isLoading,
  error,
  isVpnError,
  isAuthError,
  assignedCount,
  backlogCount,
  categoryCounts,
  boardColumns,
  loadMore,
  openModal,
  closeModal,
  setActiveTab,
  setCategory,
  setDocType,
  setSquad,
  fetchData,
  initialize
} = useJira()

onMounted(() => {
  initialize()
})
</script>

<template>
  <PageLayout>
    <div class="content" data-testid="jira-page">
      <div class="page-header">
        <div class="page-header-text">
          <p class="eyebrow">Sprint workspace</p>
          <h1>Jira BRI</h1>
          <p class="page-copy">
            Everything assigned to you, laid out as a board — filter by type and drag through the workflow. The backlog stays one tab away.
          </p>
        </div>

        <button
          class="btn btn-ghost refresh-btn"
          type="button"
          :disabled="isLoading"
          @click="fetchData"
        >
          <svg
            :class="['refresh-icon', { 'is-spinning': isLoading }]"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M21.5 2v6h-6"></path>
            <path d="M21.34 15.57a10 10 0 1 1-.57-8.38l5.67-5.67"></path>
          </svg>
          <span>{{ isLoading ? "Refreshing..." : "Refresh" }}</span>
        </button>
      </div>

      <div
        v-if="isVpnError"
        class="vpn-banner"
        data-testid="banner-jira-vpn"
      >
        <div class="banner-body">
          <svg viewBox="0 0 24 24" class="banner-icon" aria-hidden="true">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="8" x2="12" y2="12"></line>
            <line x1="12" y1="16" x2="12.01" y2="16"></line>
          </svg>
          <span>Unable to reach Jira BRI API. Please ensure you are connected to the BRI VPN.</span>
        </div>
        <button
          class="btn btn-ghost banner-btn"
          type="button"
          data-testid="btn-retry"
          @click="fetchData"
        >
          Retry
        </button>
      </div>

      <div
        v-else-if="error"
        class="error-banner"
        data-testid="banner-jira-error"
      >
        <div class="banner-body">
          <svg viewBox="0 0 24 24" class="banner-icon" aria-hidden="true">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
          </svg>
          <div class="banner-text-group">
            <span>{{ error }}</span>
            <RouterLink
              v-if="isAuthError"
              to="/settings"
              class="banner-link"
            >
              Update in Settings →
            </RouterLink>
          </div>
        </div>
        <button
          class="btn btn-ghost banner-btn"
          type="button"
          data-testid="btn-retry"
          @click="fetchData"
        >
          Retry
        </button>
      </div>

      <div
        v-else-if="!secrets?.has_jira_pat"
        class="warning-banner"
        data-testid="banner-no-pat"
      >
        <div class="banner-body">
          <svg viewBox="0 0 24 24" class="banner-icon" aria-hidden="true">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
            <line x1="12" y1="9" x2="12" y2="13"></line>
            <line x1="12" y1="17" x2="12.01" y2="17"></line>
          </svg>
          <span>Jira Personal Access Token (PAT) is not configured yet.</span>
        </div>
        <RouterLink to="/settings" class="banner-link">Configure in Settings →</RouterLink>
      </div>

      <template v-if="secrets?.has_jira_pat">
        <div class="jira-tabs" role="tablist">
        <button
          :class="['jira-tab', { 'is-active': activeTab === 'assigned' }]"
          type="button"
          role="tab"
          data-testid="tab-assigned"
          @click="setActiveTab('assigned')"
        >
          <span>Assigned to me</span>
          <span class="count" data-testid="assigned-count">{{ assignedCount }}</span>
        </button>
        <button
          :class="['jira-tab', { 'is-active': activeTab === 'backlog' }]"
          type="button"
          role="tab"
          data-testid="tab-backlog"
          @click="setActiveTab('backlog')"
        >
          <span>Backlog</span>
          <span class="count" data-testid="backlog-count">{{ backlogCount }}</span>
        </button>
      </div>

      <div v-show="activeTab === 'assigned'" class="jira-panel" role="tabpanel">
        <div class="subfilter-bar">
          <div class="subfilter" role="tablist" aria-label="Assigned category">
            <button
              :class="{ 'is-active': currentCategory === 'all' }"
              type="button"
              data-testid="filter-all"
              @click="setCategory('all')"
            >
              <span>All</span>
              <span class="chip">{{ categoryCounts.all }}</span>
            </button>
            <button
              :class="{ 'is-active': currentCategory === 'docs' }"
              type="button"
              data-testid="filter-docs"
              @click="setCategory('docs')"
            >
              <span>DEV Documents</span>
              <span class="chip">{{ categoryCounts.docs }}</span>
            </button>
            <button
              :class="{ 'is-active': currentCategory === 'bugs' }"
              type="button"
              data-testid="filter-bugs"
              @click="setCategory('bugs')"
            >
              <span>Bug / Defect</span>
              <span class="chip">{{ categoryCounts.bugs }}</span>
            </button>
            <button
              :class="{ 'is-active': currentCategory === 'subtasks' }"
              type="button"
              data-testid="filter-subtasks"
              @click="setCategory('subtasks')"
            >
              <span>Subtask</span>
              <span class="chip">{{ categoryCounts.subtasks }}</span>
            </button>
          </div>

          <div
            v-show="currentCategory === 'docs'"
            class="subfilter-secondary"
            role="tablist"
            aria-label="Document type"
          >
            <button
              :class="{ 'is-active': currentDocType === 'all' }"
              type="button"
              data-testid="doc-filter-all"
              @click="setDocType('all')"
            >
              All
            </button>
            <button
              :class="{ 'is-active': currentDocType === 'ut' }"
              type="button"
              data-testid="doc-filter-ut"
              @click="setDocType('ut')"
            >
              UT
            </button>
            <button
              :class="{ 'is-active': currentDocType === 'query' }"
              type="button"
              data-testid="doc-filter-query"
              @click="setDocType('query')"
            >
              Query Review
            </button>
            <button
              :class="{ 'is-active': currentDocType === 'sop' }"
              type="button"
              data-testid="doc-filter-sop"
              @click="setDocType('sop')"
            >
              SOP
            </button>
          </div>
        </div>

        <div v-if="isLoading" class="loading-bar">
          <div class="loading-progress"></div>
        </div>

        <KanbanSkeleton v-if="isLoading" />
        <KanbanBoard
          v-else
          :columns="boardColumns"
          :column-limits="columnLimits"
          @load-more="loadMore"
          @select-issue="openModal"
        />
      </div>

      <div v-show="activeTab === 'backlog'" class="jira-panel" role="tabpanel">
        <div v-if="isLoading" class="loading-bar">
          <div class="loading-progress"></div>
        </div>

        <BacklogSkeleton v-if="isLoading" />
        <BacklogList
          v-else
          :sprints="sprints"
          :available-squads="availableSquads"
          :selected-squad="selectedSquad"
          :detected-squad="detectedSquad"
          @select-squad="setSquad"
          @select-issue="openModal"
        />
      </div>
      </template>

      <IssueDetailModal
        :issue="selectedIssue"
        :jira-base-url="config?.jira_base_url"
        @close="closeModal"
      />
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 1320px;
  margin: 0 auto;
  padding: 36px 30px 52px;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.page-header-text {
  min-width: 0;
}

.eyebrow {
  margin: 0 0 10px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: clamp(30px, 4vw, 46px);
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.05em;
}

.page-copy {
  max-width: 620px;
  margin: 11px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.6;
}

.refresh-btn {
  gap: 8px;
}

.refresh-icon {
  width: 14px;
  height: 14px;
}

.refresh-icon.is-spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.warning-banner,
.error-banner,
.vpn-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-radius: 9px;
  font-size: 13px;
  margin-bottom: 22px;
}

.warning-banner {
  border: 1px solid color-mix(in srgb, var(--warning) 30%, transparent);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
  color: var(--warning);
}

.vpn-banner {
  border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
  color: var(--danger);
}

.error-banner {
  border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
  color: var(--danger);
}

.banner-body {
  display: flex;
  align-items: center;
  gap: 10px;
}

.banner-icon {
  width: 16px;
  height: 16px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  flex-shrink: 0;
}

.banner-text-group {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.banner-link {
  color: var(--text);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.banner-link:hover {
  opacity: 0.9;
}

.banner-btn {
  min-height: 28px;
  padding: 0 10px;
  font-size: 11px;
}

.jira-tabs {
  display: inline-flex;
  gap: 3px;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: 11px;
  background: var(--surface-raised);
  margin-bottom: 20px;
}

.jira-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 8px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: -0.005em;
  white-space: nowrap;
  cursor: pointer;
  background: transparent;
  border: 0;
  transition:
    background-color 160ms ease,
    color 160ms ease,
    box-shadow 160ms ease;
}

.jira-tab:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.jira-tab.is-active {
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.jira-tab .count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  transition:
    background-color 160ms ease,
    color 160ms ease;
}

.jira-tab.is-active .count {
  background: var(--surface-hover);
  color: var(--text);
}

.jira-panel {
  display: flex;
  flex-direction: column;
}

.subfilter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.subfilter {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
}

.subfilter button {
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
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    box-shadow 150ms ease;
}

.subfilter button:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.subfilter button.is-active {
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.subfilter button .chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 16px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 9.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  transition:
    background-color 150ms ease,
    color 150ms ease;
}

.subfilter button.is-active .chip {
  background: var(--surface-hover);
  color: var(--text);
}

.subfilter-secondary {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
  transition: opacity 180ms ease;
}

.subfilter-secondary button {
  padding: 5px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 10.5px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    box-shadow 150ms ease;
}

.subfilter-secondary button:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.subfilter-secondary button.is-active {
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.loading-bar {
  width: 100%;
  height: 2px;
  background: var(--border);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 16px;
}

.loading-progress {
  width: 40%;
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  animation: loading-slide 1.2s ease-in-out infinite;
}

@keyframes loading-slide {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(350%);
  }
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms ease,
    border-color 160ms ease,
    color 160ms ease,
    opacity 160ms ease;
}

.btn-ghost {
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
}

.btn-ghost:hover:not(:disabled) {
  border-color: var(--border-strong);
  background: var(--surface-hover);
  color: var(--text);
}

.btn-ghost:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

@media (max-width: 820px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .content {
    padding: 26px 20px 42px;
  }

  .jira-tabs {
    width: 100%;
    overflow-x: auto;
  }
}
</style>

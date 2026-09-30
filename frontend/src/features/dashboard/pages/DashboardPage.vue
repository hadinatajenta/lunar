<script setup lang="ts">
import { computed, onMounted } from "vue"
import PageLayout from "../../../components/layout/PageLayout.vue"
import { useAuth } from "../../auth/composables/useAuth"
import { useSettings } from "../../settings/composables/useSettings"
import { useToast } from "../../../composables/useToast"
import { useDashboard } from "../composables/useDashboard"
import type { DashboardSummary } from "../types"

const { user } = useAuth()
const { secrets, fetchSettings } = useSettings()
const { showToast } = useToast()
const {
  summary,
  isLoading,
  isSyncing,
  errorMessage,
  lastSyncedAt,
  loadSummary,
  syncWorkspace
} = useDashboard()

const PLACEHOLDER_VALUE = "—"

interface MetricView {
  value: string
  sub: string
  isPlaceholder: boolean
}

onMounted(() => {
  fetchSettings()
  loadSummary()
})

const pendingMetricView = computed<MetricView>(() => ({
  value: PLACEHOLDER_VALUE,
  sub: errorMessage.value ? "Temporarily unavailable" : "Loading workspace data",
  isPlaceholder: true
}))

const buildJiraTicketsMetric = (currentSummary: DashboardSummary | null): MetricView => {
  const jiraSection = currentSummary?.jira
  if (!jiraSection) {
    return pendingMetricView.value
  }
  if (jiraSection.status === "unconfigured") {
    return { value: "0", sub: "Jira PAT not configured", isPlaceholder: false }
  }
  if (jiraSection.status === "error") {
    return { value: PLACEHOLDER_VALUE, sub: "Jira data unavailable", isPlaceholder: true }
  }
  return {
    value: String(jiraSection.assigned_tickets),
    sub: jiraSection.active_sprint || "No active sprint",
    isPlaceholder: false
  }
}

const buildPullRequestsMetric = (currentSummary: DashboardSummary | null): MetricView => {
  const bitbucketSection = currentSummary?.bitbucket
  if (!bitbucketSection) {
    return pendingMetricView.value
  }
  if (bitbucketSection.status === "unconfigured") {
    return { value: "0", sub: "Bitbucket PAT not configured", isPlaceholder: false }
  }
  if (bitbucketSection.status === "error") {
    return { value: PLACEHOLDER_VALUE, sub: "Bitbucket data unavailable", isPlaceholder: true }
  }
  return {
    value: String(bitbucketSection.open_pull_requests),
    sub: `${bitbucketSection.review_requested} review requested`,
    isPlaceholder: false
  }
}

const buildDocumentsMetric = (currentSummary: DashboardSummary | null): MetricView => {
  const jiraSection = currentSummary?.jira
  if (!jiraSection) {
    return pendingMetricView.value
  }
  if (jiraSection.status === "unconfigured") {
    return { value: "0", sub: "Jira PAT not configured", isPlaceholder: false }
  }
  if (jiraSection.status === "error") {
    return { value: PLACEHOLDER_VALUE, sub: "Documents unavailable", isPlaceholder: true }
  }
  return {
    value: String(jiraSection.technical_documents),
    sub: "Jira & Confluence linked specs",
    isPlaceholder: false
  }
}

const buildCopilotToolsMetric = (currentSummary: DashboardSummary | null): MetricView => {
  const copilotSection = currentSummary?.copilot
  if (!copilotSection) {
    return pendingMetricView.value
  }
  if (copilotSection.status === "error") {
    return { value: PLACEHOLDER_VALUE, sub: "Copilot data unavailable", isPlaceholder: true }
  }
  const providerNames = copilotSection.configured_providers
  return {
    value: `${copilotSection.active_tools}/${copilotSection.total_tools}`,
    sub: providerNames.length > 0 ? `Providers: ${providerNames.join(", ")}` : "No providers configured",
    isPlaceholder: false
  }
}

const jiraTicketsMetric = computed(() => buildJiraTicketsMetric(summary.value))
const pullRequestsMetric = computed(() => buildPullRequestsMetric(summary.value))
const documentsMetric = computed(() => buildDocumentsMetric(summary.value))
const copilotToolsMetric = computed(() => buildCopilotToolsMetric(summary.value))

const handleSync = async () => {
  const didSync = await syncWorkspace()
  if (!didSync) {
    showToast(errorMessage.value || "Failed to sync workspace")
  }
}
</script>

<template>
  <PageLayout>
    <div class="content">
      <div class="heading-row">
        <div>
          <p class="eyebrow">Developer workspace</p>
          <h1>Operations Overview</h1>
          <p class="heading-copy">
            Welcome back, {{ user?.full_name || "Lunar Developer" }}. Real-time context across Jira tickets, Bitbucket PR reviews, and AI Copilot.
          </p>
          <p v-if="lastSyncedAt" class="last-synced" data-testid="last-synced">
            Synced {{ lastSyncedAt }}
          </p>
        </div>

        <button
          class="heading-action"
          type="button"
          :disabled="isSyncing"
          data-testid="btn-sync-workspace"
          @click="handleSync"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21.5 2v6h-6M2.5 22v-6h6M2 11.5a10 10 0 0 1 18.8-4.3M22 12.5a10 10 0 0 1-18.8 4.2"></path>
          </svg>
          <span>{{ isSyncing ? "Syncing..." : "Sync workspace" }}</span>
        </button>
      </div>

      <div v-if="errorMessage" class="error-banner" data-testid="banner-dashboard-error">
        <span>{{ errorMessage }}</span>
        <button
          class="banner-retry"
          type="button"
          :disabled="isLoading"
          data-testid="btn-retry"
          @click="loadSummary"
        >
          {{ isLoading ? "Retrying..." : "Retry" }}
        </button>
      </div>

      <div class="metrics-grid">
        <div class="metric-card">
          <div class="metric-label">Jira Assigned Tickets</div>
          <div
            class="metric-value"
            :class="{ 'is-placeholder': jiraTicketsMetric.isPlaceholder }"
            data-testid="metric-jira-tickets"
          >
            {{ jiraTicketsMetric.value }}
          </div>
          <div class="metric-sub">{{ jiraTicketsMetric.sub }}</div>
        </div>

        <div class="metric-card">
          <div class="metric-label">Bitbucket Pull Requests</div>
          <div
            class="metric-value"
            :class="{ 'is-placeholder': pullRequestsMetric.isPlaceholder }"
            data-testid="metric-bitbucket-prs"
          >
            {{ pullRequestsMetric.value }}
          </div>
          <div class="metric-sub">{{ pullRequestsMetric.sub }}</div>
        </div>

        <div class="metric-card">
          <div class="metric-label">Technical Documents</div>
          <div
            class="metric-value"
            :class="{ 'is-placeholder': documentsMetric.isPlaceholder }"
            data-testid="metric-documents"
          >
            {{ documentsMetric.value }}
          </div>
          <div class="metric-sub">{{ documentsMetric.sub }}</div>
        </div>

        <div class="metric-card">
          <div class="metric-label">Active Copilot Tools</div>
          <div
            class="metric-value"
            :class="{ 'is-placeholder': copilotToolsMetric.isPlaceholder }"
            data-testid="metric-copilot-tools"
          >
            {{ copilotToolsMetric.value }}
          </div>
          <div class="metric-sub">{{ copilotToolsMetric.sub }}</div>
        </div>
      </div>

      <div class="dashboard-panels">
        <div class="panel">
          <div class="panel-header">
            <span class="panel-title">Workspace Quick Access</span>
            <span class="panel-meta">Connected Systems</span>
          </div>

          <div class="panel-body">
            <div class="action-tiles">
              <RouterLink to="/copilot" class="action-tile">
                <div class="tile-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M6 8a6 6 0 0 1 12 0v6a4 4 0 0 1-4 4H10a4 4 0 0 1-4-4z"></path>
                    <path d="M9 11h.01"></path>
                    <path d="M15 11h.01"></path>
                    <path d="M9 15c2 1 4 1 6 0"></path>
                  </svg>
                </div>
                <div class="tile-title">AI Copilot</div>
                <div class="tile-desc">Ask multi-turn queries with tool calling across BRI repositories</div>
              </RouterLink>

              <RouterLink to="/jira" class="action-tile">
                <div class="tile-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M7 4h10l-3 4H7z"></path>
                    <path d="M7 20h10l-3-4H7z"></path>
                    <path d="M17 4v6a4 4 0 0 1-4 4H7"></path>
                  </svg>
                </div>
                <div class="tile-title">Jira Tasks & Sprints</div>
                <div class="tile-desc">Inspect active sprint issues, subtask status, and linked tickets</div>
              </RouterLink>

              <RouterLink to="/bitbucket" class="action-tile">
                <div class="tile-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M5 5h14l-2 14H7z"></path>
                    <path d="M9 9h6"></path>
                    <path d="M10 9l1 7"></path>
                  </svg>
                </div>
                <div class="tile-title">Bitbucket & AI Review</div>
                <div class="tile-desc">Review pending PRs with automated AI diff synthesis</div>
              </RouterLink>

              <RouterLink to="/settings" class="action-tile">
                <div class="tile-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="3"></circle>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06-.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                  </svg>
                </div>
                <div class="tile-title">Vault Settings</div>
                <div class="tile-desc">Configure personal Atlassian PATs and AI provider keys</div>
              </RouterLink>
            </div>
          </div>
        </div>

        <div class="panel">
          <div class="panel-header">
            <span class="panel-title">Integration Credentials</span>
            <span class="panel-meta">AES-256-GCM Vault</span>
          </div>

          <div class="panel-body">
            <div class="tool-status-list">
              <div class="tool-item">
                <span class="tool-name">Jira BRI</span>
                <span class="status-badge" :class="secrets?.has_jira_pat ? 'connected' : 'warning'">
                  {{ secrets?.has_jira_pat ? "Configured" : "Needs PAT" }}
                </span>
              </div>
              <div class="tool-item">
                <span class="tool-name">Bitbucket BRI</span>
                <span class="status-badge" :class="secrets?.has_bitbucket_pat ? 'connected' : 'warning'">
                  {{ secrets?.has_bitbucket_pat ? "Configured" : "Needs PAT" }}
                </span>
              </div>
              <div class="tool-item">
                <span class="tool-name">Confluence BRI</span>
                <span class="status-badge" :class="secrets?.has_confluence_pat ? 'connected' : 'warning'">
                  {{ secrets?.has_confluence_pat ? "Configured" : "Needs PAT" }}
                </span>
              </div>
            </div>

            <div class="settings-hint">
              <RouterLink to="/settings" class="settings-link">
                Manage personal credentials in Settings →
              </RouterLink>
            </div>
          </div>
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 1440px;
  margin: 0 auto;
  padding: 34px 30px 48px;
}

.heading-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
}

.eyebrow {
  margin: 0 0 9px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: clamp(28px, 3vw, 38px);
  font-weight: 600;
  line-height: 1.1;
  letter-spacing: -0.04em;
}

.heading-copy {
  margin: 10px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.last-synced {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.error-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 24px;
  padding: 12px 18px;
  border: 1px solid rgba(198, 144, 144, 0.3);
  border-radius: 9px;
  background: rgba(198, 144, 144, 0.1);
  color: var(--danger);
  font-size: 13px;
}

.banner-retry {
  display: inline-flex;
  align-items: center;
  min-height: 36px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--text);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 180ms ease, border-color 180ms ease;
}

.banner-retry:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--border-strong);
}

.banner-retry:focus-visible {
  outline: 2px solid var(--border-strong);
  outline-offset: 2px;
}

.banner-retry:disabled {
  opacity: 0.6;
  cursor: default;
}

.heading-action {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.025);
  color: var(--text);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 180ms ease, border-color 180ms ease;
}

.heading-action:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--border-strong);
}

.heading-action svg {
  width: 14px;
  height: 14px;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-top: 30px;
}

.metric-card {
  padding: 20px 22px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: rgba(16, 18, 23, 0.86);
}

.metric-label {
  color: var(--subtle);
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.metric-value {
  margin-top: 10px;
  color: var(--text);
  font-size: 32px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
}

.metric-value.is-placeholder {
  color: var(--muted);
}

.metric-sub {
  margin-top: 8px;
  color: var(--muted);
  font-size: 12px;
}

.dashboard-panels {
  display: grid;
  grid-template-columns: 2fr 1.2fr;
  gap: 20px;
  margin-top: 24px;
}

.panel {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: rgba(16, 18, 23, 0.86);
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.panel-title {
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
}

.panel-meta {
  color: var(--subtle);
  font-size: 11px;
}

.panel-body {
  padding: 20px;
}

.action-tiles {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.action-tile {
  display: flex;
  flex-direction: column;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.015);
  transition: background-color 180ms ease, border-color 180ms ease, transform 180ms ease;
}

.action-tile:hover {
  background: rgba(255, 255, 255, 0.035);
  border-color: var(--border-strong);
  transform: translateY(-1px);
}

.tile-icon {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  color: var(--text);
  margin-bottom: 12px;
}

.tile-icon svg {
  width: 16px;
  height: 16px;
}

.tile-title {
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
}

.tile-desc {
  margin-top: 4px;
  color: var(--subtle);
  font-size: 11px;
  line-height: 1.4;
}

.tool-status-list {
  display: grid;
  gap: 10px;
}

.tool-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.015);
}

.tool-name {
  color: var(--text);
  font-size: 13px;
  font-weight: 500;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 5px;
  font-size: 11px;
  font-weight: 500;
}

.status-badge.connected {
  background: rgba(159, 182, 166, 0.12);
  color: var(--positive);
  border: 1px solid rgba(159, 182, 166, 0.25);
}

.status-badge.warning {
  background: rgba(187, 169, 132, 0.12);
  color: var(--warning);
  border: 1px solid rgba(187, 169, 132, 0.25);
}

.settings-hint {
  margin-top: 18px;
  text-align: center;
}

.settings-link {
  color: var(--muted);
  font-size: 12px;
  text-decoration: underline;
  text-underline-offset: 3px;
  transition: color 180ms ease;
}

.settings-link:hover {
  color: var(--text);
}

@media (max-width: 980px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .dashboard-panels {
    grid-template-columns: 1fr;
  }
}
</style>

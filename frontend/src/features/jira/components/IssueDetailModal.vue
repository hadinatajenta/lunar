<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from "vue"
import type { JiraIssue } from "../types"

interface Props {
  issue: JiraIssue | null
  jiraBaseUrl?: string
}

const props = withDefaults(defineProps<Props>(), {
  jiraBaseUrl: "https://jira.bri.co.id"
})

const emit = defineEmits<{
  (e: "close"): void
}>()

const isDescriptionExpanded = ref(false)

watch(
  () => props.issue,
  () => {
    isDescriptionExpanded.value = false
  }
)

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Escape" && props.issue) {
    emit("close")
  }
}

onMounted(() => {
  window.addEventListener("keydown", handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeydown)
})

const typeLabel = computed(() => {
  if (!props.issue) return ""
  if (props.issue.label) return props.issue.label
  return (props.issue.kind || "TASK").toUpperCase()
})

const statusText = computed(() => {
  if (!props.issue) return ""
  const status = props.issue.status.toLowerCase()
  if (status === "progress" || status.includes("progress")) {
    return "In Progress"
  }
  if (status === "done" || status.includes("done")) {
    return "Done"
  }
  return "Open"
})

const priorityText = computed(() => {
  if (!props.issue) return ""
  const priority = (props.issue.priority || "medium").toLowerCase()
  if (priority.includes("high")) return "High"
  if (priority.includes("low")) return "Low"
  return "Medium"
})

const priorityClass = computed(() => {
  const text = priorityText.value.toLowerCase()
  if (text === "high") return "is-high"
  if (text === "low") return "is-low"
  return "is-medium"
})

const assigneeName = computed(() => {
  if (!props.issue) return "Unassigned"
  if (typeof props.issue.assignee === "string" && props.issue.assignee.length > 0) {
    return props.issue.assignee
  }
  if (props.issue.assignee && typeof props.issue.assignee === "object") {
    return (
      props.issue.assignee.display_name ||
      props.issue.assignee.name ||
      props.issue.assignee.initials ||
      "Unassigned"
    )
  }
  return "Unassigned"
})

const displayedDescription = computed(() => {
  if (!props.issue) return ""
  return props.issue.description || props.issue.title || "No description provided."
})

const isExpandable = computed(() => displayedDescription.value.length > 80)

const jiraUrl = computed(() => {
  if (!props.issue) return "#"
  const issueKey = props.issue.key || props.issue.id
  const base = (props.jiraBaseUrl || "https://jira.bri.co.id").replace(/\/+$/, "")
  return `${base}/browse/${encodeURIComponent(issueKey)}`
})

const openInJira = () => {
  if (props.issue) {
    window.open(jiraUrl.value, "_blank", "noopener,noreferrer")
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="issue" class="modal-overlay">
      <div
        class="modal-scrim"
        data-testid="modal-scrim"
        @click="emit('close')"
      ></div>

      <div
        class="modal-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-issue-title"
        data-testid="issue-detail-modal"
      >
        <div class="modal-header">
          <div class="header-left">
            <div class="modal-eyebrow" data-testid="modal-issue-key">
              {{ issue.key || issue.id }}
            </div>
            <h3 id="modal-issue-title" class="modal-title" data-testid="modal-issue-title">
              {{ issue.title }}
            </h3>
          </div>
          <button
            class="modal-close"
            type="button"
            data-testid="btn-close-modal"
            aria-label="Close dialog"
            @click="emit('close')"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M6 6l12 12"></path>
              <path d="M18 6 6 18"></path>
            </svg>
          </button>
        </div>

        <div class="modal-body">
          <div class="meta-row">
            <div class="meta-badge" data-testid="modal-issue-type">
              <span class="meta-label">Type</span>
              <strong class="meta-value">{{ typeLabel }}</strong>
            </div>

            <div class="meta-badge" data-testid="modal-issue-status">
              <span class="meta-label">Status</span>
              <strong class="meta-value">{{ statusText }}</strong>
            </div>

            <div class="meta-badge" data-testid="modal-issue-priority">
              <span class="meta-label">Priority</span>
              <div class="priority-wrapper">
                <span :class="['priority-dot', priorityClass]"></span>
                <strong class="meta-value">{{ priorityText }}</strong>
              </div>
            </div>

            <div class="meta-badge" data-testid="modal-issue-points">
              <span class="meta-label">Points</span>
              <strong class="meta-value">{{ issue.points }} pts</strong>
            </div>

            <div class="meta-badge" data-testid="modal-issue-assignee">
              <span class="meta-label">Assignee</span>
              <strong class="meta-value">{{ assigneeName }}</strong>
            </div>
          </div>

          <div class="description-section">
            <p class="section-label">Description</p>
            <div class="description-body" data-testid="modal-issue-desc">
              <p :class="['description-text', { 'is-clamped': !isDescriptionExpanded && isExpandable }]">
                {{ displayedDescription }}
              </p>
              <button
                v-if="isExpandable"
                class="btn-read-more"
                type="button"
                data-testid="btn-read-more"
                @click="isDescriptionExpanded = !isDescriptionExpanded"
              >
                {{ isDescriptionExpanded ? 'Read less' : 'Read more' }}
              </button>
            </div>
          </div>

          <div v-if="issue.generated" class="ai-output">
            <div class="ai-output-head">
              <span class="ai-badge">AI</span>
              Generated Documentation
            </div>
            <pre>{{ issue.generated }}</pre>
          </div>
        </div>

        <div class="modal-footer">
          <button
            class="btn btn-ghost"
            type="button"
            data-testid="btn-open-jira"
            @click="openInJira"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M7 17 17 7"></path>
              <path d="M8 7h9v9"></path>
            </svg>
            Open in Jira
          </button>
          <div class="spacer"></div>
          <button
            class="btn btn-primary"
            type="button"
            @click="emit('close')"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal-scrim {
  position: fixed;
  inset: 0;
  background: rgba(5, 6, 8, 0.72);
  backdrop-filter: blur(4px);
}

.modal-dialog {
  position: relative;
  z-index: 51;
  width: min(640px, 94vw);
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-strong);
  border-radius: 14px;
  background: var(--surface);
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.65);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px 18px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.header-left {
  min-width: 0;
}

.modal-eyebrow {
  display: inline-flex;
  align-items: center;
  margin-bottom: 6px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.modal-title {
  margin: 0;
  color: var(--text);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: -0.02em;
}

.modal-close {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--muted);
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    border-color 150ms ease;
}

.modal-close:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: var(--border-strong);
  color: var(--text);
}

.modal-close svg {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.modal-body::-webkit-scrollbar {
  width: 6px;
}

.modal-body::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 3px;
}

.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border);
}

.meta-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: rgba(255, 255, 255, 0.025);
  font-size: 11px;
}

.meta-label {
  color: var(--subtle);
  font-weight: 500;
}

.meta-value {
  color: var(--text);
  font-weight: 600;
}

.priority-wrapper {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.priority-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.priority-dot.is-high {
  background: #c69090;
  box-shadow: 0 0 8px rgba(198, 144, 144, 0.5);
}

.priority-dot.is-medium {
  background: #bba984;
}

.priority-dot.is-low {
  background: #9fb6a6;
}

.description-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-label {
  margin: 0;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.description-body {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.description-text {
  margin: 0;
  color: #bfc6cf;
  font-size: 13px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
}

.description-text.is-clamped {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.btn-read-more {
  padding: 2px 0;
  color: var(--accent);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 3px;
  background: transparent;
  border: 0;
}

.btn-read-more:hover {
  opacity: 0.85;
}

.ai-output {
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.018);
}

.ai-output-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-badge {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.07);
  color: #cfd6de;
  font-size: 9px;
  font-weight: 700;
}

.ai-output pre {
  margin: 0;
  padding: 12px 14px;
  border-radius: 8px;
  background: rgba(0, 0, 0, 0.3);
  color: #c8ced5;
  font-size: 11.5px;
  line-height: 1.65;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 280px;
  overflow: auto;
}

.modal-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 24px;
  border-top: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.015);
  flex-shrink: 0;
}

.spacer {
  flex: 1;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 14px;
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

.btn svg {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.btn-primary {
  background: var(--accent);
  color: #0b0c0f;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-ghost {
  border-color: var(--border);
  background: transparent;
  color: #c8ced5;
}

.btn-ghost:hover {
  border-color: var(--border-strong);
  background: rgba(255, 255, 255, 0.03);
}
</style>

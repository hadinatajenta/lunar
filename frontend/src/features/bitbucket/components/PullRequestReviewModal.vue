<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from "vue"
import type { PullRequest, PRDiff, AICodeReview } from "../types"
import { useCopilot } from "../../copilot/composables/useCopilot"

const props = defineProps<{
  pr: PullRequest | null
  diff: PRDiff | null
  aiReview: AICodeReview | null
  loadingDiff: boolean
  loadingAI: boolean
  postingComment: boolean
}>()

const emit = defineEmits<{
  (e: "close"): void
  (e: "trigger-ai", payload: { pr: PullRequest; model: string } | PullRequest): void
  (e: "send-comment", payload: { pr: PullRequest; comment: string }): void
  (e: "review-action", payload: { pr: PullRequest; action: string }): void
}>()

const commentText = ref("")
const showDiffViewer = ref(false)

const { configuredModels, hasConfiguredAI, refreshConfiguredProviders } = useCopilot()
const reviewModel = ref("DeepSeek-V4 Pro (Thinking)")

watch(
  configuredModels,
  (models) => {
    if (models.length > 0 && !models.some((m) => m.name === reviewModel.value)) {
      reviewModel.value = models[0].name
    }
  },
  { immediate: true }
)

watch(
  () => props.pr,
  (newPr) => {
    if (newPr) {
      commentText.value = ""
      showDiffViewer.value = false
    }
  },
  { immediate: true }
)

watch(
  () => props.aiReview,
  (review) => {
    if (review && review.generated_comment) {
      commentText.value = review.generated_comment
    }
  }
)

const handleSendComment = () => {
  if (!props.pr || !commentText.value.trim()) return
  emit("send-comment", {
    pr: props.pr,
    comment: commentText.value.trim(),
  })
}

const handleAction = (action: string) => {
  if (!props.pr) return
  emit("review-action", {
    pr: props.pr,
    action,
  })
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Escape") {
    emit("close")
  }
}

onMounted(async () => {
  window.addEventListener("keydown", handleKeydown)
  await refreshConfiguredProviders()
})

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <div
    v-if="pr"
    class="modal-overlay"
    data-testid="review-modal-overlay"
    @click.self="emit('close')"
  >
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="review-modal-title"
      data-testid="review-modal"
    >
      <div class="modal-header">
        <div style="min-width: 0">
          <div class="modal-eyebrow" data-testid="review-modal-id">Pull request {{ pr.id }}</div>
          <h3 id="review-modal-title" class="modal-title" data-testid="review-modal-title">{{ pr.title }}</h3>
        </div>
        <button
          class="modal-close"
          type="button"
          aria-label="Close review"
          data-testid="btn-close-review"
          @click="emit('close')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12"></path>
            <path d="M18 6 6 18"></path>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <div class="modal-meta">
          <span class="modal-meta-item">Repo <strong data-testid="review-modal-repo">{{ pr.repo }}</strong></span>
          <span class="modal-meta-item">Branch <strong data-testid="review-modal-branch">{{ pr.source_branch }} → {{ pr.target_branch }}</strong></span>
          <span class="modal-meta-item">Author <strong data-testid="review-modal-author">{{ pr.author }}</strong></span>
          <span class="modal-meta-item" data-testid="review-modal-files">{{ pr.files_count }} files · +{{ pr.lines_added }} −{{ pr.lines_deleted }}</span>
        </div>

        <div class="diff-toggle-bar">
          <button
            class="diff-toggle-btn"
            type="button"
            data-testid="btn-toggle-diff"
            @click="showDiffViewer = !showDiffViewer"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
              <polyline points="14 2 14 8 20 8"></polyline>
              <line x1="16" y1="13" x2="8" y2="13"></line>
              <line x1="16" y1="17" x2="8" y2="17"></line>
            </svg>
            {{ showDiffViewer ? "Hide Git Diff" : "Inspect Git Diff" }}
          </button>
        </div>

        <div v-if="showDiffViewer" class="diff-viewer" data-testid="diff-viewer">
          <div v-if="loadingDiff" class="diff-loading">Loading diff hunks…</div>
          <div v-else-if="diff && diff.files.length > 0">
            <div v-for="file in diff.files" :key="file.new_path" class="diff-file">
              <div class="diff-file-head">
                <span class="file-path">{{ file.new_path }}</span>
                <span class="file-stats">+{{ file.additions }} −{{ file.deletions }}</span>
              </div>
              <div v-for="(hunk, hIdx) in file.hunks" :key="hIdx" class="diff-hunk">
                <div class="hunk-header">{{ hunk.header }}</div>
                <div
                  v-for="(line, lIdx) in hunk.lines"
                  :key="lIdx"
                  class="diff-line"
                  :class="{
                    'line-add': line.startsWith('+'),
                    'line-del': line.startsWith('-'),
                  }"
                >
                  {{ line }}
                </div>
              </div>
            </div>
          </div>
          <div v-else class="diff-empty">No diff changes detected.</div>
        </div>

        <div class="ai-panel" data-testid="ai-panel">
          <div class="ai-panel-head">
            <div class="ai-panel-title-group">
              <span class="ai-badge">AI</span>
              <span>Lunar review</span>
            </div>

            <div v-if="hasConfiguredAI && configuredModels.length > 0" class="pr-model-picker">
              <label for="pr-review-model-select" class="pr-model-label">Model:</label>
              <select
                id="pr-review-model-select"
                v-model="reviewModel"
                class="pr-model-select"
                data-testid="pr-review-model-select"
              >
                <option v-for="m in configuredModels" :key="m.name" :value="m.name">
                  {{ m.name }}
                </option>
              </select>
            </div>
          </div>

          <div v-if="!hasConfiguredAI" class="ai-empty" data-testid="banner-no-ai-keys">
            <span>No AI provider configured yet. Please configure your API key in Settings to unlock AI Code Review.</span>
            <RouterLink to="/settings" class="settings-link" style="margin-left: 6px;">Configure in Settings →</RouterLink>
          </div>

          <div v-else-if="!aiReview && !loadingAI" class="ai-empty" data-testid="ai-empty">
            Click “Review with AI” to let Lunar Copilot analyze the diff, tests, and linked issues using {{ reviewModel }}. The generated review will appear in the comment box below, ready to send.
          </div>

          <div v-if="loadingAI" class="ai-loading" data-testid="ai-loading">
            <span class="ai-spinner" aria-hidden="true"></span>
            <span>Analyzing the diff, tests, and linked issues…</span>
          </div>

          <div v-if="aiReview && !loadingAI" class="ai-findings" data-testid="ai-findings">
            <p class="ai-summary-text" data-testid="ai-summary">{{ aiReview.summary }}</p>
            <div
              v-for="(finding, idx) in aiReview.findings"
              :key="idx"
              class="ai-finding"
              :data-testid="`ai-finding-${idx}`"
            >
              {{ finding }}
            </div>
          </div>
        </div>

        <div class="comment-label">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
          </svg>
          Review comment
        </div>
        <textarea
          v-model="commentText"
          class="comment-input"
          placeholder="Click “Review with AI” to generate a review, or write your own comment here."
          data-testid="input-review-comment"
        ></textarea>
      </div>

      <div class="modal-footer">
        <button
          class="btn btn-warning"
          type="button"
          data-testid="btn-action-needs-work"
          @click="handleAction('needs-work')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 8v5"></path>
            <path d="M12 16h.01"></path>
            <circle cx="12" cy="12" r="9"></circle>
          </svg>
          Needs work
        </button>

        <button
          class="btn btn-success"
          type="button"
          data-testid="btn-action-approve"
          @click="handleAction('approve')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="m5 13 4 4L19 7"></path>
          </svg>
          Approve
        </button>

        <button
          class="btn btn-danger"
          type="button"
          data-testid="btn-action-decline"
          @click="handleAction('decline')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12"></path>
            <path d="M18 6 6 18"></path>
          </svg>
          Decline
        </button>

        <div class="spacer"></div>

        <button
          class="btn btn-ghost"
          type="button"
          :disabled="loadingAI || !hasConfiguredAI"
          data-testid="btn-trigger-ai-review"
          @click="emit('trigger-ai', { pr, model: reviewModel })"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 3v3"></path>
            <path d="M12 18v3"></path>
            <path d="m5.6 5.6 2.1 2.1"></path>
            <path d="m16.3 16.3 2.1 2.1"></path>
            <path d="M3 12h3"></path>
            <path d="M18 12h3"></path>
            <path d="m5.6 18.4 2.1-2.1"></path>
            <path d="m16.3 7.7 2.1-2.1"></path>
            <circle cx="12" cy="12" r="3.5"></circle>
          </svg>
          {{ loadingAI ? "Analyzing…" : "Review with AI" }}
        </button>

        <button
          class="btn btn-primary"
          type="button"
          :disabled="postingComment || !commentText.trim()"
          data-testid="btn-send-comment"
          @click="handleSendComment"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M22 2 11 13"></path>
            <path d="M22 2 15 22l-4-9-9-4z"></path>
          </svg>
          {{ postingComment ? "Posting…" : "Send comment" }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 24px;
  background: rgba(5, 6, 8, 0.72);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.modal {
  width: 100%;
  max-width: 680px;
  max-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.35);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}

.modal-eyebrow {
  margin-bottom: 6px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.modal-title {
  margin: 0;
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.35;
}

.modal-close {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex: 0 0 auto;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease, border-color 150ms ease;
}

.modal-close:hover {
  background: var(--surface-hover);
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
  padding: 18px 20px;
}

.modal-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
  padding-bottom: 14px;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--border);
  color: var(--subtle);
  font-size: 11px;
}

.modal-meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.modal-meta-item strong {
  color: var(--text);
  font-weight: 500;
}

.diff-toggle-bar {
  margin-bottom: 12px;
}

.diff-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border: 1px solid var(--border);
  border-radius: 7px;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 11px;
  cursor: pointer;
  transition: background-color 150ms ease, border-color 150ms ease;
}

.diff-toggle-btn:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
}

.diff-toggle-btn svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
}

.diff-viewer {
  margin-bottom: 14px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  overflow: hidden;
}

.diff-file-head {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--surface-hover);
  border-bottom: 1px solid var(--border);
}

.file-path {
  color: var(--text);
  font-weight: 500;
}

.file-stats {
  color: var(--subtle);
}

.diff-hunk {
  padding: 4px 0;
}

.hunk-header {
  padding: 4px 12px;
  color: var(--muted);
  background: var(--surface);
  font-size: 10px;
}

.diff-line {
  padding: 2px 12px;
  white-space: pre-wrap;
  color: var(--text);
}

.diff-line.line-add {
  background: color-mix(in srgb, var(--positive) 14%, transparent);
  color: var(--positive);
}

.diff-line.line-del {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.diff-loading,
.diff-empty {
  padding: 14px;
  color: var(--subtle);
  text-align: center;
}

.ai-panel {
  margin-bottom: 14px;
  padding: 13px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
}

.ai-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-panel-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pr-model-picker {
  display: flex;
  align-items: center;
  gap: 6px;
}

.pr-model-label {
  font-size: 10px;
  color: var(--subtle);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pr-model-select {
  background: var(--surface-raised);
  border: 1px solid var(--border);
  color: var(--text);
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 6px;
  outline: none;
  cursor: pointer;
}

.pr-model-select:focus {
  border-color: var(--border-strong);
}

.settings-link {
  color: var(--text);
  text-decoration: underline;
  font-weight: 500;
}

.settings-link:hover {
  opacity: 0.85;
}

.ai-badge {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface-hover);
  color: var(--text);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.ai-empty {
  margin-top: 9px;
  color: var(--subtle);
  font-size: 11px;
  line-height: 1.55;
}

.ai-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 12px;
  color: var(--muted);
  font-size: 11px;
}

.ai-spinner {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  border: 2px solid color-mix(in srgb, var(--text) 20%, transparent);
  border-top-color: var(--text);
  border-radius: 50%;
  animation: spin 700ms linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.ai-findings {
  margin: 12px 0 0;
  display: grid;
  gap: 7px;
  background: var(--surface-raised);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 10px 12px;
}

.ai-summary-text {
  margin: 0 0 6px;
  color: var(--text);
  font-size: 11.5px;
  line-height: 1.65;
}

.ai-finding {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: var(--text);
  font-size: 11px;
  line-height: 1.55;
}

.ai-finding::before {
  content: "";
  width: 4px;
  height: 4px;
  flex: 0 0 auto;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--muted);
}

.comment-label {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 8px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.comment-label svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.comment-input {
  width: 100%;
  min-height: 130px;
  max-height: 240px;
  resize: vertical;
  padding: 12px 13px;
  border: 1px solid var(--border);
  border-radius: 9px;
  outline: 0;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 11.5px;
  line-height: 1.65;
  font-family: inherit;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.comment-input:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--text) 5%, transparent);
}

.comment-input::placeholder {
  color: var(--subtle);
}

.modal-footer {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  background: var(--surface);
  flex: 0 0 auto;
}

.modal-footer .spacer {
  flex: 1;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: all 160ms ease;
  white-space: nowrap;
}

.btn-warning {
  border: 1px solid color-mix(in srgb, var(--warning) 30%, transparent);
  background: color-mix(in srgb, var(--warning) 12%, transparent);
  color: var(--warning);
}

.btn-warning:hover {
  background: color-mix(in srgb, var(--warning) 20%, transparent);
}

.btn-success {
  border: 1px solid color-mix(in srgb, var(--positive) 30%, transparent);
  background: color-mix(in srgb, var(--positive) 12%, transparent);
  color: var(--positive);
}

.btn-success:hover {
  background: color-mix(in srgb, var(--positive) 20%, transparent);
}

.btn-danger {
  border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
}

.btn-danger:hover {
  background: color-mix(in srgb, var(--danger) 20%, transparent);
}

.btn-ghost {
  border: 1px solid var(--border);
  background: var(--surface-raised);
  color: var(--text);
}

.btn-ghost:hover:not(:disabled) {
  border-color: var(--border-strong);
  background: var(--surface-hover);
}

.btn-ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  border: 1px solid transparent;
  background: var(--accent);
  color: var(--accent-contrast);
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn svg {
  width: 12px;
  height: 12px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>

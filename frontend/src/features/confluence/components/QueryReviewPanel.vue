<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import type { ConfluenceDocument, QueryReviewRepository } from "../types"
import { useCopilot } from "../../copilot/composables/useCopilot"
import ModelPicker from "../../copilot/components/ModelPicker.vue"
import { useToast } from "../../../composables/useToast"
import { useQueryReview } from "../composables/query-review-store"
import {
  QUERY_REVIEW_EMPTY_MESSAGE,
  QUERY_REVIEW_LOADING_MESSAGE
} from "../composables/useQueryReview"
import { buildCaptureTableHtml, buildCaptureTableText } from "../utils/capture-table"
import QueryReviewRepositoryCard from "./QueryReviewRepositoryCard.vue"

const props = defineProps<{
  document: ConfluenceDocument
  model: string
}>()

const emit = defineEmits<{
  (event: "update:model", value: string): void
}>()

const { hasConfiguredAI, isCheckingProviders } = useCopilot()
const { showToast } = useToast()
const { status, result, errorMessage, errorCode, isLoading, retry, retryWithSource, cancel, apply, applyError, isApplying } =
  useQueryReview()

const panelHeading = ref<HTMLElement | null>(null)
const manualCompareUrl = ref("")

const repositories = computed(() => result.value?.repositories ?? [])
const skippedRepositories = computed(() => result.value?.skipped_repositories ?? [])

const copyText = async (text: string, successMessage: string): Promise<void> => {
  if (!navigator.clipboard?.writeText) {
    showToast("Copy failed: clipboard unavailable.")
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    showToast(successMessage)
  } catch {
    showToast("Copy failed: clipboard unavailable.")
  }
}

const writeCaptureTable = async (html: string, text: string): Promise<boolean> => {
  if (typeof ClipboardItem === "undefined" || !navigator.clipboard?.write) {
    return false
  }
  try {
    const item = new ClipboardItem({
      "text/html": new Blob([html], { type: "text/html" }),
      "text/plain": new Blob([text], { type: "text/plain" })
    })
    await navigator.clipboard.write([item])
    return true
  } catch {
    return false
  }
}

const copyCaptureTable = async (
  selectedRepositories: QueryReviewRepository[],
  successMessage: string
): Promise<void> => {
  const html = buildCaptureTableHtml(selectedRepositories)
  const text = buildCaptureTableText(selectedRepositories)
  if (await writeCaptureTable(html, text)) {
    showToast(successMessage)
    return
  }
  await copyText(text, successMessage)
}

const copyAllTables = (): Promise<void> =>
  copyCaptureTable(repositories.value, "Query table copied for Confluence.")

const copyRepositoryTable = (repository: QueryReviewRepository): Promise<void> =>
  copyCaptureTable([repository], "Query table copied for Confluence.")

const copyRawJson = (): Promise<void> =>
  copyText(JSON.stringify(result.value, null, 2), "Raw query review copied as JSON.")

const retryWithManualUrl = (): Promise<void> => {
  const compareUrl = manualCompareUrl.value.trim()
  if (compareUrl.length === 0) {
    return Promise.resolve()
  }
  return retryWithSource({ compare_url: compareUrl })
}

watch(status, async (nextStatus) => {
  if (nextStatus === "success" || nextStatus === "empty" || nextStatus === "error") {
    await nextTick()
    panelHeading.value?.focus()
  }
})

watch(
  () => props.document.id,
  () => {
    manualCompareUrl.value = ""
  }
)
</script>

<template>
  <div class="detail-section">
    <h3
      ref="panelHeading"
      class="ai-panel-heading"
      tabindex="-1"
      data-testid="ai-query-heading"
    >
      Query review
    </h3>

    <div class="ai-output">
      <div class="ai-output-head">
        <span class="ai-badge">AI</span>
        Lunar Copilot
        <ModelPicker
          v-if="hasConfiguredAI"
          class="ai-head-picker"
          :model-value="props.model"
          :show-effort="false"
          trigger-test-id="query-review-model-trigger"
          @update:model-value="emit('update:model', $event)"
        />
      </div>

      <div v-if="isCheckingProviders" class="ai-state" data-testid="ai-query-checking">
        <span class="ai-spinner" aria-hidden="true"></span>
        <span>Checking the configured AI providers…</span>
      </div>

      <template v-else>
        <div
          v-if="!hasConfiguredAI && status === 'idle'"
          class="ai-notice is-warning"
          data-testid="banner-no-ai-keys"
        >
          <span>No AI provider is configured yet. Add an API key in Settings to extract the query review.</span>
          <RouterLink to="/settings" class="ai-notice-link">Configure in Settings →</RouterLink>
        </div>

        <div v-if="isLoading" class="ai-loading" aria-live="polite" data-testid="ai-query-loading">
          <div class="ai-skeleton" aria-hidden="true">
            <span></span>
            <span></span>
            <span></span>
          </div>
          <p class="ai-state-text">{{ QUERY_REVIEW_LOADING_MESSAGE }}</p>
          <button class="ai-button" type="button" data-testid="ai-query-cancel" @click="cancel">
            Cancel
          </button>
        </div>

        <div
          v-else-if="status === 'error'"
          class="ai-notice is-danger"
          aria-live="polite"
          data-testid="ai-query-error"
        >
          <p class="ai-notice-text">{{ errorMessage }}</p>

          <div v-if="errorCode" class="ai-manual-source">
            <label class="ai-field-label" for="query-review-manual-url">Compare URL</label>
            <div class="ai-manual-row">
              <input
                id="query-review-manual-url"
                v-model="manualCompareUrl"
                class="ai-input"
                type="url"
                placeholder="https://bitbucket.../compare/diff?sourceBranch=...&targetBranch=..."
                data-testid="ai-query-manual-url"
              />
              <button
                class="ai-button is-primary"
                type="button"
                :disabled="manualCompareUrl.trim().length === 0"
                data-testid="ai-query-retry"
                @click="retryWithManualUrl"
              >
                Retry with this URL
              </button>
            </div>
          </div>

          <button
            v-else
            class="ai-button"
            type="button"
            data-testid="ai-query-retry"
            @click="retry"
          >
            Retry
          </button>
        </div>

        <div
          v-else-if="status === 'empty'"
          class="ai-empty-result"
          aria-live="polite"
          data-testid="ai-query-empty"
        >
          {{ QUERY_REVIEW_EMPTY_MESSAGE }}
        </div>

        <div
          v-else-if="status === 'success' && result"
          class="ai-result"
          aria-live="polite"
          data-testid="ai-query-result"
        >
          <p v-if="result.mig_page_title" class="ai-mig" data-testid="ai-query-mig-page">
            MIG page <strong>{{ result.mig_page_title }}</strong>
          </p>

          <p
            v-if="skippedRepositories.length > 0"
            class="ai-notice is-warning"
            data-testid="ai-query-skipped-repositories"
          >
            Excluded from this review:
            <span v-for="(skipped, index) in skippedRepositories" :key="skipped.repository">
              <span v-if="index > 0">, </span>{{ skipped.repository }} ({{ skipped.reason }})
            </span>
          </p>

          <div class="ai-result-actions">
            <button
              class="ai-button"
              type="button"
              data-testid="ai-query-copy-table"
              @click="copyAllTables"
            >
              Copy table
            </button>
            <button
              class="ai-button"
              type="button"
              data-testid="ai-query-copy-json"
              @click="copyRawJson"
            >
              Copy raw JSON
            </button>
          </div>

          <div class="ai-repositories" data-testid="ai-query-repositories">
            <QueryReviewRepositoryCard
              v-for="repository in repositories"
              :key="repository.repository"
              :repository="repository"
              @copy-table="copyRepositoryTable"
            />
          </div>

          <div
            v-if="applyError"
            class="ai-notice is-danger"
            aria-live="polite"
            data-testid="ai-query-apply-error"
          >
            <p class="ai-notice-text">{{ applyError }}</p>
            <button
              class="ai-button"
              type="button"
              :disabled="isApplying"
              data-testid="ai-query-apply-retry"
              @click="apply"
            >
              Retry apply
            </button>
          </div>
        </div>

        <div v-else class="ai-state" data-testid="ai-query-idle">
          Choose a model, then run <strong>Extract query review</strong>. The queries introduced on this
          branch will appear here.
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.detail-section {
  margin-bottom: 24px;
}

.ai-panel-heading {
  margin: 0 0 10px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.ai-panel-heading:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.ai-output {
  padding: 16px 18px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-raised);
}

.ai-output-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-head-picker {
  margin-left: auto;
  text-transform: none;
  letter-spacing: 0;
}

.ai-badge {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  border-radius: 6px;
  background: var(--surface-hover);
  color: var(--muted);
  font-size: 9px;
  font-weight: 700;
}

.ai-state,
.ai-loading {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  color: var(--subtle);
  font-size: 12px;
  line-height: 1.6;
}

.ai-state-text {
  margin: 0;
}

.ai-spinner {
  width: 13px;
  height: 13px;
  border: 2px solid var(--border-strong);
  border-top-color: var(--muted);
  border-radius: 50%;
  animation: ai-spin 700ms linear infinite;
}

@keyframes ai-spin {
  to {
    transform: rotate(360deg);
  }
}

.ai-skeleton {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.ai-skeleton span {
  height: 12px;
  border-radius: 6px;
  background: var(--surface-hover);
  animation: ai-pulse 1.1s ease-in-out infinite alternate;
}

.ai-skeleton span:nth-child(2) {
  width: 82%;
}

.ai-skeleton span:nth-child(3) {
  width: 64%;
}

@keyframes ai-pulse {
  from {
    opacity: 0.45;
  }
  to {
    opacity: 0.9;
  }
}

.ai-notice {
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}

.ai-notice.is-warning {
  border-color: color-mix(in srgb, var(--warning) 45%, transparent);
  color: var(--warning);
}

.ai-notice.is-danger {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  color: var(--danger);
}

.ai-notice-text {
  margin: 0 0 10px;
}

.ai-notice-link {
  color: inherit;
  text-decoration: underline;
  white-space: nowrap;
}

.ai-field-label {
  margin: 0 0 6px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-manual-source {
  margin-top: 12px;
}

.ai-manual-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.ai-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text);
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms cubic-bezier(0.2, 0.7, 0.3, 1),
    border-color 160ms cubic-bezier(0.2, 0.7, 0.3, 1),
    opacity 160ms cubic-bezier(0.2, 0.7, 0.3, 1);
}

.ai-button:hover:not(:disabled) {
  border-color: var(--border-strong);
  background: var(--surface-hover);
}

.ai-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.ai-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.ai-button.is-primary {
  border-color: transparent;
  background: var(--accent);
  color: var(--accent-contrast);
}

.ai-input {
  flex: 1;
  min-width: 240px;
  min-height: 32px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  outline: 0;
  background: var(--surface-raised);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11.5px;
}

.ai-input:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 15%, transparent);
}

.ai-empty-result {
  padding: 18px 0;
  color: var(--subtle);
  font-size: 12.5px;
  line-height: 1.6;
}

.ai-result {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ai-mig {
  margin: 0;
  color: var(--muted);
  font-size: 11.5px;
}

.ai-mig strong {
  color: var(--text);
  font-weight: 600;
}

.ai-result-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.ai-repositories {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>

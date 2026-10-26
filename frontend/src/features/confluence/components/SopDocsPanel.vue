<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue"
import type { ConfluenceDocument, SopGenerateRepository, SopScanResult } from "../types"
import { useCopilot } from "../../copilot/composables/useCopilot"
import ModelPicker from "../../copilot/components/ModelPicker.vue"
import { useMicroservices } from "../../microservices/composables/useMicroservices"
import { useToast } from "../../../composables/useToast"
import { renderMarkdown } from "../../../lib/markdown"
import { useSopDocs } from "../composables/sop-docs-store"
import {
  SOP_DOCS_LOADING_MESSAGE
} from "../composables/useSopDocs"
import { isMarkdownSopBlock, sopBlockToText } from "../utils/sop-block"

const props = defineProps<{
  document: ConfluenceDocument
  model: string
}>()

const emit = defineEmits<{
  (event: "update:model", value: string): void
}>()

const { hasConfiguredAI, isCheckingProviders } = useCopilot()
const { workspace, loadWorkspace } = useMicroservices()
const { showToast } = useToast()
const {
  status,
  result,
  scans,
  errorMessage,
  helperUrl,
  isHelperUnreachable,
  isApplying,
  applyError,
  applyResult,
  isLoading,
  cancel,
  retry,
  apply
} = useSopDocs()

const panelHeading = ref<HTMLElement | null>(null)

const rootPath = computed(() => workspace.value?.root_path.trim() ?? "")
const repositories = computed<SopGenerateRepository[]>(() => result.value?.repositories ?? [])
const warnings = computed(() => result.value?.warnings ?? [])
const deploy = computed(() => result.value?.deploy ?? null)
const operasional = computed(() => result.value?.operasional ?? null)

const deployText = computed(() => sopBlockToText(deploy.value?.managed_block ?? ""))
const isDeployMarkdown = computed(() => isMarkdownSopBlock(deploy.value?.managed_block ?? ""))
const deployMarkdownHtml = computed(() => renderMarkdown(deploy.value?.managed_block ?? ""))

const operasionalText = computed(() => {
  const plan = operasional.value
  if (plan === null) {
    return ""
  }
  const lines = [plan.title]
  for (const child of plan.children) {
    lines.push(`${child.exists ? "[x]" : "[ ]"} ${child.title}`)
  }
  return lines.join("\n")
})

const applyMode = computed(() => (deploy.value?.exists === true ? "update" : "create"))
const applyLabel = computed(() =>
  applyMode.value === "update" ? "Update SOP pages" : "Create SOP pages"
)

const scanFor = (repository: SopGenerateRepository): SopScanResult | null =>
  scans.value[repository.repository] ?? null

const copyText = async (text: string, successMessage: string): Promise<void> => {
  if (text.trim().length === 0) {
    return
  }
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

const copyDeploy = (): Promise<void> =>
  copyText(deployText.value, "SOP Deploy content copied.")

const copyOperasional = (): Promise<void> =>
  copyText(operasionalText.value, "SOP Operasional plan copied.")

watch(status, async (nextStatus) => {
  if (nextStatus === "success" || nextStatus === "error") {
    await nextTick()
    panelHeading.value?.focus()
  }
})

onMounted(async () => {
  if (workspace.value === null) {
    await loadWorkspace()
  }
})
</script>

<template>
  <div class="detail-section">
    <h3 ref="panelHeading" class="ai-panel-heading" tabindex="-1" data-testid="ai-sop-heading">
      SOP Deploy &amp; Operasional
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
          trigger-test-id="sop-docs-model-trigger"
          @update:model-value="emit('update:model', $event)"
        />
      </div>

      <div v-if="isCheckingProviders" class="ai-state" data-testid="ai-sop-checking">
        <span class="ai-spinner" aria-hidden="true"></span>
        <span>Checking the configured AI providers…</span>
      </div>

      <template v-else>
        <p class="ai-root-line" data-testid="ai-sop-root">
          <span class="ai-field-label">Workspace root</span>
          <code>{{ rootPath || "Not configured" }}</code>
        </p>

        <div v-if="isLoading" class="ai-loading" aria-live="polite" data-testid="ai-sop-loading">
          <div class="ai-skeleton" aria-hidden="true">
            <span></span>
            <span></span>
            <span></span>
          </div>
          <p class="ai-state-text">{{ SOP_DOCS_LOADING_MESSAGE }}</p>
          <button class="ai-button" type="button" data-testid="ai-sop-cancel" @click="cancel">
            Cancel
          </button>
        </div>

        <div
          v-else-if="status === 'error'"
          class="ai-notice is-danger"
          aria-live="polite"
          data-testid="ai-sop-error"
        >
          <p class="ai-notice-text">{{ errorMessage }}</p>
          <p v-if="isHelperUnreachable && helperUrl" class="ai-helper-line">
            Helper: <code>{{ helperUrl }}</code>
          </p>
          <RouterLink v-if="isHelperUnreachable" to="/settings" class="ai-notice-link">
            Open the helper setup guide →
          </RouterLink>
          <button v-else class="ai-button" type="button" data-testid="ai-sop-retry" @click="retry">
            Try again
          </button>
        </div>

        <div
          v-else-if="status === 'success' && result"
          class="ai-sop-result"
          aria-live="polite"
          data-testid="ai-sop-result"
        >
          <p class="ai-sop-issue" data-testid="ai-sop-issue-key">
            Issue <strong>{{ result.issue_key }}</strong> under page {{ result.development_page_id }}
          </p>

          <div
            v-if="warnings.length > 0"
            class="ai-notice is-warning"
            data-testid="ai-sop-warnings"
          >
            <p class="ai-notice-text">Warnings</p>
            <ul class="ai-sop-list">
              <li v-for="warning in warnings" :key="warning">{{ warning }}</li>
            </ul>
          </div>

          <div class="ai-sop-block" data-testid="ai-sop-deploy">
            <div class="ai-sop-block-head">
              <h4 class="ai-sop-block-title" data-testid="ai-sop-deploy-title">
                {{ result.deploy.title }}
              </h4>
              <span
                class="ai-sop-badge"
                :class="result.deploy.exists ? 'is-existing' : 'is-new'"
                data-testid="ai-sop-deploy-exists"
              >
                {{ result.deploy.exists ? "Exists" : "New" }}
              </span>
            </div>

            <p
              v-if="result.deploy.exists"
              class="ai-notice"
              data-testid="ai-sop-deploy-note"
            >
              This page already exists. Only the region between the lunar markers is replaced, and
              everything else on the page is kept.
            </p>

            <div
              v-if="isDeployMarkdown"
              class="ai-sop-preview"
              data-testid="ai-sop-deploy-preview"
              v-html="deployMarkdownHtml"
            ></div>
            <pre
              v-else
              class="ai-sop-preview"
              data-testid="ai-sop-deploy-preview"
            ><code>{{ deployText }}</code></pre>

            <div class="ai-result-actions">
              <button
                class="ai-button"
                type="button"
                data-testid="ai-sop-copy-deploy"
                @click="copyDeploy"
              >
                Copy SOP Deploy
              </button>
            </div>
          </div>

          <div class="ai-sop-block" data-testid="ai-sop-operasional">
            <div class="ai-sop-block-head">
              <h4 class="ai-sop-block-title" data-testid="ai-sop-operasional-title">
                {{ result.operasional.title }}
              </h4>
              <span
                class="ai-sop-badge"
                :class="result.operasional.exists ? 'is-existing' : 'is-new'"
                data-testid="ai-sop-operasional-exists"
              >
                {{ result.operasional.exists ? "Exists" : "New" }}
              </span>
            </div>

            <ul class="ai-sop-children" data-testid="ai-sop-children">
              <li
                v-for="child in result.operasional.children"
                :key="child.title"
                class="ai-sop-child"
                data-testid="ai-sop-child"
              >
                <span class="ai-sop-child-title">{{ child.title }}</span>
                <span
                  class="ai-sop-child-state"
                  :class="child.exists ? 'is-existing' : 'is-missing'"
                  data-testid="ai-sop-child-exists"
                >
                  {{ child.exists ? "Exists" : "Missing" }}
                </span>
              </li>
            </ul>

            <div class="ai-result-actions">
              <button
                class="ai-button"
                type="button"
                data-testid="ai-sop-copy-operasional"
                @click="copyOperasional"
              >
                Copy children plan
              </button>
            </div>
          </div>

          <div
            v-if="repositories.length > 0"
            class="ai-sop-repositories"
            data-testid="ai-sop-repositories"
          >
            <p class="ai-field-label">Repositories scanned</p>
            <div
              v-for="repository in repositories"
              :key="repository.repository"
              class="ai-sop-repository"
              data-testid="ai-sop-repository"
            >
              <span class="ai-sop-repository-name" data-testid="ai-sop-repository-name">
                {{ repository.repository }} · {{ repository.branch }}
              </span>
              <span
                v-if="scanFor(repository)"
                class="ai-sop-repository-tables"
                data-testid="ai-sop-repository-tables"
              >
                {{ scanFor(repository)?.tables.length ?? 0 }} table(s),
                {{ scanFor(repository)?.database_files.length ?? 0 }} file(s)
              </span>
              <ul
                v-if="scanFor(repository) && (scanFor(repository)?.warnings.length ?? 0) > 0"
                class="ai-sop-list"
                data-testid="ai-sop-repository-warnings"
              >
                <li v-for="warning in scanFor(repository)?.warnings" :key="warning">
                  {{ warning }}
                </li>
              </ul>
            </div>
          </div>

          <div class="ai-result-actions">
            <button
              class="ai-button is-primary"
              type="button"
              :disabled="isApplying"
              :aria-busy="isApplying"
              data-testid="ai-sop-apply"
              @click="apply"
            >
              {{ isApplying ? "Applying…" : applyLabel }}
            </button>
          </div>

          <p v-if="applyResult" class="ai-sop-applied" data-testid="ai-sop-applied">
            Deploy page {{ applyResult.deploy_page.action }} at version
            {{ applyResult.deploy_page.version }}.
          </p>

          <div
            v-if="applyError"
            class="ai-notice is-danger"
            aria-live="polite"
            data-testid="ai-sop-apply-error"
          >
            <p class="ai-notice-text">{{ applyError }}</p>
            <button
              class="ai-button"
              type="button"
              :disabled="isApplying"
              data-testid="ai-sop-apply-retry"
              @click="apply"
            >
              Retry apply
            </button>
          </div>
        </div>

        <div v-else class="ai-state" data-testid="ai-sop-idle">
          Choose a model, then run <strong>Generate SOP Deploy &amp; Operasional</strong>. The SOP
          Deploy preview, the SOP Operasional children, and the warnings appear here.
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
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px 18px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface-raised);
}

.ai-output-head {
  display: flex;
  align-items: center;
  gap: 8px;
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

.ai-helper-line {
  margin: 0 0 10px;
  font-size: 11px;
}

.ai-helper-line code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.ai-field-label {
  margin: 0;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-root-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin: 0;
  color: var(--muted);
  font-size: 11px;
}

.ai-root-line code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  overflow-wrap: anywhere;
}

.ai-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
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

.ai-result-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.ai-sop-result {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.ai-sop-issue {
  margin: 0;
  color: var(--muted);
  font-size: 11.5px;
}

.ai-sop-issue strong {
  color: var(--text);
  font-weight: 600;
}

.ai-sop-list {
  margin: 0;
  padding-left: 18px;
  color: inherit;
  font-size: 11.5px;
  line-height: 1.7;
}

.ai-sop-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.ai-sop-block-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.ai-sop-block-title {
  margin: 0;
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
}

.ai-sop-badge {
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 9.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.ai-sop-badge.is-existing {
  border-color: color-mix(in srgb, var(--positive) 45%, transparent);
  color: var(--positive);
}

.ai-sop-badge.is-new {
  border-color: color-mix(in srgb, var(--accent) 45%, transparent);
  color: var(--accent);
}

.ai-sop-preview {
  max-height: 340px;
  overflow: auto;
  margin: 0;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 12px;
  line-height: 1.65;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

pre.ai-sop-preview code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11.5px;
}

.ai-sop-preview :deep(p) {
  margin: 0 0 8px;
}

.ai-sop-children {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.ai-sop-child {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  font-size: 11.5px;
}

.ai-sop-child-title {
  color: var(--text);
}

.ai-sop-child-state {
  font-size: 9.5px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.ai-sop-child-state.is-existing {
  color: var(--positive);
}

.ai-sop-child-state.is-missing {
  color: var(--subtle);
}

.ai-sop-repositories {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-sop-repository {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
}

.ai-sop-repository-name {
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  overflow-wrap: anywhere;
}

.ai-sop-repository-tables {
  color: var(--subtle);
  font-size: 10.5px;
}

.ai-sop-applied {
  margin: 0;
  color: var(--positive);
  font-size: 11.5px;
}
</style>

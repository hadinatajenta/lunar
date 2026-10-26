<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import type { ConfluenceDocument, UtRunPhase } from "../types"
import { useCopilot } from "../../copilot/composables/useCopilot"
import ModelPicker from "../../copilot/components/ModelPicker.vue"
import { useMicroservices } from "../../microservices/composables/useMicroservices"
import { useToast } from "../../../composables/useToast"
import { UT_RUN_PHASES } from "../api/ut-run-events"
import { useUtDocs } from "../composables/ut-docs-store"
import { UT_DOCS_EMPTY_MESSAGE } from "../composables/useUtDocs"
import { buildUtScenarioTableHtml, buildUtScenarioTableText } from "../utils/ut-test-table"
import UtMockTestCodes from "./UtMockTestCodes.vue"
import UtRunSummary from "./UtRunSummary.vue"
import UtTestScenarioTable from "./UtTestScenarioTable.vue"

const props = defineProps<{
  document: ConfluenceDocument
  model: string
}>()

const emit = defineEmits<{
  (event: "update:model", value: string): void
}>()

const PHASE_LABELS: Record<UtRunPhase, string> = {
  scan: "Scan changed files",
  safety: "Classify safety",
  run: "Run mock-only tests",
  screenshot: "Render screenshot"
}

const { hasConfiguredAI, isCheckingProviders } = useCopilot()
const { workspace, loadWorkspace } = useMicroservices()
const { showToast } = useToast()
const {
  status,
  phase,
  outputText,
  currentRepository,
  migPageTitle,
  targets,
  skippedRepositories,
  runResults,
  generatedDocument,
  errorMessage,
  helperUrl,
  isHelperUnreachable,
  screenshotUrls,
  screenshotError,
  isApplying,
  applyError,
  isRunning,
  cancel,
  retry,
  apply,
  releaseScreenshots
} = useUtDocs()

const panelHeading = ref<HTMLElement | null>(null)

const rootPath = computed(() => workspace.value?.root_path.trim() ?? "")
const scenarioRows = computed(() => generatedDocument.value?.rows ?? [])
const mockTests = computed(() => generatedDocument.value?.mock_tests ?? [])

const phaseIndex = computed(() => {
  if (status.value === "generating" || status.value === "success") {
    return UT_RUN_PHASES.length
  }
  if (phase.value === null) {
    return -1
  }
  return UT_RUN_PHASES.indexOf(phase.value)
})

const phaseState = (phaseId: UtRunPhase): "done" | "active" | "pending" => {
  const targetIndex = UT_RUN_PHASES.indexOf(phaseId)
  if (phaseIndex.value > targetIndex) {
    return "done"
  }
  if (phaseIndex.value === targetIndex) {
    return "active"
  }
  return "pending"
}

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

const writeRichTable = async (html: string, text: string): Promise<boolean> => {
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

const copyScenarioTable = async (): Promise<void> => {
  if (generatedDocument.value === null) {
    return
  }
  const rows = scenarioRows.value
  const successMessage = "UT scenario table copied for Confluence."
  if (
    await writeRichTable(
      buildUtScenarioTableHtml(rows, mockTests.value),
      buildUtScenarioTableText(rows, mockTests.value)
    )
  ) {
    showToast(successMessage)
    return
  }
  await copyText(buildUtScenarioTableText(rows, mockTests.value), successMessage)
}

watch(status, async (nextStatus) => {
  if (
    nextStatus === "success" ||
    nextStatus === "empty" ||
    nextStatus === "error"
  ) {
    await nextTick()
    panelHeading.value?.focus()
  }
})

onMounted(async () => {
  if (workspace.value === null) {
    await loadWorkspace()
  }
})

onBeforeUnmount(() => {
  releaseScreenshots()
})
</script>

<template>
  <div class="detail-section">
    <h3 ref="panelHeading" class="ai-panel-heading" tabindex="-1" data-testid="ai-ut-heading">
      UT documentation
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
          trigger-test-id="ut-docs-model-trigger"
          @update:model-value="emit('update:model', $event)"
        />
      </div>

      <div v-if="isCheckingProviders" class="ai-state" data-testid="ai-ut-checking">
        <span class="ai-spinner" aria-hidden="true"></span>
        <span>Checking the configured AI providers…</span>
      </div>

      <template v-else>
        <div
          v-if="!hasConfiguredAI"
          class="ai-notice is-warning"
          data-testid="banner-no-ai-keys"
        >
          <span>No AI provider is configured yet. Add an API key in Settings to generate the UT document.</span>
          <RouterLink to="/settings" class="ai-notice-link">Configure in Settings →</RouterLink>
        </div>

        <div class="ai-ut-config">
          <p class="ai-root-line" data-testid="ai-ut-root">
            <span class="ai-field-label">Workspace root</span>
            <code>{{ rootPath || "Not configured" }}</code>
          </p>

          <p v-if="migPageTitle" class="ai-mig" data-testid="ai-ut-mig-page">
            MIG page <strong>{{ migPageTitle }}</strong>
          </p>

          <div v-if="targets.length > 0" class="ai-ut-targets" data-testid="ai-ut-targets">
            <p class="ai-field-label">Repositories to run</p>
            <div class="ai-chips">
              <span
                v-for="target in targets"
                :key="target.repository"
                class="ai-chip"
                data-testid="ai-ut-target"
              >
                {{ target.repository }} · {{ target.branch }}
              </span>
            </div>
          </div>

          <p
            v-if="skippedRepositories.length > 0"
            class="ai-notice is-warning"
            data-testid="ai-ut-skipped-repositories"
          >
            Excluded from this run:
            <span v-for="(skipped, index) in skippedRepositories" :key="skipped.repository">
              <span v-if="index > 0">, </span>{{ skipped.repository }} ({{ skipped.reason }})
            </span>
          </p>
        </div>

        <div
          v-if="isRunning || phase !== null"
          class="ai-ut-phases"
          aria-live="polite"
          data-testid="ai-ut-phases"
        >
          <span
            v-for="phaseId in UT_RUN_PHASES"
            :key="phaseId"
            class="ai-phase"
            :class="`is-${phaseState(phaseId)}`"
            :data-phase="phaseId"
            data-testid="ai-ut-phase"
          >
            {{ PHASE_LABELS[phaseId] }}
          </span>
        </div>

        <p
          v-if="currentRepository"
          class="ai-state"
          aria-live="polite"
          data-testid="ai-ut-current-repository"
        >
          Running unit tests in <strong>{{ currentRepository }}</strong>…
        </p>

        <pre v-if="outputText.length > 0" class="ai-ut-output" data-testid="ai-ut-output"><code>{{ outputText }}</code></pre>

        <button
          v-if="isRunning"
          class="ai-button"
          type="button"
          data-testid="ai-ut-cancel"
          @click="cancel"
        >
          Cancel run
        </button>

        <p v-if="status === 'generating'" class="ai-state" aria-live="polite" data-testid="ai-ut-generating">
          Generating the UT document from the run result…
        </p>

        <div
          v-if="status === 'error'"
          class="ai-notice is-danger"
          aria-live="polite"
          data-testid="ai-ut-error"
        >
          <p class="ai-notice-text">{{ errorMessage }}</p>
          <p v-if="isHelperUnreachable && helperUrl" class="ai-helper-line">
            Helper: <code>{{ helperUrl }}</code>
          </p>
          <template v-if="isHelperUnreachable">
            <RouterLink to="/settings" class="ai-notice-link">
              Open the helper setup guide →
            </RouterLink>
          </template>
          <button v-else class="ai-button" type="button" data-testid="ai-ut-retry" @click="retry">
            Try again
          </button>
        </div>

        <div v-if="status === 'empty'" class="ai-state" aria-live="polite" data-testid="ai-ut-empty">
          {{ UT_DOCS_EMPTY_MESSAGE }}
        </div>

        <div v-if="runResults.length > 0" class="ai-ut-runs">
          <UtRunSummary
            v-for="result in runResults"
            :key="result.run_id"
            :result="result"
            :repository="result.repo"
          />
        </div>

        <p v-if="screenshotError" class="ai-notice" data-testid="ai-ut-screenshot-error">
          {{ screenshotError }}
        </p>

        <div
          v-if="generatedDocument"
          class="ai-ut-document"
          aria-live="polite"
          data-testid="ai-ut-document"
        >
          <h4 class="ai-ut-document-title" data-testid="ai-ut-document-title">
            {{ generatedDocument.title }}
          </h4>
          <p class="ai-ut-document-summary" data-testid="ai-ut-document-summary">
            {{ generatedDocument.summary }}
          </p>
          <p
            v-if="generatedDocument.blocked_notice"
            class="ai-notice is-warning"
            data-testid="ai-ut-blocked-notice"
          >
            {{ generatedDocument.blocked_notice }}
          </p>

          <p class="ai-field-label">Normal Test Scenario</p>
          <UtTestScenarioTable :rows="scenarioRows" :screenshot-urls="screenshotUrls" />

          <UtMockTestCodes :mock-tests="mockTests" />

          <div class="ai-result-actions">
            <button
              class="ai-button"
              type="button"
              data-testid="ai-ut-copy-table"
              @click="copyScenarioTable"
            >
              Copy table
            </button>
          </div>

          <div
            v-if="applyError"
            class="ai-notice is-danger"
            aria-live="polite"
            data-testid="ai-ut-apply-error"
          >
            <p class="ai-notice-text">{{ applyError }}</p>
            <button
              class="ai-button"
              type="button"
              :disabled="isApplying"
              data-testid="ai-ut-apply-retry"
              @click="apply"
            >
              Retry apply
            </button>
          </div>
        </div>

        <div
          v-else-if="status === 'idle' && runResults.length === 0"
          class="ai-state"
          data-testid="ai-ut-idle"
        >
          Choose a model, then choose Generate UT Docs With AI. The repositories on the MIG page run
          their mock-only unit tests, and the run summary, the screenshot, and the generated document
          appear here.
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

.ai-state {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--subtle);
  font-size: 12px;
  line-height: 1.6;
}

.ai-spinner {
  width: 13px;
  height: 13px;
  flex: 0 0 auto;
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

.ai-ut-config {
  display: flex;
  flex-direction: column;
  gap: 10px;
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

.ai-field-label {
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
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

.ai-ut-targets {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ai-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.ai-chip {
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10.5px;
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

.ai-ut-phases {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.ai-phase {
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--subtle);
  font-size: 10.5px;
}

.ai-phase.is-active {
  border-color: var(--accent);
  color: var(--text);
}

.ai-phase.is-done {
  border-color: color-mix(in srgb, var(--positive) 45%, transparent);
  color: var(--positive);
}

.ai-ut-output {
  max-height: 280px;
  overflow: auto;
  margin: 0;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.28);
  color: #d7dee6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.6;
}

.ai-ut-output code {
  font-family: inherit;
}

.ai-ut-runs {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ai-ut-document {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ai-ut-document-title {
  margin: 0;
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
}

.ai-ut-document-summary {
  margin: 0;
  color: var(--text);
  font-size: 12.5px;
  line-height: 1.65;
}

.ai-result-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
</style>

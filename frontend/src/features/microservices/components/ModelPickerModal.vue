<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue"
import CollectionDownloadModal from "./CollectionDownloadModal.vue"
import ExportProgressList from "./ExportProgressList.vue"
import MicroservicesModal from "./MicroservicesModal.vue"
import ModelCatalogueList from "./ModelCatalogueList.vue"
import { useProviderConfiguration } from "../../copilot/composables/useProviderConfiguration"
import { useCollectionExport } from "../composables/useCollectionExport"

const props = defineProps<{
  repositoryName: string
  selectedModelId: string
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "select", modelId: string): void
}>()

const modelId = ref(props.selectedModelId)
const modelList = ref<InstanceType<typeof ModelCatalogueList> | null>(null)
const lockNotice = ref<HTMLElement | null>(null)
const triggerElement = ref<HTMLElement | null>(null)
let isActive = true

const { configuredModels, hasConfiguredAI, isCheckingProviders, refreshConfiguredProviders } =
  useProviderConfiguration(modelId)

const {
  stageEvents,
  isExporting,
  failureMessage,
  download,
  startExport,
  downloadCollection,
  dispose
} = useCollectionExport(props.repositoryName)

const resolveConfiguredModelId = (candidateId: string): string | null => {
  const matchedModel = configuredModels.value.find((model) => model.id === candidateId)
  return matchedModel === undefined ? null : matchedModel.id
}

const hasConfiguredModel = computed(() => resolveConfiguredModelId(modelId.value) !== null)

const isModelListDisabled = computed(() => isExporting.value || configuredModels.value.length === 0)

const canExport = computed(
  () => !isCheckingProviders.value && !isExporting.value && hasConfiguredModel.value
)

const exportButtonLabel = computed(() => {
  if (isExporting.value) return "Exporting…"
  if (failureMessage.value.length > 0) return "Retry export"
  return "Export collection"
})

const footerNote = computed(() => (isExporting.value ? "The AI model may take a minute to answer." : ""))

const unavailableReason = computed(() => {
  if (configuredModels.value.length > 0) return ""
  if (hasConfiguredAI.value) return "The configured providers do not expose a model that Lunar can use yet."
  return "No AI provider key is configured, so the export is locked. Add a provider key in Settings to unlock it."
})

const selectModel = (nextModelId: string): void => {
  if (isModelListDisabled.value || resolveConfiguredModelId(nextModelId) === null) return
  modelId.value = nextModelId
  emit("select", nextModelId)
}

const reconcileSelection = (): void => {
  const configuredModelId = resolveConfiguredModelId(modelId.value)
  if (configuredModelId !== null) {
    modelId.value = configuredModelId
    if (configuredModelId !== props.selectedModelId) emit("select", configuredModelId)
    return
  }

  const fallbackModelId = configuredModels.value.length > 0 ? configuredModels.value[0].id : ""
  modelId.value = fallbackModelId
  if (fallbackModelId.length > 0) emit("select", fallbackModelId)
}

const focusSettingsLink = (): void => {
  lockNotice.value?.querySelector<HTMLElement>("a")?.focus()
}

const hydrateSelection = async (): Promise<void> => {
  await refreshConfiguredProviders()
  if (!isActive) return
  reconcileSelection()
  await nextTick()
  if (!isActive) return
  if (configuredModels.value.length > 0) {
    modelList.value?.focusSelected()
    return
  }
  focusSettingsLink()
}

const startSelectedExport = (): void => {
  if (!canExport.value) return
  void startExport(modelId.value)
}

const handleKeydown = (event: KeyboardEvent): void => {
  if (event.key !== "Escape") return
  event.preventDefault()
  emit("close")
}

watch(
  () => props.selectedModelId,
  (nextModelId) => {
    const configuredModelId = resolveConfiguredModelId(nextModelId)
    if (configuredModelId !== null) modelId.value = configuredModelId
  }
)

onMounted(() => {
  const activeElement = document.activeElement
  triggerElement.value = activeElement instanceof HTMLElement ? activeElement : null
  window.addEventListener("keydown", handleKeydown)
  void hydrateSelection()
})

onUnmounted(() => {
  isActive = false
  window.removeEventListener("keydown", handleKeydown)
  dispose()
  const element = triggerElement.value
  if (element !== null && element.isConnected) element.focus()
})
</script>

<template>
  <CollectionDownloadModal
    v-if="download !== null"
    :service-name="download.serviceName"
    :file-name="download.fileName"
    :request-count="download.requestCount"
    :folder-count="download.folderCount"
    @download="downloadCollection"
    @close="emit('close')"
  />

  <MicroservicesModal
    v-else
    data-testid="model-picker-overlay"
    :eyebrow="props.repositoryName"
    title="Choose an AI model to export"
    title-id="model-picker-title"
    @close="emit('close')"
  >
    <p class="section-label">AI model</p>

    <p v-if="isCheckingProviders" class="model-loading" role="status">
      Checking the configured AI providers…
    </p>

    <template v-else>
      <ModelCatalogueList
        ref="modelList"
        :models="configuredModels"
        :selected-model-id="modelId"
        :is-disabled="isModelListDisabled"
        @select="selectModel"
      />

      <p v-if="unavailableReason.length > 0" ref="lockNotice" class="model-lock" role="status">
        <span>{{ unavailableReason }}</span>
        <RouterLink class="settings-link" to="/settings">Open Settings</RouterLink>
      </p>
      <p v-else class="model-hint">
        Lunar reads this repository and asks the selected model for a Postman collection. The JSON
        downloads to your browser; nothing is saved on the server.
      </p>

      <p v-if="hasConfiguredModel" class="model-saved">
        Saved as the model preference for {{ props.repositoryName }}.
      </p>
    </template>

    <ExportProgressList :stage-events="stageEvents" :is-exporting="isExporting" />

    <p v-if="failureMessage.length > 0" class="export-error" role="alert">{{ failureMessage }}</p>

    <template #footer>
      <span class="footer-note">{{ footerNote }}</span>
      <span class="spacer"></span>
      <button class="footer-btn is-secondary" type="button" @click="emit('close')">Cancel</button>
      <button class="footer-btn is-ai" type="button" :disabled="!canExport" @click="startSelectedExport">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
          <path d="M7 10l5 5 5-5"></path>
          <path d="M12 15V3"></path>
        </svg>
        {{ exportButtonLabel }}
      </button>
    </template>
  </MicroservicesModal>
</template>

<style scoped>
.section-label {
  margin: 0 0 10px; color: var(--subtle); font-size: 10px; font-weight: 600; letter-spacing: 0.09em; text-transform: uppercase;
}
.model-loading {
  margin: 0; padding: 13px 14px; border: 1px solid var(--border); border-radius: 10px;
  background: var(--surface-raised); color: var(--subtle); font-size: 11.5px;
}
.model-hint { min-height: 16px; margin: 10px 0 0; color: var(--subtle); font-size: 10.5px; line-height: 1.55; }
.model-lock {
  display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; margin: 12px 0 0;
  padding: 10px 12px; border: 1px solid color-mix(in srgb, var(--warning) 30%, transparent); border-radius: 8px;
  background: color-mix(in srgb, var(--warning) 9%, transparent); color: var(--warning); font-size: 11px; line-height: 1.5;
}
.settings-link { color: var(--warning); font-weight: 600; text-decoration: underline; white-space: nowrap; }
.settings-link:focus-visible { outline: 2px solid var(--warning); outline-offset: 2px; border-radius: 4px; }
.model-saved {
  margin: 12px 0 0; padding: 9px 11px; border: 1px solid color-mix(in srgb, var(--positive) 28%, transparent);
  border-radius: 8px; background: color-mix(in srgb, var(--positive) 9%, transparent); color: var(--positive); font-size: 11px;
}
.export-error {
  margin: 12px 0 0; padding: 10px 12px; border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  border-radius: 8px; background: color-mix(in srgb, var(--danger) 10%, transparent); color: var(--danger);
  font-size: 11.5px; line-height: 1.55; overflow-wrap: anywhere;
}

.footer-note { color: var(--subtle); font-size: 10.5px; }
.spacer { flex: 1; }
.footer-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 0 14px;
  border: 1px solid transparent; border-radius: 8px; font-size: 11.5px; font-weight: 600; white-space: nowrap;
  cursor: pointer; transition: opacity 150ms ease, border-color 150ms ease, background-color 150ms ease;
}
.footer-btn.is-secondary { border-color: var(--border); background: transparent; color: var(--text); }
.footer-btn.is-secondary:hover { border-color: var(--border-strong); background: var(--surface-hover); }
.footer-btn.is-ai {
  border-color: color-mix(in srgb, var(--positive) 32%, transparent); background: color-mix(in srgb, var(--positive) 10%, transparent);
  color: var(--positive);
}
.footer-btn.is-ai:hover:not(:disabled) { background: color-mix(in srgb, var(--positive) 18%, transparent); }
.footer-btn:disabled { opacity: 0.45; cursor: not-allowed; }
.footer-btn svg {
  width: 13px; height: 13px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round;
}

@media (max-width: 620px) {
  .spacer { display: none; }
  .footer-btn { width: 100%; }
  .footer-note { order: 3; }
}
</style>

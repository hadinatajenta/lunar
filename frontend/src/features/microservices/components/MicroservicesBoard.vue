<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import ModelPickerModal from "./ModelPickerModal.vue"
import RepositoryPickerModal from "./RepositoryPickerModal.vue"
import ServiceTable from "./ServiceTable.vue"
import ServiceToolbar from "./ServiceToolbar.vue"
import WorkspaceStatePanel from "./WorkspaceStatePanel.vue"
import { SEARCH_DEBOUNCE_MS, useDebouncedValue } from "./debounced-value"
import { findModelOption, readModelPreferences, storeModelPreference } from "./model-preferences"
import type { ServiceFilter } from "./service-filters"
import { countChangedFiles, formatServiceCount } from "./service-presentation"
import { useMicroservices } from "../composables/useMicroservices"
import type { RepositoryDetail, RepositorySummary } from "../types"

const {
  repositories,
  services,
  selectedNames,
  isLoading,
  isSaving,
  error,
  loadRepositories,
  loadServices,
  saveSelections,
  clearError
} = useMicroservices()

const searchQuery = ref("")
const debouncedSearchQuery = useDebouncedValue(searchQuery, SEARCH_DEBOUNCE_MS)
const activeFilter = ref<ServiceFilter>("all")
const openServiceNames = ref<string[]>([])
const isPickerOpen = ref(false)
const modelTargetName = ref<string | null>(null)
const modelPreferences = ref<Record<string, string>>(readModelPreferences())
const toolbar = ref<InstanceType<typeof ServiceToolbar> | null>(null)

const serviceList = computed<RepositoryDetail[]>(() =>
  services.value.map((service) => ({
    ...service,
    commit: { ...service.commit },
    files: service.files.map((file) => ({ ...file }))
  }))
)

const repositoryList = computed<RepositorySummary[]>(() =>
  repositories.value.map((repository) => ({ ...repository }))
)

const selectedRepositoryNames = computed<string[]>(() => [...selectedNames.value])

const filteredServices = computed(() => {
  const query = debouncedSearchQuery.value.trim().toLowerCase()
  return serviceList.value.filter((service) => {
    const changedFileCount = countChangedFiles(service)
    if (activeFilter.value === "dirty" && changedFileCount === 0) return false
    if (activeFilter.value === "clean" && changedFileCount > 0) return false
    if (query.length === 0) return true
    return service.name.toLowerCase().includes(query) || service.branch.toLowerCase().includes(query)
  })
})

const serviceCountLabel = computed(() =>
  serviceList.value.length === 0
    ? "No repositories selected"
    : `${filteredServices.value.length} of ${formatServiceCount(serviceList.value.length)}`
)

const tableEmptyMessage = computed(() =>
  serviceList.value.length === 0
    ? "No repository details are available yet."
    : "No repository matches the current search and filter."
)

const modelLabels = computed(() => {
  const labels: Record<string, string> = {}
  for (const [repositoryName, modelId] of Object.entries(modelPreferences.value)) {
    const modelOption = findModelOption(modelId)
    if (modelOption !== null) labels[repositoryName] = modelOption.name
  }
  return labels
})

const selectedModelId = computed(() =>
  modelTargetName.value === null ? "" : modelPreferences.value[modelTargetName.value] ?? ""
)

const restorePickerFocus = () => {
  void nextTick(() => {
    toolbar.value?.focusPickerButton()
  })
}

const toggleService = (serviceName: string) => {
  openServiceNames.value = openServiceNames.value.includes(serviceName)
    ? openServiceNames.value.filter((openName) => openName !== serviceName)
    : [...openServiceNames.value, serviceName]
}

const openPicker = () => {
  clearError()
  isPickerOpen.value = true
}

const closePicker = () => {
  isPickerOpen.value = false
  restorePickerFocus()
}

const reloadBoard = async () => {
  await loadRepositories()
  await loadServices()
}

const applySelection = async (repositoryNames: string[]) => {
  const isSaved = await saveSelections(repositoryNames)
  isPickerOpen.value = false
  if (isSaved) {
    openServiceNames.value = openServiceNames.value.filter((openName) => repositoryNames.includes(openName))
    await loadServices()
  }
  restorePickerFocus()
}

const openModelPicker = (repositoryName: string) => {
  modelTargetName.value = repositoryName
}

const closeModelPicker = () => {
  modelTargetName.value = null
}

const selectModel = (modelId: string) => {
  if (modelTargetName.value === null) return
  storeModelPreference(modelTargetName.value, modelId)
  modelPreferences.value = readModelPreferences()
}
</script>

<template>
  <div>
    <ServiceToolbar
      ref="toolbar"
      :search-query="searchQuery"
      :active-filter="activeFilter"
      :count-label="serviceCountLabel"
      :is-picker-open="isPickerOpen"
      @update:search-query="searchQuery = $event"
      @update:active-filter="activeFilter = $event"
      @open-picker="openPicker"
    />

    <div v-if="error" class="error-banner" role="alert">
      <span class="banner-message">{{ error }}</span>
      <button class="banner-retry" type="button" @click="reloadBoard">Retry</button>
    </div>

    <WorkspaceStatePanel
      v-if="selectedRepositoryNames.length === 0"
      title="No repositories selected"
      message="Pick the repositories you want on this board. Use the gear button in the toolbar to choose them."
      action-label="Open repository picker"
      @action="openPicker"
    />

    <ServiceTable
      v-else
      :services="filteredServices"
      :open-service-names="openServiceNames"
      :model-labels="modelLabels"
      :empty-message="tableEmptyMessage"
      @toggle="toggleService"
      @open-model="openModelPicker"
    />

    <RepositoryPickerModal
      v-if="isPickerOpen"
      :repositories="repositoryList"
      :selected-names="selectedRepositoryNames"
      :is-loading="isLoading"
      :is-saving="isSaving"
      :error-message="error"
      @close="closePicker"
      @apply="applySelection"
      @retry="reloadBoard"
    />

    <ModelPickerModal
      v-if="modelTargetName !== null"
      :repository-name="modelTargetName"
      :selected-model-id="selectedModelId"
      @close="closeModelPicker"
      @select="selectModel"
    />
  </div>
</template>

<style scoped>
.error-banner {
  display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap;
  margin-bottom: 16px; padding: 12px 16px; border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  border-radius: 9px; background: color-mix(in srgb, var(--danger) 10%, transparent); color: var(--danger); font-size: 12.5px;
}
.banner-message { min-width: 0; overflow-wrap: anywhere; }
.banner-retry {
  display: inline-flex; align-items: center; justify-content: center; min-height: 30px; padding: 0 12px;
  border: 1px solid color-mix(in srgb, var(--danger) 34%, transparent); border-radius: 8px; background: transparent;
  color: var(--danger); font-size: 11.5px; font-weight: 600; white-space: nowrap; cursor: pointer;
  transition: background-color 150ms ease;
}
.banner-retry:hover { background: color-mix(in srgb, var(--danger) 16%, transparent); }
</style>

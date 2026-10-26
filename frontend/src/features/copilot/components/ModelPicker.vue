<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { useCopilot } from "../composables/useCopilot"
import type { ReasoningEffort } from "../types"

const props = withDefaults(
  defineProps<{
    modelValue?: string
    effortValue?: ReasoningEffort
    showEffort?: boolean
    disabled?: boolean
    triggerTestId?: string
  }>(),
  {
    showEffort: true,
    disabled: false,
    triggerTestId: "model-picker-trigger"
  }
)

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void
  (event: "update:effortValue", value: ReasoningEffort): void
}>()

const { selectedModel, reasoningEffort, configuredModels, allSupportedModels } = useCopilot()

const EFFORT_OPTIONS: { id: ReasoningEffort; label: string }[] = [
  { id: "low", label: "Low" },
  { id: "medium", label: "Med" },
  { id: "high", label: "High" }
]

const activeModel = computed(() => props.modelValue ?? selectedModel.value)
const activeEffort = computed(() => props.effortValue ?? reasoningEffort.value)
const isModelBound = computed(() => props.modelValue !== undefined)
const isEffortBound = computed(() => props.effortValue !== undefined)

const isModelMenuOpen = ref(false)
const menuView = ref<"root" | "model" | "effort">("root")
const modelSearchQuery = ref("")
const searchInputElement = ref<HTMLInputElement | null>(null)

const unconfiguredModels = computed(() => {
  const configuredNames = new Set(configuredModels.value.map((model) => model.name))
  return allSupportedModels.filter((model) => !configuredNames.has(model.name))
})

const filteredConfiguredModels = computed(() => {
  const query = modelSearchQuery.value.trim().toLowerCase()
  if (query === "") return configuredModels.value
  return configuredModels.value.filter((model) => model.name.toLowerCase().includes(query))
})

const openMenu = (): void => {
  if (props.disabled) return
  menuView.value = "root"
  modelSearchQuery.value = ""
  isModelMenuOpen.value = true
}

const toggleMenu = (): void => {
  isModelMenuOpen.value ? closeMenu() : openMenu()
}

const closeMenu = (): void => {
  isModelMenuOpen.value = false
  modelSearchQuery.value = ""
}

const selectModel = (modelName: string): void => {
  const isAllowed = configuredModels.value.some((model) => model.name === modelName)
  if (!isAllowed) return
  if (isModelBound.value) {
    emit("update:modelValue", modelName)
  } else {
    selectedModel.value = modelName
  }
  menuView.value = "root"
  modelSearchQuery.value = ""
}

const selectEffort = (effort: ReasoningEffort): void => {
  if (isEffortBound.value) {
    emit("update:effortValue", effort)
  } else {
    reasoningEffort.value = effort
  }
  menuView.value = "root"
}

const handleEscape = (event: KeyboardEvent): void => {
  if (event.key !== "Escape") return
  event.preventDefault()
  closeMenu()
}

watch(isModelMenuOpen, (isOpen) => {
  if (isOpen) {
    window.addEventListener("keydown", handleEscape)
    return
  }
  window.removeEventListener("keydown", handleEscape)
})

watch(menuView, async (view) => {
  if (view !== "model") return
  await nextTick()
  searchInputElement.value?.focus()
})

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleEscape)
})
</script>

<template>
  <div class="model-picker">
    <button
      class="model-trigger"
      type="button"
      :aria-expanded="isModelMenuOpen"
      :disabled="props.disabled"
      :data-testid="props.triggerTestId"
      title="Select a model"
      @click="toggleMenu"
    >
      <span class="model-trigger-label">{{ activeModel }}</span>
      <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="m6 15 6-6 6 6"></path>
      </svg>
    </button>

    <div v-if="isModelMenuOpen" class="model-menu">
      <template v-if="menuView === 'root'">
        <button
          class="model-option"
          type="button"
          data-testid="model-root-option-model"
          @click="menuView = 'model'"
        >
          <span>Model</span>
        </button>
        <button
          v-if="props.showEffort"
          class="model-option"
          type="button"
          data-testid="model-root-option-effort"
          @click="menuView = 'effort'"
        >
          <span>Effort</span>
        </button>
      </template>

      <template v-else-if="menuView === 'model'">
        <div class="model-menu-header">
          <button
            class="model-back"
            type="button"
            title="Back"
            data-testid="model-picker-back"
            @click="menuView = 'root'"
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m15 6-6 6 6 6"></path>
            </svg>
          </button>
          <input
            ref="searchInputElement"
            v-model="modelSearchQuery"
            class="model-search-input"
            type="text"
            placeholder="Search models"
            data-testid="model-search-input"
            @keydown.enter.prevent
          />
        </div>

        <div v-if="configuredModels.length === 0" class="model-empty-notice">
          <span>No AI providers configured.</span>
          <RouterLink to="/settings" class="model-settings-link">Configure in Settings →</RouterLink>
        </div>

        <template v-else>
          <button
            v-for="model in filteredConfiguredModels"
            :key="model.name"
            class="model-option"
            :class="{ 'is-selected': activeModel === model.name }"
            type="button"
            :aria-pressed="activeModel === model.name"
            @click="selectModel(model.name)"
          >
            <span>{{ model.name }}</span>
            <svg v-if="activeModel === model.name" class="check" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m5 13 4 4L19 7"></path>
            </svg>
          </button>
        </template>

        <div v-if="unconfiguredModels.length > 0" class="model-menu-divider">
          <span>Requires API Key in Settings</span>
        </div>

        <div
          v-for="model in unconfiguredModels"
          :key="model.name"
          class="model-option is-disabled"
          data-testid="model-option-disabled"
        >
          <span class="disabled-name">{{ model.name }}</span>
          <span class="unconfigured-tag">Locked</span>
        </div>
      </template>

      <template v-else>
        <div class="model-menu-header">
          <button
            class="model-back"
            type="button"
            title="Back"
            data-testid="model-picker-back"
            @click="menuView = 'root'"
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="m15 6-6 6 6 6"></path>
            </svg>
          </button>
          <span class="model-header-title">Effort</span>
        </div>

        <button
          v-for="option in EFFORT_OPTIONS"
          :key="option.id"
          class="model-option"
          :class="{ 'is-selected': activeEffort === option.id }"
          type="button"
          :data-testid="`effort-option-${option.id}`"
          :aria-pressed="activeEffort === option.id"
          @click="selectEffort(option.id)"
        >
          <span>{{ option.label }}</span>
          <svg v-if="activeEffort === option.id" class="check" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m5 13 4 4L19 7"></path>
          </svg>
        </button>
      </template>
    </div>
  </div>
</template>
<style scoped src="./model-picker.css"></style>

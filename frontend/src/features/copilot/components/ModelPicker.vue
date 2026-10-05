<script setup lang="ts">
import { computed, ref } from "vue"
import { useCopilot } from "../composables/useCopilot"

const { selectedModel, thinkingMode, configuredModels, allSupportedModels } = useCopilot()

const THINKING_MODEL_HINTS = ["Thinking", "Reasoning", "Astra", "Opus", "Fable", "Pro"]

const isModelMenuOpen = ref(false)

const unconfiguredModels = computed(() => {
  const configuredNames = new Set(configuredModels.value.map((model) => model.name))
  return allSupportedModels.filter((model) => !configuredNames.has(model.name))
})

const selectModel = (modelName: string): void => {
  const isAllowed = configuredModels.value.some((model) => model.name === modelName)
  if (!isAllowed) return
  selectedModel.value = modelName
  isModelMenuOpen.value = false
  if (THINKING_MODEL_HINTS.some((hint) => modelName.includes(hint))) {
    thinkingMode.value = true
  }
}
</script>

<template>
  <div class="model-picker" @keydown.escape="isModelMenuOpen = false">
    <button
      class="model-trigger"
      type="button"
      :aria-expanded="isModelMenuOpen"
      title="Select a model"
      @click="isModelMenuOpen = !isModelMenuOpen"
    >
      <span class="model-trigger-label">{{ selectedModel }}</span>
      <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="m6 15 6-6 6 6"></path>
      </svg>
    </button>

    <div v-if="isModelMenuOpen" class="model-menu">
      <div v-if="configuredModels.length === 0" class="model-empty-notice">
        <span>No AI providers configured.</span>
        <RouterLink to="/settings" class="model-settings-link">Configure in Settings →</RouterLink>
      </div>

      <template v-else>
        <button
          v-for="model in configuredModels"
          :key="model.name"
          class="model-option"
          :class="{ 'is-selected': selectedModel === model.name }"
          type="button"
          @click="selectModel(model.name)"
        >
          <span>{{ model.name }}</span>
          <svg v-if="selectedModel === model.name" class="check" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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
    </div>
  </div>
</template>

<style scoped>
.model-picker {
  position: relative;
}

.model-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  max-width: 220px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease, border-color 180ms ease;
}

.model-trigger:hover {
  background: var(--surface-hover);
  border-color: var(--border-strong);
  color: var(--text);
}

.model-trigger:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.model-trigger svg {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
}

.model-trigger-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-menu {
  position: absolute;
  bottom: calc(100% + 6px);
  right: 0;
  width: 240px;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  display: grid;
  gap: 2px;
  z-index: 30;
}

.model-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  transition: color 160ms ease, background-color 160ms ease;
}

.model-option:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.model-option.is-selected {
  color: var(--text);
  font-weight: 600;
}

.model-option .check {
  width: 12px;
  height: 12px;
  stroke: var(--positive);
}

.model-option.is-disabled {
  opacity: 0.55;
  cursor: not-allowed;
  background: var(--surface-raised);
}

.disabled-name {
  color: var(--subtle);
  font-size: 11px;
}

.unconfigured-tag {
  padding: 1px 5px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--danger) 16%, transparent);
  color: var(--danger);
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.model-menu-divider {
  padding: 6px 10px 4px;
  margin-top: 4px;
  border-top: 1px solid var(--border);
  color: var(--subtle);
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.model-empty-notice {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 10px;
  color: var(--muted);
  font-size: 11px;
}

.model-settings-link {
  color: var(--text);
  font-weight: 500;
  text-decoration: underline;
}
</style>

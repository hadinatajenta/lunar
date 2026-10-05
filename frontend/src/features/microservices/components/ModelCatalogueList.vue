<script setup lang="ts">
import { ref } from "vue"
import { isFreeModelId, type AiModelOption } from "./model-preferences"

const props = defineProps<{
  models: AiModelOption[]
  selectedModelId: string
  isDisabled: boolean
}>()

const emit = defineEmits<{
  (event: "select", modelId: string): void
}>()

const radiogroupElement = ref<HTMLElement | null>(null)

const resolveModelLabel = (model: AiModelOption): string =>
  isFreeModelId(model.id) ? `${model.name}, free` : model.name

const focusSelected = (): void => {
  const groupElement = radiogroupElement.value
  if (groupElement === null) {
    return
  }
  const selectedElement = groupElement.querySelector<HTMLInputElement>("input[type='radio']:checked")
  const firstElement = groupElement.querySelector<HTMLInputElement>("input[type='radio']")
  const targetElement = selectedElement ?? firstElement
  targetElement?.focus()
}

defineExpose({ focusSelected })
</script>

<template>
  <div
    v-if="props.models.length > 0"
    ref="radiogroupElement"
    class="model-list"
    role="radiogroup"
    aria-label="Choose AI model"
    :aria-disabled="props.isDisabled"
  >
    <label
      v-for="model in props.models"
      :key="model.id"
      class="model-option"
      :class="{ 'is-selected': model.id === props.selectedModelId, 'is-disabled': props.isDisabled }"
    >
      <input
        class="model-radio"
        type="radio"
        name="microservices-model"
        :value="model.id"
        :checked="model.id === props.selectedModelId"
        :disabled="props.isDisabled"
        :aria-label="resolveModelLabel(model)"
        @change="emit('select', model.id)"
      />
      <span class="model-option-name">{{ model.name }}</span>
      <span v-if="isFreeModelId(model.id)" class="model-option-badge" aria-hidden="true">Free</span>
    </label>
  </div>

  <button v-else class="model-option is-unavailable" type="button" disabled>
    No AI model available
  </button>
</template>

<style scoped>
.model-list { display: flex; flex-direction: column; gap: 8px; max-height: 262px; overflow-y: auto; }
.model-option {
  position: relative; display: flex; align-items: center; gap: 10px; width: 100%; min-height: 42px; padding: 0 12px;
  border: 1px solid var(--border); border-radius: 10px; background: var(--surface-raised); color: var(--text);
  font-size: 12.5px; font-weight: 500; text-align: left; cursor: pointer;
  transition: border-color 150ms ease, background-color 150ms ease;
}
.model-option:hover:not(.is-disabled):not(.is-unavailable) { border-color: var(--border-strong); }
.model-option.is-selected {
  border-color: color-mix(in srgb, var(--positive) 38%, transparent);
  background: color-mix(in srgb, var(--positive) 10%, transparent); color: var(--positive);
}
.model-option:focus-within { border-color: var(--border-strong); box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 22%, transparent); }
.model-option.is-disabled { opacity: 0.6; cursor: not-allowed; }
.model-option.is-unavailable { justify-content: center; color: var(--subtle); cursor: not-allowed; }
.model-radio { position: absolute; width: 1px; height: 1px; opacity: 0; }
.model-option-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.model-option-badge {
  display: inline-flex; align-items: center; padding: 2px 7px; border: 1px solid color-mix(in srgb, var(--positive) 32%, transparent);
  border-radius: 999px; background: color-mix(in srgb, var(--positive) 12%, transparent); color: var(--positive);
  font-size: 9.5px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase;
}
</style>

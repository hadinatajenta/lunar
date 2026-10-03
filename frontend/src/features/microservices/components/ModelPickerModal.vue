<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue"
import MicroservicesModal from "./MicroservicesModal.vue"
import { AI_MODEL_OPTIONS, findModelOption } from "./model-preferences"

const props = defineProps<{
  repositoryName: string
  selectedModelId: string
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "select", modelId: string): void
}>()

const modelSelect = ref<HTMLSelectElement | null>(null)

const selectedModel = computed(() => findModelOption(props.selectedModelId))
const selectedModelIcon = computed(() => selectedModel.value?.icon ?? "·")
const hasSelectedModel = computed(() => selectedModel.value !== null)

const handleModelChange = (event: Event) => {
  const selectElement = event.target as HTMLSelectElement
  if (selectElement.value.length === 0) {
    return
  }
  emit("select", selectElement.value)
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === "Escape") {
    emit("close")
  }
}

onMounted(() => {
  window.addEventListener("keydown", handleKeydown)
  modelSelect.value?.focus()
})

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <MicroservicesModal
    data-testid="model-picker-overlay"
    :eyebrow="props.repositoryName"
    title="Choose an AI model to export"
    title-id="model-picker-title"
    @close="emit('close')"
  >
    <p class="section-label">AI model</p>

    <div class="model-select-wrap">
      <span class="model-select-icon" aria-hidden="true">{{ selectedModelIcon }}</span>
      <select
        ref="modelSelect"
        class="model-select"
        aria-label="Choose AI model"
        :value="props.selectedModelId"
        @change="handleModelChange"
      >
        <option value="" disabled>Select an AI model…</option>
        <option v-for="model in AI_MODEL_OPTIONS" :key="model.id" :value="model.id">{{ model.name }}</option>
      </select>
      <span class="model-select-caret" aria-hidden="true">
        <svg viewBox="0 0 24 24"><path d="M6 9l6 6 6-6"></path></svg>
      </span>
    </div>

    <p class="model-hint" :class="{ 'is-active': hasSelectedModel }">
      {{ selectedModel?.description ?? "Pick a model to analyze routes and generate the collection." }}
    </p>

    <p v-if="hasSelectedModel" class="model-saved">
      Saved as the model preference for {{ props.repositoryName }}.
    </p>

    <template #footer>
      <span class="footer-note">Export collection is not available yet in this phase.</span>
      <span class="spacer"></span>
      <button class="footer-btn is-secondary" type="button" @click="emit('close')">Cancel</button>
      <button class="footer-btn is-ai" type="button" disabled aria-disabled="true">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
          <path d="M7 10l5 5 5-5"></path>
          <path d="M12 15V3"></path>
        </svg>
        Export collection
      </button>
    </template>
  </MicroservicesModal>
</template>

<style scoped>
.section-label {
  margin: 0 0 10px; color: var(--subtle); font-size: 10px; font-weight: 600; letter-spacing: 0.09em; text-transform: uppercase;
}
.model-select-wrap { position: relative; }
.model-select {
  width: 100%; height: 44px; padding: 0 40px 0 46px; border: 1px solid var(--border); border-radius: 10px; outline: 0;
  background: var(--surface-raised); color: var(--text); font-size: 12.5px; font-weight: 500; cursor: pointer;
  appearance: none; -webkit-appearance: none;
}
.model-select:hover, .model-select:focus { border-color: var(--border-strong); }
.model-select-icon {
  position: absolute; left: 12px; top: 50%; display: grid; place-items: center; width: 26px; height: 26px;
  transform: translateY(-50%); border: 1px solid var(--border-strong); border-radius: 7px; background: var(--surface);
  color: var(--muted); font-size: 10px; font-weight: 700; pointer-events: none;
}
.model-select-caret {
  position: absolute; right: 14px; top: 50%; width: 14px; height: 14px; transform: translateY(-50%);
  color: var(--muted); pointer-events: none;
}
.model-select-caret svg {
  width: 100%; height: 100%; stroke: currentColor; stroke-width: 2; fill: none; stroke-linecap: round; stroke-linejoin: round;
}
.model-hint { min-height: 16px; margin: 8px 0 0; color: var(--subtle); font-size: 10.5px; line-height: 1.55; }
.model-hint.is-active { color: var(--positive); }
.model-saved {
  margin: 12px 0 0; padding: 9px 11px; border: 1px solid color-mix(in srgb, var(--positive) 28%, transparent);
  border-radius: 8px; background: color-mix(in srgb, var(--positive) 9%, transparent); color: var(--positive); font-size: 11px;
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
  color: var(--positive); opacity: 0.45; cursor: not-allowed;
}
.footer-btn svg {
  width: 13px; height: 13px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round;
}

@media (max-width: 620px) {
  .spacer { display: none; }
  .footer-btn { width: 100%; }
  .footer-note { order: 3; }
}
</style>

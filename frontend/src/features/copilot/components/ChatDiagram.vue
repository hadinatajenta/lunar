<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useTheme } from "../../../composables/useTheme"
import { extractDiagramTitle, renderDiagram, toMermaidSource } from "../../../lib/diagram"

const props = defineProps<{
  source: string
  language?: string
  index?: number
  total?: number
}>()

const emit = defineEmits<{
  (event: "navigate", delta: number): void
}>()

const hasSiblings = computed(() => (props.total ?? 1) > 1)
const isFirst = computed(() => (props.index ?? 0) === 0)
const isLast = computed(() => (props.index ?? 0) >= (props.total ?? 1) - 1)
const positionLabel = computed(() => `${(props.index ?? 0) + 1} / ${props.total ?? 1}`)

const { isDark } = useTheme()

const title = computed(() => extractDiagramTitle(props.source, props.language ?? "mermaid"))
const svg = ref("")
const renderError = ref("")
const isRendering = ref(false)

const draw = async (): Promise<void> => {
  isRendering.value = true
  renderError.value = ""
  try {
    const mermaidSource = toMermaidSource(props.source, props.language ?? "mermaid")
    if (mermaidSource.length === 0) {
      renderError.value = "This block is not a diagram"
      return
    }
    svg.value = await renderDiagram(mermaidSource, isDark.value)
  } catch (thrownError) {
    svg.value = ""
    renderError.value = thrownError instanceof Error ? thrownError.message : "The diagram could not be rendered"
  } finally {
    isRendering.value = false
  }
}

onMounted(draw)

watch(() => props.source, draw)
watch(isDark, draw)
</script>

<template>
  <div class="diagram">
    <div class="diagram-head">
      <span class="diagram-tag" data-testid="diagram-tag">{{ title }}</span>

      <div v-if="hasSiblings" class="diagram-nav">
        <button
          type="button"
          class="diagram-nav-button"
          :disabled="isFirst"
          aria-label="Previous diagram"
          data-testid="diagram-previous"
          @click="emit('navigate', -1)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>

        <span class="diagram-counter" data-testid="diagram-counter" aria-live="polite">
          Diagram {{ positionLabel }}
        </span>

        <button
          type="button"
          class="diagram-nav-button"
          :disabled="isLast"
          aria-label="Next diagram"
          data-testid="diagram-next"
          @click="emit('navigate', 1)"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </button>
      </div>
    </div>

    <div v-if="isRendering" class="diagram-state" role="status" aria-live="polite">
      <span class="diagram-spinner" aria-hidden="true"></span>
      <span>Rendering diagram…</span>
    </div>

    <div v-else-if="renderError" class="diagram-state diagram-error" role="alert">
      <span>{{ renderError }}</span>
      <pre class="diagram-fallback">{{ props.source }}</pre>
    </div>

    <div v-else class="diagram-canvas" v-html="svg"></div>
  </div>
</template>

<style scoped>
.diagram {
  margin: 0 0 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  overflow: hidden;
}

.diagram-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
}

.diagram-tag {
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.01em;
}

.diagram-nav {
  display: flex;
  align-items: center;
  gap: 6px;
}

.diagram-nav-button {
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 5px;
  background: var(--surface-raised);
  color: var(--muted);
  cursor: pointer;
  transition: color 160ms ease, border-color 160ms ease;
}

.diagram-nav-button svg {
  width: 12px;
  height: 12px;
}

.diagram-nav-button:hover:not(:disabled) {
  color: var(--text);
  border-color: var(--border-strong);
}

.diagram-nav-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.diagram-nav-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}

.diagram-counter {
  color: var(--muted);
  font-size: 10.5px;
  font-variant-numeric: tabular-nums;
}

.diagram-canvas {
  display: flex;
  justify-content: center;
  padding: 14px;
  overflow-x: auto;
}

.diagram-canvas :deep(svg) {
  max-width: 100%;
  height: auto;
}

.diagram-state {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  color: var(--muted);
  font-size: 12px;
}

.diagram-error {
  color: var(--danger);
}

.diagram-spinner {
  width: 13px;
  height: 13px;
  border-radius: 50%;
  border: 2px solid var(--border-strong);
  border-top-color: var(--accent);
  animation: diagram-spin 700ms linear infinite;
}

.diagram-fallback {
  margin: 0;
  padding: 10px;
  border-radius: 6px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11.5px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

@keyframes diagram-spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .diagram-spinner {
    animation-duration: 2s;
  }
}
</style>

<script setup lang="ts">
import type { CollectionExportStage, CollectionExportStageEvent } from "../types"

const props = defineProps<{
  stageEvents: CollectionExportStageEvent[]
  isExporting: boolean
}>()

const STAGE_LABELS: Record<CollectionExportStage, string> = {
  selecting: "Selecting the service files",
  prompt: "Building the analysis prompt",
  model: "Asking the AI model",
  validating: "Validating the collection"
}
</script>

<template>
  <div v-if="props.stageEvents.length > 0" class="stage-panel" role="status" :aria-busy="props.isExporting">
    <ol class="stage-list" aria-label="Export progress">
      <li
        v-for="(stageEvent, stageIndex) in props.stageEvents"
        :key="`${stageEvent.stage}-${stageIndex}`"
        class="stage-row"
        :class="{ 'is-current': props.isExporting && stageIndex === props.stageEvents.length - 1 }"
      >
        <span class="stage-mark" aria-hidden="true"></span>
        <span class="stage-body">
          <span class="stage-name">{{ STAGE_LABELS[stageEvent.stage] }}</span>
          <span v-if="stageEvent.detail.length > 0" class="stage-detail">{{ stageEvent.detail }}</span>
        </span>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.stage-panel { margin-top: 14px; padding: 12px 14px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface-raised); }
.stage-list { display: flex; flex-direction: column; gap: 10px; margin: 0; padding: 0; list-style: none; }
.stage-row { display: flex; align-items: flex-start; gap: 10px; }
.stage-mark {
  width: 7px; height: 7px; flex: 0 0 auto; margin-top: 4px; border-radius: 50%; background: var(--positive);
}
.stage-row.is-current .stage-mark { animation: stage-pulse 1.1s ease-in-out infinite; }
.stage-body { min-width: 0; }
.stage-name { display: block; color: var(--text); font-size: 11.5px; font-weight: 600; }
.stage-detail { display: block; margin-top: 2px; color: var(--subtle); font-size: 10.5px; line-height: 1.5; overflow-wrap: anywhere; }
@keyframes stage-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.35; } }
@media (prefers-reduced-motion: reduce) {
  .stage-row.is-current .stage-mark { animation: none; }
}
</style>

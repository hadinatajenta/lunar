<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useCodeAgent } from "../../code-context/composables/useCodeAgent"

const INDEX_POLL_INTERVAL_MS = 4000

const {
  indexStatus,
  indexStatusError,
  indexHint,
  isIndexStatusLoading,
  isIndexBuildStarting,
  isIndexBuilding,
  refreshIndexStatus,
  startIndexBuild
} = useCodeAgent()

const isPanelOpen = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const stopPolling = (): void => {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const startPolling = (): void => {
  stopPolling()
  pollTimer = setInterval(() => {
    void refreshIndexStatus()
  }, INDEX_POLL_INTERVAL_MS)
}

const shouldPoll = computed(() => isPanelOpen.value || isIndexBuilding.value)

watch(shouldPoll, (value) => {
  if (value) startPolling()
  else stopPolling()
}, { immediate: true })

onMounted(() => {
  void refreshIndexStatus()
})

onBeforeUnmount(stopPolling)

const progressPercent = computed(() => {
  const rawPercent = Math.round(indexStatus.value?.progress_percent ?? 0)
  return Math.min(Math.max(rawPercent, 0), 100)
})

const statusTone = computed<"ready" | "building" | "missing">(() => {
  if (isIndexBuilding.value) return "building"
  const status = indexStatus.value
  if (status !== null && status.indexed && status.chunk_count > 0) return "ready"
  return "missing"
})

const statusLabel = computed(() => {
  if (isIndexBuilding.value) return `Indexing ${progressPercent.value}%`
  if (isIndexStatusLoading.value && indexStatus.value === null) return "Checking code index"
  if (statusTone.value === "ready") return "Code index ready"
  if (indexStatusError.value !== null) return "Code index unavailable"
  return "Code index not built"
})

const updatedAtLabel = computed(() => {
  const updatedAt = indexStatus.value?.updated_at ?? ""
  if (updatedAt.length === 0) return "Never"
  const parsedDate = new Date(updatedAt)
  return Number.isNaN(parsedDate.getTime()) ? updatedAt : parsedDate.toLocaleString()
})

const repoCountLabel = computed(() => (indexStatus.value?.repo_count ?? 0).toLocaleString())
const fileCountLabel = computed(() => (indexStatus.value?.file_count ?? 0).toLocaleString())
const chunkCountLabel = computed(() => (indexStatus.value?.chunk_count ?? 0).toLocaleString())

const handleBuild = async (): Promise<void> => {
  await startIndexBuild()
}
</script>

<template>
  <div class="index-status-wrap">
    <button
      class="index-chip"
      :class="statusTone"
      type="button"
      data-testid="code-index-status"
      :aria-expanded="isPanelOpen"
      @click="isPanelOpen = !isPanelOpen"
    >
      <span class="index-dot" aria-hidden="true"></span>
      <span class="index-chip-label">{{ statusLabel }}</span>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="m6 15 6-6 6 6"></path>
      </svg>
    </button>

    <div v-if="isPanelOpen" class="index-panel">
      <div class="panel-head">
        <span class="panel-title">Code index</span>
        <span class="panel-state" :class="statusTone">{{ statusLabel }}</span>
      </div>

      <div
        v-if="isIndexBuilding"
        class="progress-track"
        role="progressbar"
        :aria-valuenow="progressPercent"
        aria-valuemin="0"
        aria-valuemax="100"
      >
        <div class="progress-fill" :style="{ width: `${progressPercent}%` }"></div>
      </div>

      <dl v-if="indexStatus" class="metric-grid">
        <div class="metric">
          <dt>Repositories</dt>
          <dd>{{ repoCountLabel }}</dd>
        </div>
        <div class="metric">
          <dt>Files</dt>
          <dd>{{ fileCountLabel }}</dd>
        </div>
        <div class="metric">
          <dt>Chunks</dt>
          <dd>{{ chunkCountLabel }}</dd>
        </div>
        <div class="metric">
          <dt>Updated</dt>
          <dd>{{ updatedAtLabel }}</dd>
        </div>
      </dl>

      <p v-if="indexStatusError" class="panel-message error" role="alert">{{ indexStatusError }}</p>
      <p v-else-if="indexHint" class="panel-message hint">{{ indexHint }}</p>

      <button
        class="build-btn"
        type="button"
        data-testid="code-index-build"
        :disabled="isIndexBuildStarting || isIndexBuilding"
        @click="handleBuild"
      >
        {{ isIndexBuilding ? "Build in progress" : isIndexBuildStarting ? "Starting build" : "Build index" }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.index-status-wrap { position: relative; }

.index-chip {
  display: flex; align-items: center; gap: 6px; height: 28px; padding: 0 10px;
  border: 1px solid var(--border); border-radius: 6px; background: var(--surface-raised);
  color: var(--muted); font-size: 11px; font-weight: 500; white-space: nowrap; cursor: pointer;
  transition: background-color 180ms ease, color 180ms ease;
}

.index-chip:hover { background: var(--surface-hover); color: var(--text); }
.index-chip svg { width: 12px; height: 12px; }
.index-chip.ready { color: var(--positive); border-color: color-mix(in srgb, var(--positive) 40%, transparent); }
.index-chip.building { color: #d9a441; border-color: rgba(217, 164, 65, 0.4); }
.index-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--subtle); }
.index-chip.ready .index-dot { background: var(--positive); }
.index-chip.building .index-dot { background: #d9a441; }

.index-panel {
  position: absolute; top: calc(100% + 6px); right: 0; width: 280px; padding: 14px;
  border: 1px solid var(--border); border-radius: 10px; background: var(--surface);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15); z-index: 30;
}

.panel-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.panel-title { color: var(--text); font-size: 12px; font-weight: 600; }
.panel-state { color: var(--muted); font-size: 10px; text-transform: uppercase; letter-spacing: 0.06em; }
.panel-state.ready { color: var(--positive); }
.panel-state.building { color: #d9a441; }

.progress-track { height: 4px; margin-top: 10px; border-radius: 999px; background: var(--surface-raised); overflow: hidden; }
.progress-fill { height: 100%; background: var(--positive); transition: width 200ms cubic-bezier(0.16, 1, 0.3, 1); }

.metric-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 12px; margin: 12px 0 0; }
.metric dt { color: var(--subtle); font-size: 10px; text-transform: uppercase; letter-spacing: 0.06em; }
.metric dd { margin: 2px 0 0; color: var(--text); font-size: 12px; font-variant-numeric: tabular-nums; }
.panel-message { margin: 12px 0 0; font-size: 11px; line-height: 1.5; }
.panel-message.hint { color: var(--muted); }
.panel-message.error { color: #f87171; }

.build-btn {
  display: inline-flex; align-items: center; justify-content: center; width: 100%; height: 32px;
  margin-top: 12px; border: 0; border-radius: 8px; background: var(--accent); color: var(--accent-contrast);
  font-size: 12px; font-weight: 600; white-space: nowrap; cursor: pointer; transition: opacity 180ms ease;
}

.build-btn:hover:not(:disabled) { opacity: 0.9; }
.build-btn:disabled { opacity: 0.4; cursor: not-allowed; }
</style>

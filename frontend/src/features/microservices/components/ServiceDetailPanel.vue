<script setup lang="ts">
import { computed } from "vue"
import type { RepositoryDetail } from "../types"
import ServiceFileList from "./ServiceFileList.vue"
import { formatChangedFileCount, resolveShortCommitHash } from "./service-presentation"

const props = defineProps<{
  service: RepositoryDetail
  modelLabel: string
}>()

const emit = defineEmits<{
  (event: "open-model", repositoryName: string): void
}>()

const commitHash = computed(() => resolveShortCommitHash(props.service.commit.hash))
</script>

<template>
  <div class="expand-panel">
    <div class="expand-grid">
      <div class="expand-block">
        <div class="block-head">
          <span class="block-title">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <rect x="4" y="3" width="16" height="18" rx="2"></rect>
              <path d="M9 3v18"></path>
              <path d="M15 3v18"></path>
            </svg>
            Files changed
          </span>
          <span class="block-count">{{ formatChangedFileCount(props.service.files.length) }}</span>
        </div>

        <ServiceFileList :files="props.service.files" :is-truncated="props.service.files_truncated" />
      </div>

      <div class="expand-block">
        <div class="block-head">
          <span class="block-title">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M3 7l9-4 9 4-9 4z"></path>
              <path d="M3 12l9 4 9-4"></path>
              <path d="M3 17l9 4 9-4"></path>
            </svg>
            Repository
          </span>
          <span class="block-count">{{ props.service.name }}</span>
        </div>

        <div class="repo-list">
          <div class="repo-item">
            <span class="repo-item-icon">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="12" cy="12" r="3"></circle>
                <path d="M12 3v3"></path>
                <path d="M12 18v3"></path>
              </svg>
            </span>
            <div class="repo-item-body">
              <div class="repo-item-label">Last commit</div>
              <div class="repo-item-value">{{ props.service.commit.subject || "No commit recorded" }}</div>
              <div v-if="props.service.commit.hash" class="repo-item-sub mono">
                {{ commitHash }} · {{ props.service.commit.author || "Unknown author" }} ·
                {{ props.service.commit.relative_time || "Unknown time" }}
              </div>
            </div>
          </div>

          <div class="repo-item">
            <span class="repo-item-icon">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="6" cy="6" r="2.5"></circle>
                <circle cx="6" cy="18" r="2.5"></circle>
                <circle cx="18" cy="12" r="2.5"></circle>
                <path d="M6 8.5v7"></path>
                <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
              </svg>
            </span>
            <div class="repo-item-body">
              <div class="repo-item-label">Branch</div>
              <div class="repo-item-value mono">{{ props.service.branch || "No branch" }}</div>
              <div class="repo-item-sub">Ahead {{ props.service.ahead }} · Behind {{ props.service.behind }}</div>
            </div>
          </div>

          <div class="repo-item">
            <span class="repo-item-icon">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <circle cx="12" cy="12" r="9"></circle>
                <path d="M12 7v5l3 2"></path>
              </svg>
            </span>
            <div class="repo-item-body">
              <div class="repo-item-label">Last updated</div>
              <div class="repo-item-value">{{ props.service.updated_relative || "Unknown" }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="export-bar">
      <div class="export-bar-text">
        <strong>Ready to export this service as a Postman collection?</strong>
        <span>
          Pick an AI model to analyze routes, then generate the collection.
          <template v-if="props.modelLabel">Saved model for this repository: {{ props.modelLabel }}.</template>
        </span>
      </div>
      <button
        class="export-btn"
        type="button"
        :aria-label="`Choose an AI model for ${props.service.name}`"
        @click="emit('open-model', props.service.name)"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="3.5"></circle>
          <path d="M12 3v3"></path>
          <path d="M12 18v3"></path>
          <path d="m5.6 5.6 2.1 2.1"></path>
          <path d="m16.3 16.3 2.1 2.1"></path>
          <path d="M3 12h3"></path>
          <path d="M18 12h3"></path>
          <path d="m5.6 18.4 2.1-2.1"></path>
          <path d="m16.3 7.7 2.1-2.1"></path>
        </svg>
        Choose AI model
      </button>
    </div>
  </div>
</template>

<style scoped>
.expand-panel { display: flex; flex-direction: column; gap: 18px; padding: 20px 22px 18px; }
.expand-grid { display: grid; grid-template-columns: minmax(0, 1.35fr) minmax(0, 1fr); gap: 16px; }
.expand-block {
  min-width: 0; padding: 14px 16px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface);
}

.block-head {
  display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 12px;
  padding-bottom: 10px; border-bottom: 1px solid var(--border);
}
.block-title {
  display: flex; align-items: center; gap: 8px; color: var(--text); font-size: 11px; font-weight: 600;
  letter-spacing: 0.08em; text-transform: uppercase;
}
.block-title svg {
  width: 12px; height: 12px; stroke: var(--muted); stroke-width: 2; fill: none; stroke-linecap: round; stroke-linejoin: round;
}
.block-count {
  overflow: hidden; color: var(--subtle); font-size: 10.5px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap;
}

.repo-list { display: grid; gap: 10px; }
.repo-item {
  display: flex; align-items: flex-start; gap: 10px; padding: 8px 0; border-bottom: 1px dashed var(--border);
}
.repo-item:last-child { padding-bottom: 0; border-bottom: 0; }
.repo-item-icon {
  display: grid; place-items: center; width: 22px; height: 22px; flex: 0 0 auto; border: 1px solid var(--border);
  border-radius: 6px; background: var(--surface-raised); color: var(--muted);
}
.repo-item-icon svg {
  width: 11px; height: 11px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round;
}
.repo-item-body { min-width: 0; flex: 1; }
.repo-item-label { color: var(--subtle); font-size: 10px; font-weight: 600; letter-spacing: 0.08em; text-transform: uppercase; }
.repo-item-value {
  margin-top: 3px; color: var(--text); font-size: 12px; font-weight: 500; line-height: 1.45; overflow-wrap: anywhere;
}
.repo-item-value.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; }
.repo-item-sub { margin-top: 3px; color: var(--subtle); font-size: 10.5px; }
.repo-item-sub.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 10px; }

.export-bar {
  display: flex; align-items: center; justify-content: space-between; gap: 14px; flex-wrap: wrap; padding: 14px 16px;
  border: 1px dashed var(--border-strong); border-radius: 10px; background: color-mix(in srgb, var(--positive) 6%, transparent);
}
.export-bar-text { min-width: 0; }
.export-bar-text strong { display: block; margin-bottom: 3px; color: var(--text); font-size: 12px; font-weight: 500; }
.export-bar-text span { color: var(--subtle); font-size: 10.5px; line-height: 1.5; }
.export-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 0 12px;
  border: 1px solid color-mix(in srgb, var(--positive) 32%, transparent); border-radius: 8px;
  background: color-mix(in srgb, var(--positive) 10%, transparent); color: var(--positive); font-size: 11.5px;
  font-weight: 500; white-space: nowrap; cursor: pointer; transition: background-color 160ms ease;
}
.export-btn:hover { background: color-mix(in srgb, var(--positive) 18%, transparent); }
.export-btn svg {
  width: 13px; height: 13px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round;
}

@media (max-width: 1100px) {
  .expand-grid { grid-template-columns: 1fr; }
}
@media (max-width: 620px) {
  .expand-panel { padding: 16px; }
  .export-bar { flex-direction: column; align-items: stretch; }
  .export-btn { width: 100%; }
}
</style>

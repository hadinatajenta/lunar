<script setup lang="ts">
import { computed } from "vue"
import type { RepositoryDetail } from "../types"
import ServiceDetailPanel from "./ServiceDetailPanel.vue"
import {
  countChangedFiles,
  formatDirtyLabel,
  resolveDirtyLevel,
  resolveRepositoryInitials,
  resolveShortCommitHash
} from "./service-presentation"

const props = defineProps<{
  service: RepositoryDetail
  isOpen: boolean
  modelLabel: string
}>()

const emit = defineEmits<{
  (event: "toggle"): void
  (event: "open-model", repositoryName: string): void
}>()

const changedFileCount = computed(() => countChangedFiles(props.service))
const dirtyLevel = computed(() => resolveDirtyLevel(changedFileCount.value))
const dirtyLabel = computed(() => formatDirtyLabel(changedFileCount.value))
const initials = computed(() => resolveRepositoryInitials(props.service.name))
const commitHash = computed(() => resolveShortCommitHash(props.service.commit.hash))
</script>

<template>
  <tr v-if="props.service.error" class="service-row is-error">
    <td>
      <div class="svc-name">
        <span class="svc-mark">{{ initials }}</span>
        <div class="svc-text">
          <div class="svc-title">{{ props.service.name }}</div>
        </div>
      </div>
    </td>
    <td class="row-error" colspan="3">{{ props.service.error }}</td>
    <td class="col-action"></td>
  </tr>

  <template v-else>
    <tr class="service-row" :class="{ 'is-open': props.isOpen }" @click="emit('toggle')">
      <td>
        <div class="svc-name">
          <span class="svc-mark">{{ initials }}</span>
          <div class="svc-text">
            <div class="svc-title">{{ props.service.name }}</div>
            <div v-if="props.service.commit.hash" class="svc-sub">{{ commitHash }}</div>
          </div>
        </div>
      </td>

      <td class="col-branch">
        <span v-if="props.service.branch" class="branch-pill">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="6" cy="6" r="2.5"></circle>
            <circle cx="6" cy="18" r="2.5"></circle>
            <circle cx="18" cy="12" r="2.5"></circle>
            <path d="M6 8.5v7"></path>
            <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
          </svg>
          {{ props.service.branch }}
        </span>
        <span v-else class="muted-value">No branch</span>
      </td>

      <td>
        <span class="dirty-count" :class="`is-${dirtyLevel}`">{{ dirtyLabel }}</span>
      </td>

      <td class="col-updated">
        <span class="updated-value">{{ props.service.updated_relative || "Unknown" }}</span>
      </td>

      <td class="col-action">
        <button
          class="expand-btn"
          type="button"
          :aria-expanded="props.isOpen"
          :aria-label="props.isOpen ? `Collapse ${props.service.name}` : `Expand ${props.service.name}`"
          @click.stop="emit('toggle')"
        >
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 9l6 6 6-6"></path>
          </svg>
        </button>
      </td>
    </tr>

    <tr v-if="props.isOpen" class="expand-row">
      <td colspan="5">
        <ServiceDetailPanel
          :service="props.service"
          :model-label="props.modelLabel"
          @open-model="emit('open-model', $event)"
        />
      </td>
    </tr>
  </template>
</template>

<style scoped>
.service-row { cursor: pointer; transition: background-color 150ms ease; }
.service-row:hover { background: var(--surface-hover); }
.service-row.is-open { background: var(--surface-raised); }
.service-row.is-error { cursor: default; }
.service-row td { padding: 14px 18px; border-bottom: 1px solid var(--border); color: var(--text); font-size: 12px; vertical-align: middle; }
.service-row td.row-error { color: var(--danger); font-size: 11.5px; }

.svc-name { display: flex; align-items: center; gap: 11px; min-width: 0; }
.svc-mark {
  display: grid; place-items: center; width: 30px; height: 30px; flex: 0 0 auto; border: 1px solid var(--border-strong);
  border-radius: 8px; background: var(--surface-raised); color: var(--muted); font-size: 10px; font-weight: 700;
}
.svc-text { min-width: 0; }
.svc-title { overflow: hidden; color: var(--text); font-size: 12.5px; font-weight: 500; text-overflow: ellipsis; white-space: nowrap; }
.svc-sub { margin-top: 3px; color: var(--subtle); font-size: 10px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }

.branch-pill {
  display: inline-flex; align-items: center; gap: 5px; max-width: 100%; padding: 3px 7px; border: 1px solid var(--border);
  border-radius: 6px; background: var(--surface-raised); color: var(--muted); font-size: 10.5px; overflow-wrap: anywhere;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.branch-pill svg {
  width: 10px; height: 10px; flex: 0 0 auto; stroke: currentColor; stroke-width: 2; fill: none;
  stroke-linecap: round; stroke-linejoin: round;
}

.muted-value, .updated-value { color: var(--subtle); font-size: 11px; }

.dirty-count {
  display: inline-flex; align-items: center; gap: 6px; padding: 3px 9px; border: 1px solid var(--border);
  border-radius: 999px; font-size: 10.5px; font-weight: 600; font-variant-numeric: tabular-nums; white-space: nowrap;
}
.dirty-count::before { content: ""; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
.dirty-count.is-clean { color: var(--muted); background: var(--surface-raised); }
.dirty-count.is-dirty {
  color: var(--warning); border-color: color-mix(in srgb, var(--warning) 30%, transparent);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
}
.dirty-count.is-heavy {
  color: var(--danger); border-color: color-mix(in srgb, var(--danger) 30%, transparent);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
}

.col-action { width: 44px; text-align: right; }
.expand-btn {
  display: grid; place-items: center; width: 30px; height: 30px; margin-left: auto; border: 1px solid var(--border);
  border-radius: 8px; background: transparent; color: var(--muted); cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease, border-color 150ms ease;
}
.expand-btn:hover { border-color: var(--border-strong); background: var(--surface-hover); color: var(--text); }
.expand-btn svg {
  width: 13px; height: 13px; stroke: currentColor; stroke-width: 2; fill: none; stroke-linecap: round;
  stroke-linejoin: round; transition: transform 200ms ease;
}
.service-row.is-open .expand-btn svg { transform: rotate(180deg); }

.expand-row td { padding: 0; border-bottom: 1px solid var(--border); background: var(--surface); }

@media (max-width: 820px) {
  .col-branch { display: none; }
}
@media (max-width: 620px) {
  .col-updated { display: none; }
}
</style>

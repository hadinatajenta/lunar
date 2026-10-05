<script setup lang="ts">
import { computed } from "vue"
import type { RepositoryDetail } from "../types"
import BranchPicker from "./BranchPicker.vue"
import RepositoryActionsMenu from "./RepositoryActionsMenu.vue"
import ServiceDetailPanel from "./ServiceDetailPanel.vue"
import { useRepositoryActions } from "../composables/useRepositoryActions"
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

const { resolveActionState } = useRepositoryActions()
const actionState = computed(() => resolveActionState(props.service.name))
const isBusy = computed(() => actionState.value.status === "running")
const actionFailureMessage = computed(() =>
  actionState.value.status === "failed" ? actionState.value.message : ""
)
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
    <tr
      class="service-row"
      :class="{ 'is-open': props.isOpen, 'is-busy': isBusy }"
      @click="emit('toggle')"
    >
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
        <BranchPicker
          v-if="props.service.branch"
          :repository-name="props.service.name"
          :branch="props.service.branch"
        />
        <span v-else class="muted-value">No branch</span>
      </td>

      <td>
        <span class="dirty-count" :class="`is-${dirtyLevel}`">{{ dirtyLabel }}</span>
      </td>

      <td class="col-updated">
        <span class="updated-value">{{ props.service.updated_relative || "Unknown" }}</span>
      </td>

      <td class="col-action">
        <RepositoryActionsMenu
          :repository-name="props.service.name"
          :is-details-open="props.isOpen"
          :is-busy="isBusy"
          :failure-message="actionFailureMessage"
          @toggle-details="emit('toggle')"
        />
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
.service-row.is-busy { background: color-mix(in srgb, var(--accent) 6%, transparent); }

.expand-row td { padding: 0; border-bottom: 1px solid var(--border); background: var(--surface); }

@media (max-width: 820px) {
  .col-branch { display: none; }
}
@media (max-width: 620px) {
  .col-updated { display: none; }
}
</style>

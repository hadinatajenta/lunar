<script setup lang="ts">
import { computed } from "vue"
import type { IssueBranchMatch, WorkspaceScope } from "../branch-types"
import { branchRows, emptyBranchMessage, remoteNoticeText, remoteOnlyRepositories } from "../logic/branch-groups"

const props = defineProps<{
  issueKey: string
  matches: IssueBranchMatch[]
  scope: WorkspaceScope | null
  isLoading: boolean
  errorMessage: string
}>()

const emit = defineEmits<{
  (event: "retry"): void
}>()

const rows = computed(() => branchRows(props.matches))
const remoteNotice = computed(() => remoteNoticeText(remoteOnlyRepositories(props.matches)))
const emptyMessage = computed(() => {
  if (props.scope === null) {
    return ""
  }
  return emptyBranchMessage(props.issueKey, props.scope)
})
</script>

<template>
  <section class="branch-list" data-testid="issue-branch-list">
    <h2 class="branch-list-title">Branches</h2>

    <p v-if="isLoading" class="branch-list-status">Loading branches…</p>

    <div v-else-if="errorMessage.length > 0" class="branch-list-status is-failed">
      <span>{{ errorMessage }}</span>
      <button class="link-btn" type="button" data-testid="branch-list-retry" @click="emit('retry')">
        Retry
      </button>
    </div>

    <template v-else>
      <p v-if="rows.length === 0 && emptyMessage.length > 0" class="empty" data-testid="branch-list-empty">
        {{ emptyMessage }}
      </p>

      <div
        v-for="row in rows"
        :key="`${row.repo}/${row.branch}`"
        class="branch"
        :class="{ 'is-remote': row.source === 'remote' }"
        data-testid="issue-branch-row"
      >
        <span class="branch-name">{{ row.branch }}</span>
        <span class="branch-where">in {{ row.repo }}</span>
        <span class="branch-when">{{ row.lastTouched }}</span>
      </div>

      <p v-if="remoteNotice.length > 0" class="notice" data-testid="branch-remote-notice">
        {{ remoteNotice }}
      </p>
    </template>
  </section>
</template>

<style scoped src="./issue-branch-list.css"></style>

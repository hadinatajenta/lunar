<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import PageLayout from "../../../components/layout/PageLayout.vue"
import MicroservicesBoard from "../components/MicroservicesBoard.vue"
import ServiceTableSkeleton from "../components/ServiceTableSkeleton.vue"
import WorkspaceStatePanel from "../components/WorkspaceStatePanel.vue"
import { useMicroservices } from "../composables/useMicroservices"
import type { ValidatePathResponse } from "../types"

type MicroservicesPageState =
  | "loading"
  | "no-workspace"
  | "helper-missing"
  | "folder-missing"
  | "empty-folder"
  | "no-selection"
  | "failed"
  | "ready"

const {
  workspace,
  repositories,
  selectedNames,
  error,
  helperStatus,
  hasHelperToken,
  loadWorkspace,
  loadRepositories,
  loadServices,
  validateRootPath
} = useMicroservices()

const isBootstrapping = ref(true)
const rootValidation = ref<ValidatePathResponse | null>(null)

const initialize = async () => {
  isBootstrapping.value = true
  rootValidation.value = null

  const settings = await loadWorkspace()
  if (settings !== null && settings.root_path.trim().length > 0) {
    if (settings.source === "helper") {
      const validation = await validateRootPath(settings.root_path)
      rootValidation.value = validation
      if (!validation.valid) {
        isBootstrapping.value = false
        return
      }
    }
    await loadRepositories()
    await loadServices()
  }

  isBootstrapping.value = false
}

const pageState = computed<MicroservicesPageState>(() => {
  if (isBootstrapping.value) return "loading"

  const settings = workspace.value
  if (settings === null || settings.root_path.trim().length === 0) {
    return error.value === null ? "no-workspace" : "failed"
  }

  if (settings.source === "helper") {
    if (!hasHelperToken.value || helperStatus.value === "offline") return "helper-missing"
    if (rootValidation.value !== null && !rootValidation.value.valid) return "folder-missing"
  }

  if (repositories.value.length === 0 && error.value !== null) return "failed"
  if (repositories.value.length === 0) return "empty-folder"
  if (selectedNames.value.length === 0) return "no-selection"
  return "ready"
})

const folderMissingMessage = computed(() => {
  const reason = rootValidation.value?.reason
  if (reason !== undefined && reason.length > 0) return reason
  return "The Lunar helper could not find the repository root folder on this machine. Check the folder path in Settings."
})

const emptyFolderMessage = computed(() => {
  const rootPath = workspace.value?.root_path ?? ""
  if (rootPath.length === 0) return "No repositories were found in the configured folder."
  return `No repositories were found in ${rootPath}.`
})

const retry = async () => {
  await initialize()
}

onMounted(() => {
  void initialize()
})
</script>

<template>
  <PageLayout>
    <div class="content" data-testid="microservices-page">
      <header class="page-header">
        <p class="eyebrow">Service registry</p>
        <h1>Microservices BRI</h1>
        <p class="page-copy">
          Every service, its repository state, and the files currently in flight. Expand a row to inspect what is dirty before you ship.
        </p>
      </header>

      <ServiceTableSkeleton v-if="pageState === 'loading'" />

      <WorkspaceStatePanel
        v-else-if="pageState === 'no-workspace'"
        title="No workspace configured"
        message="Choose where Lunar reads your repositories and set the folder that holds them. Nothing is read until you do."
        action-label="Open Settings"
        action-to="/settings"
      />

      <WorkspaceStatePanel
        v-else-if="pageState === 'helper-missing'"
        tone="warning"
        title="Lunar helper not detected"
        message="Start the Lunar helper on this machine, then confirm the helper URL and token in Settings. The helper reads git metadata from your laptop only, and file contents are never read."
        action-label="Open Settings"
        action-to="/settings"
        secondary-action-label="Check again"
        @secondary-action="retry"
      />

      <WorkspaceStatePanel
        v-else-if="pageState === 'folder-missing'"
        tone="danger"
        title="Repository folder not found"
        :message="folderMissingMessage"
        action-label="Fix the folder in Settings"
        action-to="/settings"
        secondary-action-label="Check again"
        @secondary-action="retry"
      />

      <WorkspaceStatePanel
        v-else-if="pageState === 'empty-folder'"
        title="No repositories found"
        :message="emptyFolderMessage"
        action-label="Change the folder in Settings"
        action-to="/settings"
        secondary-action-label="Check again"
        @secondary-action="retry"
      />

      <WorkspaceStatePanel
        v-else-if="pageState === 'failed'"
        tone="danger"
        title="Could not read the repositories"
        :message="error ?? 'The request failed without a message.'"
        action-label="Retry"
        secondary-action-label="Open Settings"
        secondary-action-to="/settings"
        @action="retry"
      />

      <MicroservicesBoard v-else />
    </div>
  </PageLayout>
</template>

<style scoped>
.content { max-width: 1320px; margin: 0 auto; padding: 36px 30px 52px; }
.page-header { margin-bottom: 26px; }
.eyebrow {
  margin: 0 0 10px; color: var(--muted); font-size: 11px; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase;
}
h1 { margin: 0; color: var(--text); font-size: clamp(30px, 4vw, 46px); font-weight: 600; line-height: 1; letter-spacing: -0.05em; }
.page-copy { max-width: 620px; margin: 11px 0 0; color: var(--muted); font-size: 13px; line-height: 1.6; }

@media (max-width: 820px) {
  .content { padding: 26px 20px 42px; }
}
@media (max-width: 420px) {
  .content { padding: 20px 14px 36px; }
}
</style>

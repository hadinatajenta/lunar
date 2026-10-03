<script setup lang="ts">
import { onMounted, ref, watch } from "vue"
import HelperConnectionFields from "../../microservices/components/HelperConnectionFields.vue"
import HelperSetupGuide from "../../microservices/components/HelperSetupGuide.vue"
import { useMicroservices } from "../../microservices/composables/useMicroservices"
import { useHelperConnection } from "../composables/useHelperConnection"
import type { WorkspaceSource } from "../../microservices/types"

const {
  workspace,
  isSaving,
  error,
  helperStatus,
  loadWorkspace,
  saveWorkspace,
  clearError
} = useMicroservices()

const helperStatusTone = helperStatus

const {
  helperUrl,
  helperToken,
  isCheckingHelper,
  isFolderChecking,
  isFolderValid,
  folderValidationMessage,
  helperTestMessage,
  isHelperOnline,
  isTokenSaved,
  tokenPlaceholder,
  tokenStatusMessage,
  handleTestConnection,
  handleConnectHelper,
  handleCheckHelper,
  handleValidateFolder,
  resetFolderValidation
} = useHelperConnection()

const source = ref<WorkspaceSource>("helper")
const rootPath = ref("")
const saveMessage = ref("")

const selectSource = (nextSource: WorkspaceSource) => {
  source.value = nextSource
  clearError()
  resetFolderValidation()
  helperTestMessage.value = ""
  saveMessage.value = ""
}

const handleSave = async () => {
  clearError()
  saveMessage.value = ""
  resetFolderValidation()

  const trimmedToken = helperToken.value.trim()
  const isSaved = await saveWorkspace({
    source: source.value,
    root_path: rootPath.value.trim(),
    helper_url: helperUrl.value.trim(),
    helper_token: trimmedToken.length > 0 ? trimmedToken : undefined
  })

  if (!isSaved) {
    return
  }

  helperToken.value = ""
  saveMessage.value = "Workspace settings saved. The helper token is encrypted at rest and is never shown again."
}

watch(
  workspace,
  (settings) => {
    if (settings === null) return
    source.value = settings.source
    rootPath.value = settings.root_path
    helperUrl.value = settings.helper_url
  },
  { immediate: true }
)

onMounted(async () => {
  await loadWorkspace()
})
</script>

<template>
  <div class="workspace-settings">
    <div v-if="saveMessage" class="toast-banner success">{{ saveMessage }}</div>
    <div v-if="error" class="toast-banner error">{{ error }}</div>

    <div class="settings-card">
      <div class="settings-card-header">
        <div class="settings-card-title">Microservices repository source</div>
        <div class="settings-card-description">
          Lunar reads git metadata only: branch, changed files, and the last commit. File contents are never read.
        </div>
      </div>

      <div class="settings-card-body">
        <div class="field-wrap">
          <span class="field-label">Source</span>
          <div class="source-choice" role="radiogroup" aria-label="Repository source">
            <button class="source-option" :class="{ 'is-active': source === 'helper' }" type="button"
              role="radio" :aria-checked="source === 'helper'" @click="selectSource('helper')">
              <span class="source-name">Local helper</span>
              <span class="source-note">Reads repositories on this machine. Recommended default.</span>
            </button>
            <button class="source-option" :class="{ 'is-active': source === 'server' }" type="button"
              role="radio" :aria-checked="source === 'server'" @click="selectSource('server')">
              <span class="source-name">Lunar backend</span>
              <span class="source-note">Reads repositories on the machine that runs the backend.</span>
            </button>
          </div>
        </div>

        <div v-if="source === 'server'" class="source-warning">
          The backend reads this path on its own machine. When the backend runs on a server and your repositories are
          on your laptop, the backend cannot see them. Keep the local helper as the source for laptop repositories.
        </div>

        <div class="field-wrap">
          <label class="field-label" for="workspace-root-path">Repository root folder</label>
          <div class="field-row">
            <input id="workspace-root-path" v-model="rootPath" class="field-input" type="text"
              autocomplete="off" placeholder="/Users/you/repositories" />
            <button v-if="source === 'helper'" class="field-btn" type="button"
              :disabled="isFolderChecking || rootPath.trim().length === 0" @click="handleValidateFolder(rootPath)">
              {{ isFolderChecking ? "Checking…" : "Validate folder" }}
            </button>
          </div>
          <p v-if="folderValidationMessage" class="field-message" :class="isFolderValid ? 'is-valid' : 'is-invalid'">
            {{ folderValidationMessage }}
          </p>
        </div>

        <HelperConnectionFields
          v-if="source === 'helper'"
          :helper-url="helperUrl"
          :helper-token="helperToken"
          :token-placeholder="tokenPlaceholder"
          :token-status-message="tokenStatusMessage"
          :is-token-saved="isTokenSaved"
          :status-tone="helperStatusTone"
          :is-checking="isCheckingHelper"
          :test-message="helperTestMessage"
          @update:helper-url="helperUrl = $event"
          @update:helper-token="helperToken = $event"
          @test-connection="handleTestConnection"
          @connect-helper="handleConnectHelper"
        />

        <HelperSetupGuide
          v-if="source === 'helper' && !isHelperOnline"
          :is-checking="isCheckingHelper"
          @check-again="handleCheckHelper"
        />


        <div class="save-row">
          <span class="save-note">
            The helper token is stored encrypted and is never returned by the API. Repository selections stay per user.
          </span>
          <button class="save-button" type="button" :disabled="isSaving" @click="handleSave">
            {{ isSaving ? "Saving…" : "Save workspace" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.workspace-settings { display: flex; flex-direction: column; gap: 16px; }

.toast-banner { padding: 12px 16px; border-radius: 9px; font-size: 13px; font-weight: 500; }
.toast-banner.success {
  border: 1px solid color-mix(in srgb, var(--positive) 30%, transparent); color: var(--positive);
  background: color-mix(in srgb, var(--positive) 10%, transparent);
}
.toast-banner.error {
  border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent); color: var(--danger);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
}

.settings-card { border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); overflow: hidden; }
.settings-card-header { padding: 20px 24px; border-bottom: 1px solid var(--border); }
.settings-card-title { color: var(--text); font-size: 15px; font-weight: 600; letter-spacing: -0.01em; }
.settings-card-description { margin-top: 4px; color: var(--muted); font-size: 13px; line-height: 1.55; }
.settings-card-body { display: flex; flex-direction: column; gap: 22px; padding: 24px; }

.field-wrap { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field-label { color: var(--muted); font-size: 12px; font-weight: 500; }
.field-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.field-input {
  width: 100%; min-width: 0; height: 40px; padding: 0 12px; border: 1px solid var(--border); border-radius: 8px;
  outline: none; background: var(--surface-raised); color: var(--text); font-size: 13px;
}
.field-input:focus { border-color: var(--border-strong); }
.field-input::placeholder { color: var(--subtle); }
.field-row .field-input { flex: 1; }
.field-message { margin: 0; font-size: 12px; line-height: 1.5; }
.field-message.is-valid { color: var(--positive); }
.field-message.is-invalid { color: var(--danger); }
.field-btn {
  display: inline-flex; align-items: center; justify-content: center; min-height: 40px; padding: 0 14px;
  border: 1px solid var(--border); border-radius: 8px; background: var(--surface-raised); color: var(--text);
  font-size: 12px; font-weight: 500; white-space: nowrap; cursor: pointer;
  transition: border-color 160ms ease, background-color 160ms ease, opacity 160ms ease;
}
.field-btn:hover:not(:disabled) { border-color: var(--border-strong); background: var(--surface-hover); }
.field-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.source-choice { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.source-option {
  display: flex; flex-direction: column; gap: 4px; padding: 12px 14px; border: 1px solid var(--border);
  border-radius: 8px; background: var(--surface-raised); text-align: left; cursor: pointer;
  transition: border-color 160ms ease, background-color 160ms ease;
}
.source-option:hover, .source-option.is-active { border-color: var(--border-strong); background: var(--surface-hover); }
.source-name { color: var(--text); font-size: 13px; font-weight: 600; }
.source-note { color: var(--subtle); font-size: 11.5px; line-height: 1.45; }

.source-warning {
  padding: 11px 13px; border: 1px solid color-mix(in srgb, var(--warning) 30%, transparent); border-radius: 8px;
  background: color-mix(in srgb, var(--warning) 10%, transparent); color: var(--warning); font-size: 12px; line-height: 1.55;
}

.save-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding-top: 4px; flex-wrap: wrap; }
.save-note { max-width: 460px; color: var(--subtle); font-size: 12px; line-height: 1.5; }
.save-button {
  display: inline-flex; align-items: center; justify-content: center; min-height: 38px; padding: 0 18px;
  border: 1px solid transparent; border-radius: 8px; background: var(--accent); color: var(--accent-contrast);
  font-size: 13px; font-weight: 600; cursor: pointer; transition: opacity 180ms ease;
}
.save-button:hover:not(:disabled) { opacity: 0.9; }
.save-button:disabled { opacity: 0.5; cursor: not-allowed; }

@media (max-width: 768px) {
  .source-choice { grid-template-columns: 1fr; }
  .settings-card-body { padding: 18px; }
}
</style>

<script setup lang="ts">
import { ref, watch } from "vue"
import { useSettings } from "../composables/useSettings"

const { secrets, config, isSaving, saveSuccess, errorMessage, save } = useSettings()

const jiraUrl = ref("https://jira.bri.co.id")
const jiraPat = ref("")
const jiraUsername = ref("")

const bitbucketUrl = ref("https://bitbucket.bri.co.id")
const bitbucketPat = ref("")
const bitbucketUsername = ref("")

const confluenceUrl = ref("https://confluence.bri.co.id")
const confluencePat = ref("")

watch(
  [secrets, config],
  () => {
    if (config.value?.jira_base_url) jiraUrl.value = config.value.jira_base_url
    if (config.value?.bitbucket_base_url) bitbucketUrl.value = config.value.bitbucket_base_url
    if (config.value?.confluence_base_url) confluenceUrl.value = config.value.confluence_base_url

    if (secrets.value?.jira_username) jiraUsername.value = secrets.value.jira_username
    if (secrets.value?.bitbucket_username) bitbucketUsername.value = secrets.value.bitbucket_username
  },
  { immediate: true }
)

const handleSave = async () => {
  await save({
    jira_pat: jiraPat.value.trim() || undefined,
    jira_username: jiraUsername.value.trim() || undefined,
    bitbucket_pat: bitbucketPat.value.trim() || undefined,
    bitbucket_username: bitbucketUsername.value.trim() || undefined,
    confluence_pat: confluencePat.value.trim() || undefined
  })
  jiraPat.value = ""
  bitbucketPat.value = ""
  confluencePat.value = ""
}
</script>

<template>
  <div class="space-y-6">
    <div v-if="saveSuccess" class="toast-banner success">
      Settings saved successfully. Credentials encrypted with AES-256-GCM.
    </div>

    <div v-if="errorMessage" class="toast-banner error">
      {{ errorMessage }}
    </div>

    <div class="settings-card">
      <div class="settings-card-header">
        <div class="settings-card-title">Developer integrations</div>
        <div class="settings-card-description">
          Credentials are stored securely in your private vault and used only for your connected BRI workspace actions.
        </div>
      </div>

      <div class="settings-card-body">
        <div class="credential-row">
          <div class="credential-meta">
            <div class="credential-name">Jira BRI</div>
            <div class="credential-subtitle">Tickets, sprint status, and issue context</div>
            <span class="status-tag" :class="secrets?.has_jira_pat ? 'connected' : 'unconfigured'">
              {{ secrets?.has_jira_pat ? "Configured" : "Not configured" }}
            </span>
          </div>

          <div class="credential-fields">
            <div class="field-wrap">
              <label class="field-label" for="jira-url">Base URL</label>
              <input class="field-input" id="jira-url" v-model="jiraUrl" readonly />
            </div>

            <div class="field-wrap">
              <label class="field-label" for="jira-pat">Personal Access Token (PAT)</label>
              <input class="field-input" id="jira-pat" type="password" v-model="jiraPat"
                :placeholder="secrets?.has_jira_pat ? '•••••••••••••••• (Leave blank to keep existing)' : 'Enter Jira PAT'" />
            </div>
          </div>
        </div>

        <div class="credential-row">
          <div class="credential-meta">
            <div class="credential-name">Bitbucket BRI</div>
            <div class="credential-subtitle">Pull requests, git diffs, and AI code review</div>
            <span class="status-tag" :class="secrets?.has_bitbucket_pat ? 'connected' : 'unconfigured'">
              {{ secrets?.has_bitbucket_pat ? "Configured" : "Not configured" }}
            </span>
          </div>

          <div class="credential-fields">
            <div class="field-wrap">
              <label class="field-label" for="bb-user">Username</label>
              <input class="field-input" id="bb-user" v-model="bitbucketUsername" placeholder="e.g. 0099999" />
            </div>

            <div class="field-wrap">
              <label class="field-label" for="bb-pat">Personal Access Token (PAT)</label>
              <input class="field-input" id="bb-pat" type="password" v-model="bitbucketPat"
                :placeholder="secrets?.has_bitbucket_pat ? '•••••••••••••••• (Leave blank to keep existing)' : 'Enter Bitbucket PAT'" />
            </div>
          </div>
        </div>

        <div class="credential-row">
          <div class="credential-meta">
            <div class="credential-name">Confluence BRI</div>
            <div class="credential-subtitle">Documentation, specs, and SQL query review</div>
            <span class="status-tag" :class="secrets?.has_confluence_pat ? 'connected' : 'unconfigured'">
              {{ secrets?.has_confluence_pat ? "Configured" : "Not configured" }}
            </span>
          </div>

          <div class="credential-fields">
            <div class="field-wrap">
              <label class="field-label" for="conf-url">Base URL</label>
              <input class="field-input" id="conf-url" v-model="confluenceUrl" readonly />
            </div>

            <div class="field-wrap">
              <label class="field-label" for="conf-pat">Personal Access Token (PAT)</label>
              <input class="field-input" id="conf-pat" type="password" v-model="confluencePat"
                :placeholder="secrets?.has_confluence_pat ? '•••••••••••••••• (Leave blank to keep existing)' : 'Enter Confluence PAT'" />
            </div>
          </div>
        </div>

        <div class="save-row">
          <span class="save-note">Credentials are encrypted at rest and never rendered in plain text.</span>
          <button class="save-button" type="button" :disabled="isSaving" @click="handleSave">
            {{ isSaving ? "Saving..." : "Save changes" }}
          </button>
        </div>
      </div>
    </div>

    <div class="settings-card">
      <div class="settings-card-header">
        <div class="settings-card-title">Connection state</div>
        <div class="settings-card-description">
          Current integration readiness across your individual developer tools.
        </div>
      </div>

      <div class="settings-card-body">
        <div class="context-list">
          <div class="context-item">
            <span class="context-name">Jira BRI</span>
            <span class="context-value" :class="secrets?.has_jira_pat ? 'connected' : 'warning'">
              {{ secrets?.has_jira_pat ? "Connected" : "Not configured" }}
            </span>
          </div>
          <div class="context-item">
            <span class="context-name">Bitbucket BRI</span>
            <span class="context-value" :class="secrets?.has_bitbucket_pat ? 'connected' : 'warning'">
              {{ secrets?.has_bitbucket_pat ? "Connected" : "Not configured" }}
            </span>
          </div>
          <div class="context-item">
            <span class="context-name">Confluence BRI</span>
            <span class="context-value" :class="secrets?.has_confluence_pat ? 'connected' : 'warning'">
              {{ secrets?.has_confluence_pat ? "Connected" : "Not configured" }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toast-banner {
  padding: 12px 16px;
  border-radius: 9px;
  font-size: 13px;
  font-weight: 500;
}

.toast-banner.success {
  background: rgba(159, 182, 166, 0.12);
  border: 1px solid rgba(159, 182, 166, 0.3);
  color: var(--positive);
}

.toast-banner.error {
  background: rgba(198, 144, 144, 0.12);
  border: 1px solid rgba(198, 144, 144, 0.3);
  color: var(--danger);
}

.settings-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  overflow: hidden;
}

.settings-card-header {
  padding: 20px 24px;
  border-bottom: 1px solid var(--border);
}

.settings-card-title {
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.settings-card-description {
  margin-top: 4px;
  color: var(--muted);
  font-size: 13px;
}

.settings-card-body {
  padding: 24px;
}

.credential-row {
  display: grid;
  grid-template-columns: 240px 1fr;
  gap: 24px;
  padding: 22px 0;
  border-bottom: 1px solid var(--border);
}

.credential-row:first-child {
  padding-top: 0;
}

.credential-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.credential-name {
  color: var(--text);
  font-size: 14px;
  font-weight: 600;
}

.credential-subtitle {
  color: var(--subtle);
  font-size: 12px;
  line-height: 1.4;
}

.status-tag {
  display: inline-block;
  align-self: flex-start;
  margin-top: 8px;
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
}

.status-tag.connected {
  background: rgba(159, 182, 166, 0.12);
  color: var(--positive);
  border: 1px solid rgba(159, 182, 166, 0.25);
}

.status-tag.unconfigured {
  background: rgba(255, 255, 255, 0.04);
  color: var(--subtle);
  border: 1px solid var(--border);
}

.credential-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.field-wrap {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-label {
  color: var(--muted);
  font-size: 12px;
  font-weight: 500;
}

.field-input {
  width: 100%;
  height: 40px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
  color: var(--text);
  font-size: 13px;
  outline: none;
  transition: border-color 180ms ease;
}

.field-input:focus {
  border-color: var(--border-strong);
}

.field-input[readonly] {
  opacity: 0.65;
  cursor: not-allowed;
}

.save-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 24px;
}

.save-note {
  color: var(--subtle);
  font-size: 12px;
}

.save-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 38px;
  padding: 0 18px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--accent);
  color: var(--accent-contrast);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 180ms ease, transform 180ms ease;
}

.save-button:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.save-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.context-list {
  display: grid;
  gap: 12px;
}

.context-item {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-raised);
}

.context-name {
  color: var(--text);
  font-size: 13px;
  font-weight: 500;
}

.context-value {
  font-size: 12px;
  font-weight: 500;
}

.context-value.connected {
  color: var(--positive);
}

.context-value.warning {
  color: var(--warning);
}

@media (max-width: 768px) {
  .credential-row {
    grid-template-columns: 1fr;
  }

  .credential-fields {
    grid-template-columns: 1fr;
  }
}
</style>

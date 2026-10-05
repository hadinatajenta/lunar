<script setup lang="ts">
import { reactive } from "vue"
import AiProviderRow from "./AiProviderRow.vue"
import { useSettings } from "../composables/useSettings"
import { useTheme } from "../../../composables/useTheme"

interface ProviderSetting {
  id: string
  name: string
  subtitle: string
  placeholder: string
}

const providerSettings: ProviderSetting[] = [
  {
    id: "openai",
    name: "OpenAI",
    subtitle: "GPT-6 Astra, GPT-6 Sol, GPT-6 Luna (Reasoning Effort: none to max)",
    placeholder: "sk-proj-... (OpenAI API key)"
  },
  {
    id: "claude",
    name: "Anthropic Claude",
    subtitle: "Claude Opus 5.5, Claude Sonnet 5.5, Claude Fable 5.1, Claude Haiku 4.5",
    placeholder: "sk-ant-... (Anthropic API key)"
  },
  {
    id: "gemini",
    name: "Google Gemini",
    subtitle: "Gemini 3.8 Flash, Gemini 3.1 Pro, Gemini 3.5 Flash-Lite (Extended Thinking)",
    placeholder: "AIzaSy... (Google AI Studio key)"
  },
  {
    id: "deepseek",
    name: "DeepSeek",
    subtitle: "deepseek-v4-pro, deepseek-flash (Thinking Mode: enabled, reasoning_content)",
    placeholder: "sk-... (DeepSeek API key)"
  },
  {
    id: "mimo",
    name: "Xiaomi MiMo",
    subtitle: "xiaomi/mimo-v2.5-pro (MiMo Code Engine, reasoningEffort: low/medium/high)",
    placeholder: "Enter Xiaomi MiMo API key"
  },
  {
    id: "opencode-go",
    name: "OpenCode Go",
    subtitle: "deepseek-v4.1-flash, mimo-v2.6-flash, space-bunny-free (OpenCode Go subscription)",
    placeholder: "Enter OpenCode Go API key"
  }
]

const { secrets, config, isSaving, saveSuccess, errorMessage, save } = useSettings()
const { isDark } = useTheme()

const keyValues = reactive<Record<string, string>>({})

for (const providerSetting of providerSettings) {
  keyValues[providerSetting.id] = ""
}

const isConfigured = (provider: string): boolean => {
  return secrets.value?.configured_ai_providers?.includes(provider) ?? false
}

const isSystemProvided = (provider: string): boolean => {
  return config.value?.system_ai_providers?.includes(provider) ?? false
}

const handleSave = async (provider: string): Promise<void> => {
  const keyValue = keyValues[provider].trim()
  if (keyValue.length === 0) return
  await save({
    ai_keys: {
      [provider]: keyValue
    }
  })
  keyValues[provider] = ""
}
</script>

<template>
  <div class="space-y-6" :class="{ 'is-light': !isDark }">
    <div v-if="saveSuccess" class="toast-banner success">
      AI provider key saved and encrypted successfully.
    </div>

    <div v-if="errorMessage" class="toast-banner error">
      {{ errorMessage }}
    </div>

    <div class="settings-card">
      <div class="settings-card-header">
        <div class="settings-card-title">AI model providers</div>
        <div class="settings-card-description">
          Configure API keys for AI models used in Copilot and AI Code Review. System defaults are used automatically if available.
        </div>
      </div>

      <div class="settings-card-body">
        <AiProviderRow
          v-for="providerSetting in providerSettings"
          :key="providerSetting.id"
          v-model="keyValues[providerSetting.id]"
          :provider-id="providerSetting.id"
          :name="providerSetting.name"
          :subtitle="providerSetting.subtitle"
          :placeholder="providerSetting.placeholder"
          :configured="isConfigured(providerSetting.id)"
          :system-provided="isSystemProvided(providerSetting.id)"
          :is-saving="isSaving"
          @save="handleSave"
        />

        <div class="encryption-note">
          API keys are masked in the interface and encrypted at rest by the backend using AES-256-GCM.
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
.encryption-note {
  margin-top: 20px;
  color: var(--subtle);
  font-size: 12px;
}
</style>

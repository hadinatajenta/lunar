<script setup lang="ts">
import { computed } from "vue"
import { DEFAULT_HELPER_URL } from "../api/helper-api"
import type { HelperConnectionStatus } from "../types"

const props = defineProps<{
  helperUrl: string
  helperToken: string
  tokenPlaceholder: string
  tokenStatusMessage: string
  isTokenSaved: boolean
  statusTone: HelperConnectionStatus
  isChecking: boolean
  testMessage: string
}>()

const emit = defineEmits<{
  (event: "update:helperUrl", value: string): void
  (event: "update:helperToken", value: string): void
  (event: "test-connection"): void
  (event: "connect-helper"): void
}>()

const HELPER_STATUS_LABELS: Record<HelperConnectionStatus, string> = {
  unknown: "Not checked",
  checking: "Checking",
  online: "Online",
  offline: "Offline"
}

const statusLabel = computed(() => HELPER_STATUS_LABELS[props.statusTone])
const isOnline = computed(() => props.statusTone === "online")

const handleUrlInput = (event: Event) => {
  emit("update:helperUrl", (event.target as HTMLInputElement).value)
}

const handleTokenInput = (event: Event) => {
  emit("update:helperToken", (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="helper-block">
    <div class="helper-head">
      <span class="helper-title">Local helper</span>
      <span class="status-tag" :class="`is-${props.statusTone}`">{{ statusLabel }}</span>
    </div>

    <div class="field-grid">
      <div class="field-wrap">
        <label class="field-label" for="workspace-helper-url">Helper URL</label>
        <input id="workspace-helper-url" class="field-input" type="text" autocomplete="off"
          :value="props.helperUrl" :placeholder="DEFAULT_HELPER_URL" @input="handleUrlInput" />
      </div>

      <div class="field-wrap">
        <label class="field-label" for="workspace-helper-token">Helper token</label>
        <input id="workspace-helper-token" class="field-input" type="password" autocomplete="off"
          :value="props.helperToken" :placeholder="props.tokenPlaceholder" @input="handleTokenInput" />
        <p class="field-hint">
          <span v-if="props.isTokenSaved" class="token-indicator">Token saved</span>
          {{ props.tokenStatusMessage }}
        </p>
      </div>
    </div>

    <div class="helper-actions">
      <button class="field-btn" type="button" :disabled="props.isChecking" @click="emit('test-connection')">
        {{ props.isChecking ? "Testing…" : "Test connection" }}
      </button>
      <button class="field-btn is-primary" type="button" :disabled="props.isChecking" @click="emit('connect-helper')">
        Connect helper
      </button>
      <p v-if="props.testMessage" class="field-message" :class="isOnline ? 'is-valid' : 'is-invalid'">
        {{ props.testMessage }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.helper-block {
  display: flex; flex-direction: column; gap: 16px; padding: 16px; border: 1px solid var(--border);
  border-radius: 10px; background: var(--surface-raised);
}
.helper-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.helper-title { color: var(--text); font-size: 13px; font-weight: 600; }
.helper-actions { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }

.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.field-wrap { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.field-label { color: var(--muted); font-size: 12px; font-weight: 500; }
.field-input {
  width: 100%; min-width: 0; height: 40px; padding: 0 12px; border: 1px solid var(--border); border-radius: 8px;
  outline: none; background: var(--surface); color: var(--text); font-size: 13px;
}
.field-input:focus { border-color: var(--border-strong); }
.field-input::placeholder { color: var(--subtle); }
.field-hint { margin: 0; color: var(--subtle); font-size: 11.5px; line-height: 1.5; }
.field-message { margin: 0; font-size: 12px; line-height: 1.5; }
.field-message.is-valid { color: var(--positive); }
.field-message.is-invalid { color: var(--danger); }

.token-indicator {
  display: inline-block; margin-right: 6px; padding: 1px 7px; border-radius: 999px; font-size: 10.5px; font-weight: 600;
  border: 1px solid color-mix(in srgb, var(--positive) 30%, transparent); color: var(--positive);
  background: color-mix(in srgb, var(--positive) 10%, transparent);
}

.field-btn {
  display: inline-flex; align-items: center; justify-content: center; min-height: 40px; padding: 0 14px;
  border: 1px solid var(--border); border-radius: 8px; background: var(--surface); color: var(--text);
  font-size: 12px; font-weight: 500; white-space: nowrap; cursor: pointer;
  transition: border-color 160ms ease, background-color 160ms ease, opacity 160ms ease;
}
.field-btn:hover:not(:disabled) { border-color: var(--border-strong); background: var(--surface-hover); }
.field-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.field-btn.is-primary {
  border-color: transparent; background: var(--accent); color: var(--accent-contrast); font-weight: 600;
}
.field-btn.is-primary:hover:not(:disabled) { border-color: transparent; background: var(--accent); opacity: 0.9; }

.status-tag {
  padding: 2px 8px; border: 1px solid var(--border); border-radius: 6px; color: var(--subtle);
  font-size: 11px; font-weight: 500; white-space: nowrap;
}
.status-tag.is-online {
  border-color: color-mix(in srgb, var(--positive) 30%, transparent); color: var(--positive);
  background: color-mix(in srgb, var(--positive) 10%, transparent);
}
.status-tag.is-offline {
  border-color: color-mix(in srgb, var(--danger) 30%, transparent); color: var(--danger);
  background: color-mix(in srgb, var(--danger) 10%, transparent);
}
.status-tag.is-checking {
  border-color: color-mix(in srgb, var(--warning) 30%, transparent); color: var(--warning);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
}

@media (max-width: 768px) {
  .field-grid { grid-template-columns: 1fr; }
}
</style>

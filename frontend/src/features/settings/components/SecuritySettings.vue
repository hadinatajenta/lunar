<script setup lang="ts">
import { computed } from "vue"
import { useSettings } from "../composables/useSettings"

const { secrets } = useSettings()

const formattedUpdatedAt = computed(() => {
  if (!secrets.value?.updated_at) return "Not updated yet"
  const date = new Date(secrets.value.updated_at)
  return date.toLocaleString()
})
</script>

<template>
  <div class="settings-card">
    <div class="settings-card-header">
      <div class="settings-card-title">Security & Encryption</div>
      <div class="settings-card-description">
        Cryptographic protection and access control status for this workspace instance.
      </div>
    </div>

    <div class="settings-card-body">
      <div class="security-list">
        <div class="security-item">
          <div>
            <div class="security-name">Credential storage at rest</div>
            <div class="security-sub">All personal PATs and AI API keys are encrypted via AES-256-GCM with unique salts</div>
          </div>
          <span class="status-badge secure">AES-256-GCM Active</span>
        </div>

        <div class="security-item">
          <div>
            <div class="security-name">Session token authentication</div>
            <div class="security-sub">Cryptographically signed HMAC-SHA256 JWT with stateless Bearer verification</div>
          </div>
          <span class="status-badge secure">HMAC-SHA256 Signed</span>
        </div>

        <div class="security-item">
          <div>
            <div class="security-name">Multi-tenant boundary isolation</div>
            <div class="security-sub">Database queries enforce user_id scoping on every credential read and write</div>
          </div>
          <span class="status-badge secure">Enforced</span>
        </div>

        <div class="security-item">
          <div>
            <div class="security-name">Last vault update</div>
            <div class="security-sub">Timestamp of the most recent credential mutation for your account</div>
          </div>
          <span class="status-time">{{ formattedUpdatedAt }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: rgba(16, 18, 23, 0.86);
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

.security-list {
  display: grid;
  gap: 16px;
}

.security-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.015);
}

.security-name {
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
}

.security-sub {
  margin-top: 4px;
  color: var(--subtle);
  font-size: 12px;
}

.status-badge.secure {
  padding: 4px 10px;
  border-radius: 6px;
  background: rgba(159, 182, 166, 0.12);
  border: 1px solid rgba(159, 182, 166, 0.25);
  color: var(--positive);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.status-time {
  color: var(--muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
</style>

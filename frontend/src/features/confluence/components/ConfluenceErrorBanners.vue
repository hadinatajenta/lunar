<script setup lang="ts">
interface Props {
  error: string | null
  errorCode: number | null
  hasConfluencePat: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  (e: "retry"): void
}>()
</script>

<template>
  <div v-if="errorCode === 502" class="vpn-banner" data-testid="banner-confluence-vpn">
    <div class="banner-body">
      <svg viewBox="0 0 24 24" class="banner-icon" aria-hidden="true">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="8" x2="12" y2="12"></line>
        <line x1="12" y1="16" x2="12.01" y2="16"></line>
      </svg>
      <span>Unable to reach Confluence BRI API. Please ensure you are connected to the BRI VPN.</span>
    </div>
    <button
      class="btn btn-ghost banner-btn"
      type="button"
      data-testid="btn-retry"
      @click="emit('retry')"
    >
      Retry
    </button>
  </div>

  <div v-else-if="error" class="error-banner" data-testid="banner-confluence-error">
    <div class="banner-body">
      <svg viewBox="0 0 24 24" class="banner-icon" aria-hidden="true">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="15" y1="9" x2="9" y2="15"></line>
        <line x1="9" y1="9" x2="15" y2="15"></line>
      </svg>
      <div class="banner-text-group">
        <span>{{ error }}</span>
        <RouterLink v-if="errorCode === 401" to="/settings" class="banner-link">
          Update in Settings →
        </RouterLink>
      </div>
    </div>
    <button
      class="btn btn-ghost banner-btn"
      type="button"
      data-testid="btn-retry"
      @click="emit('retry')"
    >
      Retry
    </button>
  </div>

  <div v-else-if="!hasConfluencePat" class="warning-banner" data-testid="banner-no-confluence-pat">
    <div class="banner-body">
      <svg viewBox="0 0 24 24" class="banner-icon" aria-hidden="true">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
        <line x1="12" y1="9" x2="12" y2="13"></line>
        <line x1="12" y1="17" x2="12.01" y2="17"></line>
      </svg>
      <span>Confluence Personal Access Token (PAT) is not configured yet.</span>
    </div>
    <RouterLink to="/settings" class="banner-link">Configure in Settings →</RouterLink>
  </div>
</template>

<style scoped>
.warning-banner,
.error-banner,
.vpn-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 18px;
  border-radius: 9px;
  font-size: 13px;
  margin-bottom: 22px;
}

.warning-banner {
  border: 1px solid rgba(187, 169, 132, 0.3);
  background: rgba(187, 169, 132, 0.1);
  color: var(--warning);
}

.vpn-banner,
.error-banner {
  border: 1px solid rgba(198, 144, 144, 0.3);
  background: rgba(198, 144, 144, 0.1);
  color: var(--danger);
}

.banner-body {
  display: flex;
  align-items: center;
  gap: 10px;
}

.banner-icon {
  width: 16px;
  height: 16px;
  stroke: currentColor;
  stroke-width: 1.8;
  fill: none;
  flex-shrink: 0;
}

.banner-text-group {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.banner-link {
  color: var(--text);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.banner-link:hover {
  opacity: 0.9;
}

.banner-btn {
  min-height: 28px;
  padding: 0 10px;
  font-size: 11px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms ease,
    border-color 160ms ease,
    color 160ms ease,
    opacity 160ms ease;
}

.btn-ghost {
  border-color: var(--border);
  background: transparent;
  color: #c8ced5;
}

.btn-ghost:hover {
  border-color: var(--border-strong);
  background: rgba(255, 255, 255, 0.03);
}
</style>

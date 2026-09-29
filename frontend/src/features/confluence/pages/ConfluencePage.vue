<script setup lang="ts">
import PageLayout from "../../../components/layout/PageLayout.vue"
import { useSettings } from "../../settings/composables/useSettings"

const { secrets } = useSettings()
</script>

<template>
  <PageLayout>
    <div class="content">
      <div class="page-header">
        <p class="eyebrow">Atlassian Integration</p>
        <h1>Confluence BRI</h1>
        <p class="page-copy">
          Technical specifications, architecture documents, and DBA query review tables.
        </p>
      </div>

      <div v-if="!secrets?.has_confluence_pat" class="warning-banner">
        <span>Confluence Personal Access Token (PAT) is not configured yet.</span>
        <RouterLink to="/settings" class="banner-link">Configure in Settings →</RouterLink>
      </div>

      <div class="placeholder-card">
        <div class="card-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 16c3-5 5-8 12-8"></path>
            <path d="M8 18c2-4 4-6 10-6"></path>
          </svg>
        </div>
        <div class="card-title">Confluence Module Ready for Live Data</div>
        <div class="card-desc">
          Connected to https://confluence.bri.co.id. Document previews, rich table rendering, and Query Review extraction are configured.
        </div>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 34px 30px 48px;
}

.page-header {
  margin-bottom: 24px;
}

.eyebrow {
  margin: 0 0 9px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.03em;
}

.page-copy {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 13px;
}

.warning-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 18px;
  border: 1px solid rgba(187, 169, 132, 0.3);
  border-radius: 9px;
  background: rgba(187, 169, 132, 0.1);
  color: var(--warning);
  font-size: 13px;
  margin-bottom: 24px;
}

.banner-link {
  color: var(--text);
  font-weight: 600;
  text-decoration: underline;
}

.placeholder-card {
  padding: 60px 24px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: rgba(16, 18, 23, 0.86);
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.card-icon {
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
  color: var(--muted);
  margin-bottom: 16px;
}

.card-icon svg {
  width: 22px;
  height: 22px;
}

.card-title {
  color: var(--text);
  font-size: 16px;
  font-weight: 600;
}

.card-desc {
  max-width: 460px;
  margin-top: 8px;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.5;
}
</style>

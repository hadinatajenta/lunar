<script setup lang="ts">
import { onMounted } from "vue"
import { useRouter } from "vue-router"
import PageLayout from "../../../components/layout/PageLayout.vue"
import { useSettings } from "../../settings/composables/useSettings"
import { useConfluence } from "../composables/useConfluence"
import { useToast } from "../../../composables/useToast"
import ConfluenceErrorBanners from "../components/ConfluenceErrorBanners.vue"
import ConfluenceStatsGrid from "../components/ConfluenceStatsGrid.vue"
import DocumentToolbar from "../components/DocumentToolbar.vue"
import DocumentList from "../components/DocumentList.vue"
import type { ConfluenceDocument } from "../types"

const router = useRouter()
const { secrets, fetchSettings } = useSettings()
const { showToast } = useToast()
const {
  documents,
  isLoading,
  error,
  errorCode,
  currentFilter,
  searchInput,
  statsCounts,
  visibleDocuments,
  remainingCount,
  visibleSummary,
  fetchData,
  initialize,
  setCurrentFilter,
  setSearchQuery,
  showMore,
  clearError
} = useConfluence()

const handleNewPage = () => {
  showToast("New page is ready for backend wiring.")
}

const handleOpenDocument = (confluenceDocument: ConfluenceDocument) => {
  router.push(`/confluence/${encodeURIComponent(confluenceDocument.id)}`)
}

const handleRetry = () => {
  fetchData()
}

onMounted(() => {
  const settingsPromise = secrets.value ? Promise.resolve() : fetchSettings()
  initialize()
  settingsPromise.then(() => {
    if (secrets.value?.has_confluence_pat === false) {
      clearError()
    }
  })
})
</script>

<template>
  <PageLayout>
    <div class="content" data-testid="confluence-page">
      <div class="page-header">
        <div class="page-header-text">
          <p class="eyebrow">Engineering documentation</p>
          <h1>Confluence</h1>
          <p class="page-copy">
            Every UT spec, query review, and SOP in one place. Search, filter by type, and let
            Lunar Copilot help generate or extract documentation on the fly.
          </p>
        </div>

        <button class="btn btn-primary" type="button" @click="handleNewPage">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M12 5v14"></path>
            <path d="M5 12h14"></path>
          </svg>
          <span>New page</span>
        </button>
      </div>

      <ConfluenceErrorBanners
        :error="error"
        :error-code="errorCode"
        :has-confluence-pat="secrets?.has_confluence_pat === true"
        @retry="handleRetry"
      />

      <div
        v-if="isLoading && documents.length === 0"
        class="loading-placeholder"
        data-testid="confluence-loading"
      >
        <div class="loading-bar">
          <div class="loading-progress"></div>
        </div>
        <div class="skeleton-grid">
          <div v-for="skeletonIndex in 6" :key="skeletonIndex" class="skeleton-card"></div>
        </div>
      </div>

      <template v-else>
        <ConfluenceStatsGrid :counts="statsCounts" />
        <DocumentToolbar
          :search-value="searchInput"
          :current-filter="currentFilter"
          :counts="statsCounts"
          :visible-summary="visibleSummary"
          @search="setSearchQuery"
          @select-filter="setCurrentFilter"
        />
        <DocumentList
          :documents="visibleDocuments"
          :remaining-count="remainingCount"
          @open="handleOpenDocument"
          @show-more="showMore"
        />
      </template>
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 1320px;
  margin: 0 auto;
  padding: 36px 30px 52px;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.page-header-text {
  min-width: 0;
}

.eyebrow {
  margin: 0 0 10px;
  color: var(--muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: clamp(30px, 4vw, 46px);
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.05em;
}

.page-copy {
  max-width: 620px;
  margin: 11px 0 0;
  color: var(--muted);
  font-size: 13px;
  line-height: 1.6;
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
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms ease,
    opacity 160ms ease;
}

.btn svg {
  width: 14px;
  height: 14px;
}

.btn-primary {
  background: var(--accent);
  color: #0b0c0f;
}

.btn-primary:hover {
  opacity: 0.9;
}

.loading-bar {
  width: 100%;
  height: 2px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 16px;
}

.loading-progress {
  width: 40%;
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  animation: loading-slide 1.2s ease-in-out infinite;
}

@keyframes loading-slide {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(350%);
  }
}

.skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}

.skeleton-card {
  height: 168px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: rgba(16, 18, 23, 0.75);
  animation: skeleton-pulse 1.4s ease-in-out infinite;
}

@keyframes skeleton-pulse {
  0%,
  100% {
    opacity: 0.5;
  }
  50% {
    opacity: 1;
  }
}

@media (max-width: 820px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .content {
    padding: 26px 20px 42px;
  }
}
</style>

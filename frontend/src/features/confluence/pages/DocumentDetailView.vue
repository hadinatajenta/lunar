<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import PageLayout from "../../../components/layout/PageLayout.vue"
import { useSettings } from "../../settings/composables/useSettings"
import { useConfluence } from "../composables/useConfluence"
import ConfluenceErrorBanners from "../components/ConfluenceErrorBanners.vue"
import DocumentTypeBadge from "../components/DocumentTypeBadge.vue"
import DocumentStatusPill from "../components/DocumentStatusPill.vue"
import DocumentActionsRow from "../components/DocumentActionsRow.vue"
import DocumentMetaGrid from "../components/DocumentMetaGrid.vue"
import DocumentAiPanel from "../components/DocumentAiPanel.vue"
import DocumentCommentsSection from "../components/DocumentCommentsSection.vue"

const route = useRoute()
const router = useRouter()
const { secrets, fetchSettings } = useSettings()
const {
  selectedDocument,
  isDetailLoading,
  detailNotFound,
  error,
  errorCode,
  initialize,
  openDocument,
  closeDocument,
  clearError
} = useConfluence()

const documentId = computed(() => String(route.params.id))
const isPatMissing = computed(() => secrets.value?.has_confluence_pat === false)
const isLoadingDocument = computed(
  () =>
    isDetailLoading.value ||
    (!selectedDocument.value &&
      !detailNotFound.value &&
      !error.value &&
      !isPatMissing.value)
)

const handleBack = () => {
  router.push("/confluence")
}

const handleRetry = () => {
  openDocument(documentId.value)
}

onMounted(() => {
  const settingsPromise = secrets.value ? Promise.resolve() : fetchSettings()
  initialize()
  openDocument(documentId.value)
  settingsPromise.then(() => {
    if (secrets.value?.has_confluence_pat === false) {
      clearError()
    }
  })
})

watch(
  () => route.params.id,
  (nextDocumentId) => {
    if (typeof nextDocumentId === "string" && nextDocumentId.length > 0) {
      openDocument(nextDocumentId)
    }
  }
)

onBeforeUnmount(() => {
  closeDocument()
})
</script>

<template>
  <PageLayout>
    <div class="content">
      <button class="detail-back" type="button" data-testid="detail-back" @click="handleBack">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M19 12H5"></path>
          <path d="m11 18-6-6 6-6"></path>
        </svg>
        <span>Back to Confluence</span>
      </button>

      <div v-if="isLoadingDocument" class="detail-loading">
        <span class="detail-spinner" aria-hidden="true"></span>
        <span>Loading document…</span>
      </div>

      <div v-else-if="detailNotFound" class="detail-empty">
        <strong>Document not found.</strong>
        <span>This document may have been moved or deleted in Confluence.</span>
      </div>

      <div v-else-if="selectedDocument" class="detail-shell">
        <div class="detail-header">
          <div class="detail-badges">
            <DocumentTypeBadge
              :type="selectedDocument.type"
              :label="selectedDocument.type_label"
              data-testid="detail-type-badge"
            />
            <span class="detail-id">{{ selectedDocument.id }}</span>
            <DocumentStatusPill
              :status="selectedDocument.status"
              data-testid="detail-status"
            />
          </div>

          <h1 class="detail-title" data-testid="detail-title">{{ selectedDocument.title }}</h1>

          <DocumentActionsRow :document="selectedDocument" />
        </div>

        <DocumentMetaGrid :document="selectedDocument" />

        <div class="detail-section">
          <p class="detail-section-label">Description</p>
          <p class="detail-description" data-testid="detail-description">
            {{ selectedDocument.description || "No description available." }}
          </p>
        </div>

        <DocumentAiPanel />
        <DocumentCommentsSection />
      </div>

      <div v-else class="detail-error">
        <ConfluenceErrorBanners
          :error="error"
          :error-code="errorCode"
          :has-confluence-pat="secrets?.has_confluence_pat === true"
          @retry="handleRetry"
        />
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
.content {
  max-width: 900px;
  margin: 0 auto;
  padding: 36px 30px 52px;
}

.detail-back {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 22px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 12px;
  cursor: pointer;
  transition: color 150ms ease;
}

.detail-back:hover {
  color: var(--text);
}

.detail-back svg {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.detail-loading {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 48px 0;
  color: var(--muted);
  font-size: 13px;
}

.detail-spinner {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  border: 2px solid rgba(255, 255, 255, 0.15);
  border-top-color: #cfd6de;
  border-radius: 50%;
  animation: spin 700ms linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.detail-empty {
  padding: 48px 24px;
  text-align: center;
  border: 1px dashed var(--border-strong);
  border-radius: 12px;
  color: var(--subtle);
  font-size: 12px;
}

.detail-empty strong {
  display: block;
  margin-bottom: 6px;
  color: #c8ced5;
  font-size: 13px;
  font-weight: 500;
}

.detail-header {
  margin-bottom: 22px;
}

.detail-badges {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.detail-id {
  color: var(--subtle);
  font-size: 10px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.02em;
  white-space: nowrap;
}

.detail-title {
  margin: 0;
  font-size: clamp(24px, 3.2vw, 34px);
  font-weight: 600;
  line-height: 1.15;
  letter-spacing: -0.035em;
  color: #f0f3f6;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section-label {
  margin: 0 0 10px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.detail-description {
  margin: 0;
  color: #c4ccd4;
  font-size: 13px;
  line-height: 1.75;
}

@media (max-width: 820px) {
  .content {
    padding: 26px 20px 42px;
  }
}
</style>

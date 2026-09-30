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
        </div>

        <div class="detail-actions-sticky-container">
          <DocumentActionsRow :document="selectedDocument" />
        </div>

        <DocumentMetaGrid :document="selectedDocument" />

        <div
          v-if="selectedDocument.body && selectedDocument.body.trim().length > 0"
          class="detail-section"
        >
          <p class="detail-section-label">Document Body</p>
          <div
            class="detail-body"
            data-testid="detail-body"
            v-html="selectedDocument.body"
          ></div>
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
  width: 100%;
  min-width: 0;
  margin: 0 auto;
  padding: 36px 30px 52px;
  box-sizing: border-box;
}

.detail-shell {
  min-width: 0;
  max-width: 100%;
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
  margin-bottom: 16px;
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
  color: var(--text);
}

.detail-actions-sticky-container {
  position: sticky;
  top: 64px;
  z-index: 9;
  background: color-mix(in srgb, var(--bg) 95%, transparent);
  border-bottom: 1px solid var(--border);
  backdrop-filter: blur(14px);
  -webkit-backdrop-filter: blur(14px);
  padding: 10px 0;
  margin-bottom: 24px;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section-label {
  margin: 0 0 10px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.detail-body {
  color: var(--text);
  font-size: 13px;
  line-height: 1.75;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.detail-body :deep(h1),
.detail-body :deep(h2),
.detail-body :deep(h3),
.detail-body :deep(h4) {
  color: var(--text);
  font-weight: 600;
  line-height: 1.3;
  margin: 24px 0 12px;
}

.detail-body :deep(h1) {
  font-size: 20px;
  letter-spacing: -0.02em;
}

.detail-body :deep(h2) {
  font-size: 17px;
  letter-spacing: -0.015em;
}

.detail-body :deep(h3) {
  font-size: 15px;
}

.detail-body :deep(h4) {
  font-size: 13.5px;
}

.detail-body :deep(p) {
  color: var(--text);
  font-size: 13px;
  line-height: 1.7;
}

.detail-body :deep(ul),
.detail-body :deep(ol) {
  color: var(--text);
  font-size: 13px;
  line-height: 1.7;
}

.detail-body :deep(li) {
  margin-bottom: 6px;
}

.detail-body :deep(.table-wrap) {
  width: 100% !important;
  max-width: 100% !important;
  overflow-x: auto;
  margin: 18px 0;
}

.detail-body :deep(table) {
  width: 100% !important;
  max-width: 100% !important;
  table-layout: fixed;
  border-collapse: collapse;
  margin: 18px 0;
  font-size: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
}

.detail-body :deep(colgroup),
.detail-body :deep(col) {
  display: none !important;
}

.detail-body :deep(th),
.detail-body :deep(td) {
  padding: 10px 14px;
  text-align: left;
  border: 1px solid var(--border);
  word-break: break-word;
  overflow-wrap: anywhere;
  vertical-align: top;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}

.detail-body :deep(tr:nth-child(even)) {
  background: color-mix(in srgb, var(--surface-raised) 40%, transparent);
}

.detail-body :deep(span[style*="color"]) {
  color: inherit !important;
}

.detail-body :deep(pre) {
  white-space: pre-wrap !important;
  word-break: break-word !important;
  overflow-wrap: anywhere !important;
  padding: 10px 12px;
  margin: 6px 0;
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--text);
  border: 1px solid var(--border);
  line-height: 1.6;
}

.detail-body :deep(code) {
  white-space: pre-wrap !important;
  word-break: break-word !important;
  overflow-wrap: anywhere !important;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11.5px;
}

.detail-body :deep(pre code) {
  color: var(--text);
  line-height: 1.6;
}

.detail-body :deep(:not(pre) > code) {
  background: var(--surface-raised);
  color: var(--text);
  border: 1px solid var(--border);
  padding: 2px 5px;
  border-radius: 4px;
  font-size: 11px;
}

.detail-body :deep(blockquote) {
  margin: 16px 0;
  padding: 12px 18px;
  border-left: 3px solid var(--accent);
  background: var(--surface-raised);
  border-radius: 0 8px 8px 0;
  color: var(--text);
  font-size: 13px;
  line-height: 1.65;
}

.detail-body :deep(blockquote p:last-child) {
  margin-bottom: 0;
}

@media (max-width: 820px) {
  .content {
    padding: 26px 20px 42px;
  }
}
</style>

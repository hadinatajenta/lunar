<script setup lang="ts">
import type { ConfluenceDocument } from "../types"
import DocumentCard from "./DocumentCard.vue"

interface Props {
  documents: ConfluenceDocument[]
  remainingCount: number
}

defineProps<Props>()

const emit = defineEmits<{
  (e: "open", confluenceDocument: ConfluenceDocument): void
  (e: "show-more"): void
}>()
</script>

<template>
  <div v-if="documents.length > 0" class="doc-grid">
    <DocumentCard
      v-for="confluenceDocument in documents"
      :key="confluenceDocument.id"
      :document="confluenceDocument"
      @open="emit('open', $event)"
    />
  </div>

  <div v-else class="empty-state" data-testid="doc-empty">
    <strong>No documents match your filters.</strong>
    <span>Try a different keyword or switch to “All”.</span>
  </div>

  <button
    v-if="remainingCount > 0"
    class="show-more-btn"
    type="button"
    data-testid="doc-show-more"
    @click="emit('show-more')"
  >
    Show more · {{ remainingCount }} remaining
  </button>
</template>

<style scoped>
.doc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  border: 1px dashed var(--border-strong);
  border-radius: 12px;
  color: var(--subtle);
  font-size: 12px;
}

.empty-state strong {
  display: block;
  margin-bottom: 6px;
  color: #c8ced5;
  font-size: 13px;
  font-weight: 500;
}

.show-more-btn {
  display: block;
  width: 100%;
  margin-top: 16px;
  min-height: 42px;
  padding: 0 12px;
  border: 1px dashed var(--border-strong);
  border-radius: 10px;
  background: transparent;
  color: var(--muted);
  font-size: 11.5px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    border-color 150ms ease;
}

.show-more-btn:hover {
  background: rgba(255, 255, 255, 0.03);
  color: var(--text);
  border-color: rgba(255, 255, 255, 0.2);
}

@media (max-width: 1000px) {
  .doc-grid {
    grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  }
}

@media (max-width: 620px) {
  .doc-grid {
    grid-template-columns: 1fr;
  }
}
</style>

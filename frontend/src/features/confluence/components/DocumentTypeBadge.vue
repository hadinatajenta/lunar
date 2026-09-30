<script setup lang="ts">
import type { ConfluenceDocType } from "../types"

interface Props {
  type: ConfluenceDocType
  label: string
}

defineProps<Props>()
</script>

<template>
  <span class="doc-type-badge" :class="`is-${type}`">{{ label }}</span>
</template>

<style scoped>
.doc-type-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  height: 22px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 9.5px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  flex: 0 0 auto;
}

.doc-type-badge.is-ut {
  color: #b5cbbd;
  border-color: rgba(159, 182, 166, 0.3);
  background: rgba(159, 182, 166, 0.08);
}

.doc-type-badge.is-query {
  color: #cbbb97;
  border-color: rgba(187, 169, 132, 0.3);
  background: rgba(187, 169, 132, 0.08);
}

.doc-type-badge.is-sop {
  color: #a9b8c9;
  border-color: rgba(150, 170, 195, 0.3);
  background: rgba(150, 170, 195, 0.08);
}

.doc-type-badge.is-doc {
  color: #b9c0c9;
  border-color: rgba(141, 149, 159, 0.3);
  background: rgba(141, 149, 159, 0.08);
}
</style>

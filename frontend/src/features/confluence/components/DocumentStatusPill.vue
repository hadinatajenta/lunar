<script setup lang="ts">
import { computed } from "vue"
import type { ConfluenceDocStatus } from "../types"

interface Props {
  status: ConfluenceDocStatus
}

const props = defineProps<Props>()

const statusLabel = computed(() => {
  if (props.status === "progress") {
    return "In progress"
  }
  if (props.status === "done") {
    return "Done"
  }
  return "Open"
})
</script>

<template>
  <span class="status-pill" :class="`is-${status}`">{{ statusLabel }}</span>
</template>

<style scoped>
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 8px;
  border-radius: 999px;
  border: 1px solid var(--border);
  font-size: 9.5px;
  font-weight: 500;
  white-space: nowrap;
  flex: 0 0 auto;
}

.status-pill::before {
  content: "";
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
}

.status-pill.is-open {
  color: #a9b1bb;
  border-color: var(--border);
  background: rgba(255, 255, 255, 0.03);
}

.status-pill.is-progress {
  color: #cbbb97;
  border-color: rgba(187, 169, 132, 0.3);
  background: rgba(187, 169, 132, 0.08);
}

.status-pill.is-done {
  color: #b5cbbd;
  border-color: rgba(159, 182, 166, 0.3);
  background: rgba(159, 182, 166, 0.08);
}
</style>

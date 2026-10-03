<script setup lang="ts">
import type { RepositoryDetail } from "../types"
import ServiceRow from "./ServiceRow.vue"

const props = defineProps<{
  services: RepositoryDetail[]
  openServiceNames: string[]
  modelLabels: Record<string, string>
  emptyMessage: string
}>()

const emit = defineEmits<{
  (event: "toggle", serviceName: string): void
  (event: "open-model", repositoryName: string): void
}>()
</script>

<template>
  <div class="table-wrap">
    <table class="table">
      <thead>
        <tr>
          <th>Service</th>
          <th class="col-branch">Branch</th>
          <th>Dirty</th>
          <th class="col-updated">Updated</th>
          <th class="col-action"></th>
        </tr>
      </thead>
      <tbody>
        <ServiceRow
          v-for="service in props.services"
          :key="service.name"
          :service="service"
          :is-open="props.openServiceNames.includes(service.name)"
          :model-label="props.modelLabels[service.name] ?? ''"
          @toggle="emit('toggle', service.name)"
          @open-model="emit('open-model', $event)"
        />

        <tr v-if="props.services.length === 0" class="empty-row">
          <td colspan="5">{{ props.emptyMessage }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--surface);
  overflow: hidden;
}

.table {
  width: 100%;
  border-collapse: collapse;
  table-layout: auto;
}

.table thead th {
  padding: 11px 18px;
  border-bottom: 1px solid var(--border);
  background: var(--surface-raised);
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  text-align: left;
  white-space: nowrap;
}

.table thead th.col-action {
  width: 44px;
  text-align: right;
}

.empty-row td {
  padding: 34px 18px;
  color: var(--subtle);
  font-size: 12px;
  text-align: center;
}

@media (max-width: 820px) {
  .table thead th.col-branch {
    display: none;
  }

  .table thead th,
  .empty-row td {
    padding-left: 12px;
    padding-right: 12px;
  }
}

@media (max-width: 620px) {
  .table thead th.col-updated {
    display: none;
  }
}
</style>

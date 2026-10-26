<script setup lang="ts">
import { computed } from "vue"
import type { UtDocsRow } from "../types"
import {
  NO_UT_SCENARIO_MESSAGE,
  UT_SCENARIO_GROUP_COLSPAN,
  UT_SCENARIO_HEADERS,
  formatUtResultLabel,
  groupUtScenarioRows,
  hasExecutedUtRows,
  resolveUtResultTone
} from "../utils/ut-test-table"

const props = defineProps<{
  rows: UtDocsRow[]
  screenshotUrls: Record<string, string>
}>()

const groups = computed(() => groupUtScenarioRows(props.rows))
const isGrouped = computed(() => groups.value.length > 1)
const screenshotEntries = computed(() => Object.entries(props.screenshotUrls))
const hasExecutedRows = computed(() => hasExecutedUtRows(props.rows))

const screenshotFor = (repository: string): string => {
  const direct = props.screenshotUrls[repository]
  if (direct !== undefined) {
    return direct
  }
  if (screenshotEntries.value.length === 1) {
    return screenshotEntries.value[0][1]
  }
  return ""
}

const screenshotGroups = computed(() =>
  groups.value
    .map((group) => ({ repository: group.repository, url: screenshotFor(group.repository) }))
    .filter((entry) => entry.url.length > 0)
)

const screenshotCaption = (repository: string): string =>
  repository.length > 0 ? `Run screenshot for ${repository}` : "Run screenshot"
</script>

<template>
  <div class="ut-scenario">
    <div
      v-if="hasExecutedRows && screenshotGroups.length > 0"
      class="ut-screenshot-strip"
      data-testid="ai-ut-screenshot-strip"
    >
      <figure
        v-for="entry in screenshotGroups"
        :key="entry.repository || 'ungrouped'"
        class="ut-screenshot"
        data-testid="ai-ut-screenshot"
      >
        <img :src="entry.url" :alt="screenshotCaption(entry.repository)" />
        <figcaption>{{ screenshotCaption(entry.repository) }}</figcaption>
      </figure>
    </div>

    <div class="ai-table-wrap">
      <table class="ai-table" data-testid="ai-ut-table">
        <thead>
          <tr>
            <th v-for="header in UT_SCENARIO_HEADERS" :key="header">{{ header }}</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="group in groups" :key="group.repository || 'ungrouped'">
            <tr v-if="isGrouped" class="ai-group-row" data-testid="ai-ut-group">
              <td :colspan="UT_SCENARIO_GROUP_COLSPAN">
                Layanan / Microservice: <code>{{ group.repository }}</code>
              </td>
            </tr>
            <tr
              v-for="row in group.rows"
              :key="`${group.repository}-${row.no}-${row.test_function_name}`"
              data-testid="ai-ut-row"
            >
              <td class="tabular">{{ row.no }}</td>
              <td data-testid="ai-ut-row-function">{{ row.function }}</td>
              <td>{{ row.scenario }}</td>
              <td>{{ row.expected_result }}</td>
              <td>{{ row.input_data }}</td>
              <td>
                <span
                  class="ai-status-pill"
                  :class="`is-${resolveUtResultTone(row.result)}`"
                  data-testid="ai-ut-row-result"
                >
                  {{ formatUtResultLabel(row.result) }}
                </span>
              </td>
              <td>{{ row.screen_capture }}</td>
            </tr>
          </template>
          <tr v-if="rows.length === 0" data-testid="ai-ut-table-empty">
            <td :colspan="UT_SCENARIO_GROUP_COLSPAN">{{ NO_UT_SCENARIO_MESSAGE }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.ut-scenario {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ut-screenshot-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.ut-screenshot {
  flex: 0 0 auto;
  max-width: 100%;
  margin: 0;
}

.ut-screenshot img {
  display: block;
  width: 400px;
  max-width: 100%;
  height: auto;
  border: 1px solid var(--border);
  border-radius: 10px;
}

.ut-screenshot figcaption {
  margin-top: 6px;
  color: var(--muted);
  font-size: 11px;
}

.ai-table-wrap {
  max-width: 100%;
  overflow-x: auto;
}

.ai-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11.5px;
}

.ai-table th,
.ai-table td {
  padding: 8px 10px;
  border: 1px solid var(--border);
  text-align: left;
  vertical-align: top;
  color: var(--text);
  overflow-wrap: anywhere;
}

.ai-table th {
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  background: var(--surface-raised);
}

.ai-group-row td {
  background: var(--surface-raised);
  font-weight: 600;
}

.ai-group-row code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}

.ai-status-pill {
  display: inline-block;
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  white-space: nowrap;
}

.ai-status-pill.is-passed {
  border-color: color-mix(in srgb, var(--positive) 45%, transparent);
  color: var(--positive);
}

.ai-status-pill.is-failed {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  color: var(--danger);
}

.ai-status-pill.is-skipped {
  border-color: color-mix(in srgb, var(--warning) 45%, transparent);
  color: var(--warning);
}

.ai-status-pill.is-not_run {
  border-color: var(--border-strong);
  border-style: dashed;
  color: var(--muted);
}

.ai-status-pill.is-unknown {
  color: var(--muted);
}

.tabular {
  font-variant-numeric: tabular-nums;
}
</style>

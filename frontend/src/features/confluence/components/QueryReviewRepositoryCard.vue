<script setup lang="ts">
import type { QueryReviewRepository } from "../types"
import { NO_QUERY_CHANGES_MESSAGE } from "../utils/capture-table"

const noQueryChangesMessage = NO_QUERY_CHANGES_MESSAGE

defineProps<{
  repository: QueryReviewRepository
}>()

const emit = defineEmits<{
  (event: "copy-table", repository: QueryReviewRepository): void
}>()

const STATUS_LABELS: Record<QueryReviewRepository["status"], string> = {
  success: "Success",
  no_queries: "No queries",
  error: "Error"
}
</script>

<template>
  <article class="ai-repository" data-testid="ai-query-repository">
    <header class="ai-repository-head">
      <h4 class="ai-repository-name" data-testid="ai-query-repository-name">
        [{{ repository.repository }}]
      </h4>
      <span
        class="ai-status-pill"
        :class="`is-${repository.status}`"
        data-testid="ai-query-repository-status"
      >
        {{ STATUS_LABELS[repository.status] }}
      </span>
      <span class="ai-repository-branches tabular" data-testid="ai-query-repository-branches">
        {{ repository.source_branch }} → {{ repository.target_branch }}
      </span>
      <button
        v-if="repository.queries.length > 0"
        class="ai-button is-small ai-repository-copy"
        type="button"
        data-testid="ai-query-copy-repository"
        @click="emit('copy-table', repository)"
      >
        Copy table
      </button>
      <a
        v-if="repository.compare_url"
        class="ai-source-link"
        :href="repository.compare_url"
        target="_blank"
        rel="noopener noreferrer"
        data-testid="ai-query-repository-compare"
      >
        Compare diff
      </a>
    </header>

    <p
      v-if="repository.status === 'error'"
      class="ai-notice is-danger"
      data-testid="ai-query-repository-error"
    >
      {{ repository.error || "This repository could not be analyzed." }}
    </p>

    <template v-else>
      <p v-if="repository.summary" class="ai-summary" data-testid="ai-query-summary">
        {{ repository.summary }}
      </p>

      <p v-if="repository.truncated" class="ai-notice is-warning" data-testid="ai-query-truncated">
        The compare diff exceeded the per-file budget, so some files were not analyzed.
      </p>

      <div
        v-if="repository.analyzed_files.length > 0"
        class="ai-chips"
        data-testid="ai-query-analyzed"
      >
        <span v-for="file in repository.analyzed_files" :key="file.path" class="ai-chip tabular">
          {{ file.path }} +{{ file.added_lines }}
        </span>
      </div>

      <details
        v-if="repository.skipped_files.length > 0"
        class="ai-skipped"
        data-testid="ai-query-skipped"
      >
        <summary>{{ repository.skipped_files.length }} skipped files</summary>
        <ul>
          <li v-for="file in repository.skipped_files" :key="file.path">
            <code>{{ file.path }}</code>
            <span>{{ file.reason }}</span>
          </li>
        </ul>
      </details>

      <div class="ai-table-wrap">
        <table class="ai-table" data-testid="ai-query-table">
          <thead>
            <tr>
              <th>No</th>
              <th>Fitur</th>
              <th>Query Statement</th>
              <th>Index</th>
              <th>Capture Explain / Execution Plan</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-if="repository.queries.length === 0"
              data-testid="ai-query-repository-no-queries"
            >
              <td colspan="5">{{ noQueryChangesMessage }}</td>
            </tr>
            <tr
              v-for="entry in repository.queries"
              :key="entry.no"
              data-testid="ai-query-row"
            >
              <td class="tabular">{{ entry.no }}</td>
              <td>{{ entry.feature }}</td>
              <td>
                <pre class="ai-statement"><code>{{ entry.query_statement }}</code></pre>
              </td>
              <td data-testid="ai-query-index">
                <span v-if="entry.index">{{ entry.index }}</span>
                <pre
                  v-if="entry.create_index"
                  class="ai-index-statement"
                  data-testid="ai-query-create-index"
                ><code>{{ entry.create_index }}</code></pre>
              </td>
              <td></td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </article>
</template>

<style scoped>
.ai-repository {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
}

.ai-repository-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.ai-repository-name {
  margin: 0;
  color: var(--text);
  font-size: 12.5px;
  font-weight: 600;
}

.ai-status-pill {
  padding: 2px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  white-space: nowrap;
}

.ai-status-pill.is-success {
  border-color: color-mix(in srgb, var(--positive) 45%, transparent);
  color: var(--positive);
}

.ai-status-pill.is-no_queries {
  border-color: color-mix(in srgb, var(--warning) 45%, transparent);
  color: var(--warning);
}

.ai-status-pill.is-error {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  color: var(--danger);
}

.ai-repository-branches {
  color: var(--muted);
  font-size: 11px;
}

.ai-repository-copy {
  margin-left: auto;
}

.ai-source-link {
  color: var(--accent);
  text-decoration: underline;
  white-space: nowrap;
  font-size: 11.5px;
}

.ai-summary {
  margin: 0;
  color: var(--text);
  font-size: 12.5px;
  line-height: 1.65;
}

.ai-notice {
  margin: 0;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface);
  color: var(--muted);
  font-size: 12px;
  line-height: 1.6;
}

.ai-notice.is-warning {
  border-color: color-mix(in srgb, var(--warning) 45%, transparent);
  color: var(--warning);
}

.ai-notice.is-danger {
  border-color: color-mix(in srgb, var(--danger) 45%, transparent);
  color: var(--danger);
}

.ai-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.ai-chip {
  padding: 4px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 10.5px;
}

.ai-skipped {
  color: var(--muted);
  font-size: 11.5px;
}

.ai-skipped summary {
  cursor: pointer;
}

.ai-skipped ul {
  margin: 8px 0 0;
  padding-left: 18px;
}

.ai-skipped li {
  margin-bottom: 4px;
}

.ai-skipped code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}

.ai-no-queries {
  margin: 0;
  color: var(--subtle);
  font-size: 12px;
  line-height: 1.6;
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

.ai-statement {
  margin: 0 0 6px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--text);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.55;
}

.ai-statement code {
  font-family: inherit;
}

.ai-index-statement {
  margin: 6px 0 0;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface-raised);
  color: var(--text);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  line-height: 1.55;
}

.ai-index-statement code {
  font-family: inherit;
}

.ai-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 26px;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text);
  font-size: 10.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 160ms cubic-bezier(0.2, 0.7, 0.3, 1),
    border-color 160ms cubic-bezier(0.2, 0.7, 0.3, 1);
}

.ai-button:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
}

.ai-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.tabular {
  font-variant-numeric: tabular-nums;
}
</style>

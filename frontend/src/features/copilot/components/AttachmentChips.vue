<script setup lang="ts">
import { computed } from "vue"
import { formatAttachmentSize } from "../utils/attachment-rules"
import type { AttachmentDraft } from "../composables/useAttachments"

const props = defineProps<{
  drafts: AttachmentDraft[]
}>()

const emit = defineEmits<{
  (event: "remove", localId: string): void
}>()

const failedDrafts = computed(() =>
  props.drafts.filter((draft) => draft.status === "failed")
)

const failureAnnouncement = computed(() =>
  failedDrafts.value
    .map((draft) => (draft.error === null ? draft.file.name : `${draft.file.name}: ${draft.error}`))
    .join(". ")
)
</script>

<template>
  <div class="attachment-chips">
    <ul v-if="props.drafts.length > 0" class="chip-list" data-testid="attachment-chips">
      <li
        v-for="draft in props.drafts"
        :key="draft.localId"
        class="chip"
        :class="`is-${draft.status}`"
        data-testid="attachment-chip"
        :data-status="draft.status"
      >
        <span class="chip-visual">
          <img
            v-if="draft.thumbnailUrl !== null"
            class="chip-thumb"
            data-testid="attachment-thumbnail"
            :src="draft.thumbnailUrl"
            alt=""
          />
          <svg
            v-else
            class="chip-file-icon"
            data-testid="attachment-file-icon"
            aria-hidden="true"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z"></path>
            <path d="M14 3v5h5"></path>
          </svg>
        </span>

        <span class="chip-body">
          <span class="chip-name" :title="draft.file.name">{{ draft.file.name }}</span>
          <span class="chip-meta">
            <span class="chip-status-icon" aria-hidden="true">
              <svg v-if="draft.status === 'uploading'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="9"></circle>
                <path d="M12 7v5l3 2"></path>
              </svg>
              <svg v-else-if="draft.status === 'ready'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="m5 13 4 4L19 7"></path>
              </svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="9"></circle>
                <path d="M12 8v5"></path>
                <path d="M12 16h.01"></path>
              </svg>
            </span>
            <span class="chip-size">{{ formatAttachmentSize(draft.file.size) }}</span>
            <span v-if="draft.status === 'uploading'" class="chip-state">Uploading</span>
            <span v-else-if="draft.status === 'failed'" class="chip-state">{{ draft.error }}</span>
          </span>
        </span>

        <button
          class="chip-remove"
          type="button"
          data-testid="attachment-remove"
          :aria-label="`Remove ${draft.file.name}`"
          @click="emit('remove', draft.localId)"
        >
          <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 6 6 18"></path>
            <path d="m6 6 12 12"></path>
          </svg>
        </button>
      </li>
    </ul>

    <p
      class="sr-only"
      role="status"
      aria-live="polite"
      data-testid="attachment-failure-announcement"
    >
      {{ failureAnnouncement }}
    </p>
  </div>
</template>

<style scoped>
.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0 0 8px;
  padding: 0;
  list-style: none;
}

.chip {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  padding: 5px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-raised);
  color: var(--muted);
  font-size: 11px;
  line-height: 1.4;
}

.chip-visual {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  color: var(--muted);
  overflow: hidden;
}

.chip-thumb {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.chip-file-icon {
  width: 15px;
  height: 15px;
}

.chip-body {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
  max-width: 320px;
}

.chip-name {
  overflow: hidden;
  color: var(--text);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chip-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.chip-status-icon {
  display: inline-flex;
  flex-shrink: 0;
  color: var(--muted);
}

.chip-status-icon svg {
  width: 12px;
  height: 12px;
}

.chip-size {
  color: var(--muted);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.chip-state {
  color: var(--muted);
}

.chip.is-ready .chip-status-icon {
  color: var(--positive);
}

.chip.is-failed {
  border-color: color-mix(in srgb, var(--danger) 40%, transparent);
  background: color-mix(in srgb, var(--danger) 8%, transparent);
}

.chip.is-failed .chip-status-icon,
.chip.is-failed .chip-state {
  color: var(--danger);
}

.chip-remove {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 180ms ease, background-color 180ms ease;
}

.chip-remove:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.chip-remove:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.chip-remove svg {
  width: 12px;
  height: 12px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  border: 0;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

@media (prefers-reduced-motion: reduce) {
  .chip-remove {
    transition: none;
  }
}
</style>

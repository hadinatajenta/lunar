<script setup lang="ts">
import { onMounted, ref } from "vue"
import MicroservicesModal from "./MicroservicesModal.vue"

const props = defineProps<{
  serviceName: string
  fileName: string
  requestCount: number
  folderCount: number
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "download"): void
}>()

const downloadButton = ref<HTMLButtonElement | null>(null)

onMounted(() => {
  downloadButton.value?.focus()
})
</script>

<template>
  <MicroservicesModal
    data-testid="collection-download-overlay"
    :eyebrow="props.serviceName"
    title="Your Postman collection is ready"
    title-id="collection-download-title"
    @close="emit('close')"
  >
    <div class="download-stats">
      <div class="download-stat">
        <span class="download-stat-label">Requests</span>
        <span class="download-stat-value">{{ props.requestCount }}</span>
      </div>
      <div class="download-stat">
        <span class="download-stat-label">Folders</span>
        <span class="download-stat-value">{{ props.folderCount }}</span>
      </div>
    </div>

    <p class="download-file">{{ props.fileName }}</p>
    <p class="download-note">The file downloads straight to your browser. Nothing was saved on the server.</p>

    <template #footer>
      <span class="footer-note">Check the request count before you download.</span>
      <span class="spacer"></span>
      <button class="footer-btn is-secondary" type="button" @click="emit('close')">Close</button>
      <button ref="downloadButton" class="footer-btn is-ai" type="button" @click="emit('download')">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
          <path d="M7 10l5 5 5-5"></path>
          <path d="M12 15V3"></path>
        </svg>
        Download JSON
      </button>
    </template>
  </MicroservicesModal>
</template>

<style scoped>
.download-stats { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
.download-stat {
  display: flex; flex-direction: column; gap: 4px; padding: 11px 13px; border: 1px solid var(--border);
  border-radius: 10px; background: var(--surface-raised);
}
.download-stat-label { color: var(--subtle); font-size: 10px; font-weight: 600; letter-spacing: 0.09em; text-transform: uppercase; }
.download-stat-value { color: var(--text); font-size: 19px; font-weight: 600; font-variant-numeric: tabular-nums; }
.download-file {
  margin: 14px 0 0; padding: 10px 12px; border: 1px dashed var(--border-strong); border-radius: 8px;
  color: var(--text); font-size: 11.5px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; overflow-wrap: anywhere;
}
.download-note { margin: 10px 0 0; color: var(--subtle); font-size: 10.5px; line-height: 1.55; }

.footer-note { color: var(--subtle); font-size: 10.5px; }
.spacer { flex: 1; }
.footer-btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 34px; padding: 0 14px;
  border: 1px solid transparent; border-radius: 8px; font-size: 11.5px; font-weight: 600; white-space: nowrap;
  cursor: pointer; transition: opacity 150ms ease, border-color 150ms ease, background-color 150ms ease;
}
.footer-btn.is-secondary { border-color: var(--border); background: transparent; color: var(--text); }
.footer-btn.is-secondary:hover { border-color: var(--border-strong); background: var(--surface-hover); }
.footer-btn.is-ai {
  border-color: color-mix(in srgb, var(--positive) 32%, transparent); background: color-mix(in srgb, var(--positive) 12%, transparent);
  color: var(--positive);
}
.footer-btn.is-ai:hover { background: color-mix(in srgb, var(--positive) 20%, transparent); }
.footer-btn svg { width: 13px; height: 13px; stroke: currentColor; stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; }

@media (max-width: 620px) {
  .spacer { display: none; }
  .footer-btn { width: 100%; }
  .footer-note { order: 3; }
}
</style>

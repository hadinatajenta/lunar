<script setup lang="ts">
const props = defineProps<{
  eyebrow: string
  title: string
  titleId: string
}>()

const emit = defineEmits<{
  (event: "close"): void
}>()
</script>

<template>
  <div class="modal-overlay" @click.self="emit('close')">
    <div class="modal" role="dialog" aria-modal="true" :aria-labelledby="props.titleId">
      <div class="modal-header">
        <div class="modal-header-text">
          <div v-if="props.eyebrow" class="modal-eyebrow">{{ props.eyebrow }}</div>
          <h3 :id="props.titleId" class="modal-title">{{ props.title }}</h3>
        </div>
        <button class="modal-close" type="button" aria-label="Close" @click="emit('close')">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 6l12 12"></path>
            <path d="M18 6 6 18"></path>
          </svg>
        </button>
      </div>

      <div class="modal-body">
        <slot />
      </div>

      <div v-if="$slots.alert" class="modal-alert">
        <slot name="alert" />
      </div>

      <div v-if="$slots.footer" class="modal-footer">
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed; inset: 0; z-index: 40; display: grid; place-items: center; padding: 24px;
  background: rgba(5, 6, 8, 0.72); backdrop-filter: blur(6px); -webkit-backdrop-filter: blur(6px);
}
.modal {
  display: flex; flex-direction: column; width: 100%; max-width: 520px; max-height: calc(100vh - 48px);
  border: 1px solid var(--border-strong); border-radius: 14px; background: var(--surface);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.45); overflow: hidden;
}
.modal-header {
  display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex: 0 0 auto;
  padding: 18px 20px; border-bottom: 1px solid var(--border);
}
.modal-header-text { min-width: 0; }
.modal-eyebrow {
  margin-bottom: 6px; color: var(--subtle); font-size: 10px; font-weight: 600; letter-spacing: 0.09em;
  text-transform: uppercase; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; overflow-wrap: anywhere;
}
.modal-title { margin: 0; color: var(--text); font-size: 15px; font-weight: 600; letter-spacing: -0.02em; line-height: 1.35; }
.modal-close {
  display: grid; place-items: center; width: 30px; height: 30px; flex: 0 0 auto; border: 1px solid var(--border);
  border-radius: 8px; background: transparent; color: var(--muted); cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease, border-color 150ms ease;
}
.modal-close:hover { border-color: var(--border-strong); background: var(--surface-hover); color: var(--text); }
.modal-close svg {
  width: 14px; height: 14px; stroke: currentColor; stroke-width: 1.8; fill: none;
  stroke-linecap: round; stroke-linejoin: round;
}
.modal-body { display: flex; flex-direction: column; flex: 1; padding: 18px 20px; overflow-y: auto; }
.modal-alert { flex: 0 0 auto; }
.modal-footer {
  display: flex; align-items: center; justify-content: space-between; gap: 10px; flex: 0 0 auto; flex-wrap: wrap;
  padding: 14px 20px; border-top: 1px solid var(--border); background: var(--surface-raised);
}
</style>

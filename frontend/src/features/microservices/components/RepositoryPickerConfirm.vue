<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
  repositoryNames: string[]
}>()

const emit = defineEmits<{
  (event: "keep"): void
  (event: "confirm"): void
}>()

const visibleNames = computed(() => props.repositoryNames.slice(0, 4))
const hiddenNameCount = computed(() => Math.max(props.repositoryNames.length - visibleNames.value.length, 0))
</script>

<template>
  <div class="confirm-bar" role="alert">
    <div class="confirm-text">
      <strong>{{ props.repositoryNames.length }} repositories with uncommitted changes are being removed</strong>
      <span class="confirm-list">
        {{ visibleNames.join(", ") }}
        <template v-if="hiddenNameCount > 0"> and {{ hiddenNameCount }} more</template>
      </span>
      <span class="confirm-note">This only removes them from the board. Nothing on disk is changed.</span>
    </div>

    <div class="confirm-actions">
      <button class="confirm-btn is-secondary" type="button" @click="emit('keep')">Keep them selected</button>
      <button class="confirm-btn is-danger" type="button" @click="emit('confirm')">Apply anyway</button>
    </div>
  </div>
</template>

<style scoped>
.confirm-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 13px 15px;
  border: 1px solid color-mix(in srgb, var(--warning) 32%, transparent);
  border-radius: 9px;
  background: color-mix(in srgb, var(--warning) 10%, transparent);
  flex-wrap: wrap;
}

.confirm-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.confirm-text strong {
  color: var(--warning);
  font-size: 12px;
  font-weight: 600;
}

.confirm-list,
.confirm-note {
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.confirm-list {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.confirm-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.confirm-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text);
  font-size: 11.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 150ms ease, background-color 150ms ease;
}

.confirm-btn:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
}

.confirm-btn.is-danger {
  border-color: color-mix(in srgb, var(--danger) 34%, transparent);
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
}

.confirm-btn.is-danger:hover {
  background: color-mix(in srgb, var(--danger) 20%, transparent);
}
</style>

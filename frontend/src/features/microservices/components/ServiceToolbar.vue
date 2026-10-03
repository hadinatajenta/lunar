<script setup lang="ts">
import { ref } from "vue"
import { SERVICE_FILTER_OPTIONS, type ServiceFilter } from "./service-filters"

const props = defineProps<{
  searchQuery: string
  activeFilter: ServiceFilter
  countLabel: string
  isPickerOpen: boolean
}>()

const emit = defineEmits<{
  (event: "update:searchQuery", value: string): void
  (event: "update:activeFilter", value: ServiceFilter): void
  (event: "open-picker"): void
}>()

const pickerButton = ref<HTMLButtonElement | null>(null)

const handleSearchInput = (event: Event) => {
  const inputElement = event.target as HTMLInputElement
  emit("update:searchQuery", inputElement.value)
}

const focusPickerButton = () => {
  pickerButton.value?.focus()
}

defineExpose({ focusPickerButton })
</script>

<template>
  <div class="toolbar">
    <div class="toolbar-left">
      <div class="search">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="11" cy="11" r="6.5"></circle>
          <path d="m16 16 5 5"></path>
        </svg>
        <input
          type="search"
          aria-label="Search services"
          placeholder="Search service or branch…"
          :value="props.searchQuery"
          @input="handleSearchInput"
        />
      </div>

      <div class="filter" role="tablist" aria-label="Repository state">
        <button
          v-for="option in SERVICE_FILTER_OPTIONS"
          :key="option.value"
          class="filter-btn"
          :class="{ 'is-active': props.activeFilter === option.value }"
          type="button"
          role="tab"
          :aria-selected="props.activeFilter === option.value"
          @click="emit('update:activeFilter', option.value)"
        >
          {{ option.label }}
        </button>
      </div>

      <button
        ref="pickerButton"
        class="gear-btn"
        type="button"
        aria-label="Choose repositories"
        title="Choose repositories"
        :aria-expanded="props.isPickerOpen"
        @click="emit('open-picker')"
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.87-.34 1.7 1.7 0 0 0-1.03 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 8.9 19.3a1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.54 15a1.7 1.7 0 0 0-1.56-1.03H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.7 8.9a1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.54a1.7 1.7 0 0 0 1.03-1.56V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15 4.7a1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.46 9v.09A1.7 1.7 0 0 0 21 10.09h.09a2 2 0 1 1 0 4H21a1.7 1.7 0 0 0-1.6 1.03z"></path>
        </svg>
      </button>
    </div>

    <div class="toolbar-right">{{ props.countLabel }}</div>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.search {
  position: relative;
  flex: 1;
  max-width: 340px;
}

.search svg {
  position: absolute;
  left: 11px;
  top: 50%;
  width: 14px;
  height: 14px;
  transform: translateY(-50%);
  stroke: var(--subtle);
  stroke-width: 1.8;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
}

.search input {
  width: 100%;
  height: 36px;
  padding: 0 12px 0 34px;
  border: 1px solid var(--border);
  border-radius: 9px;
  outline: 0;
  background: var(--surface);
  color: var(--text);
  font-size: 12px;
  transition: border-color 160ms ease;
}

.search input::placeholder {
  color: var(--subtle);
}

.search input:focus {
  border-color: var(--border-strong);
}

.filter {
  display: inline-flex;
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
}

.filter-btn {
  padding: 6px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease;
}

.filter-btn:hover {
  background: var(--surface-hover);
  color: var(--text);
}

.filter-btn.is-active {
  background: var(--surface-hover);
  color: var(--text);
}

.gear-btn {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-raised);
  color: var(--muted);
  cursor: pointer;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease;
}

.gear-btn:hover {
  border-color: var(--border-strong);
  background: var(--surface-hover);
  color: var(--text);
}

.gear-btn:focus-visible {
  outline: 2px solid var(--border-strong);
  outline-offset: 2px;
}

.gear-btn svg {
  width: 15px;
  height: 15px;
  stroke: currentColor;
  stroke-width: 1.7;
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.toolbar-right {
  color: var(--subtle);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 820px) {
  .toolbar,
  .toolbar-left {
    flex-direction: column;
    align-items: stretch;
  }

  .search {
    max-width: none;
  }

  .toolbar-left {
    gap: 10px;
  }

  .gear-btn {
    align-self: flex-start;
  }
}
</style>

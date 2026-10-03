<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue"
import type { RepositorySummary } from "../types"
import MicroservicesModal from "./MicroservicesModal.vue"
import RepositoryPickerConfirm from "./RepositoryPickerConfirm.vue"
import RepositoryPickerRow from "./RepositoryPickerRow.vue"
import { SEARCH_DEBOUNCE_MS, useDebouncedValue } from "./debounced-value"

const props = defineProps<{
  repositories: RepositorySummary[]
  selectedNames: string[]
  isLoading: boolean
  isSaving: boolean
  errorMessage: string | null
}>()

const emit = defineEmits<{
  (event: "close"): void
  (event: "apply", repositoryNames: string[]): void
  (event: "retry"): void
}>()

const searchInput = ref<HTMLInputElement | null>(null)
const searchQuery = ref("")
const debouncedSearchQuery = useDebouncedValue(searchQuery, SEARCH_DEBOUNCE_MS)
const draftSelection = ref<string[]>([...props.selectedNames])
const isConfirmingRemoval = ref(false)

const gitRepositoryCount = computed(() => props.repositories.filter((repository) => repository.is_git).length)

const visibleRepositories = computed(() => {
  const query = debouncedSearchQuery.value.trim().toLowerCase()
  if (query.length === 0) {
    return props.repositories
  }
  return props.repositories.filter(
    (repository) =>
      repository.name.toLowerCase().includes(query) || repository.branch.toLowerCase().includes(query)
  )
})

const dirtyUnselectedNames = computed(() =>
  props.repositories
    .filter(
      (repository) =>
        repository.dirty_count > 0 &&
        props.selectedNames.includes(repository.name) &&
        !draftSelection.value.includes(repository.name)
    )
    .map((repository) => repository.name)
)

const handleSearchInput = (event: Event) => {
  const inputElement = event.target as HTMLInputElement
  searchQuery.value = inputElement.value
}

const toggleRepository = (repositoryName: string, isSelected: boolean) => {
  if (isSelected) {
    if (!draftSelection.value.includes(repositoryName)) {
      draftSelection.value = [...draftSelection.value, repositoryName]
    }
    return
  }
  draftSelection.value = draftSelection.value.filter((selectedName) => selectedName !== repositoryName)
}

const selectAllRepositories = () => {
  draftSelection.value = props.repositories
    .filter((repository) => repository.is_git)
    .map((repository) => repository.name)
}

const clearSelection = () => {
  draftSelection.value = []
}

const applySelection = () => {
  emit("apply", [...draftSelection.value])
}

const requestApply = () => {
  if (dirtyUnselectedNames.value.length > 0) {
    isConfirmingRemoval.value = true
    return
  }
  applySelection()
}

const confirmRemoval = () => {
  isConfirmingRemoval.value = false
  applySelection()
}

const keepDirtyRepositories = () => {
  isConfirmingRemoval.value = false
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key !== "Escape") {
    return
  }
  if (isConfirmingRemoval.value) {
    isConfirmingRemoval.value = false
    return
  }
  emit("close")
}

onMounted(() => {
  window.addEventListener("keydown", handleKeydown)
  searchInput.value?.focus()
})

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <MicroservicesModal
    data-testid="repository-picker-overlay"
    eyebrow="Repository root"
    title="Choose repositories"
    title-id="repository-picker-title"
    @close="emit('close')"
  >
    <div class="search-row">
      <div class="search">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="11" cy="11" r="6.5"></circle>
          <path d="m16 16 5 5"></path>
        </svg>
        <input
          ref="searchInput"
          type="search"
          aria-label="Search repositories"
          placeholder="Search repository or branch…"
          :value="searchQuery"
          @input="handleSearchInput"
        />
      </div>

      <div class="bulk-actions">
        <button class="bulk-btn" type="button" @click="selectAllRepositories">Select all</button>
        <button class="bulk-btn" type="button" @click="clearSelection">Clear</button>
      </div>
    </div>

    <div v-if="props.errorMessage" class="picker-error" role="alert">
      <span class="picker-error-text">{{ props.errorMessage }}</span>
      <button class="bulk-btn" type="button" :disabled="props.isLoading" @click="emit('retry')">
        {{ props.isLoading ? "Loading…" : "Retry" }}
      </button>
    </div>

    <div v-if="props.isLoading" class="picker-status">Loading repositories…</div>
    <div v-else-if="props.repositories.length === 0" class="picker-status">
      No repositories were found in the configured folder.
    </div>
    <div v-else-if="visibleRepositories.length === 0" class="picker-status">
      No repository matches that search.
    </div>

    <div v-else class="picker-list">
      <RepositoryPickerRow
        v-for="repository in visibleRepositories"
        :key="repository.name"
        :repository="repository"
        :is-selected="draftSelection.includes(repository.name)"
        @toggle="toggleRepository"
      />
    </div>

    <template #alert>
      <div v-if="isConfirmingRemoval" class="confirm-slot">
        <RepositoryPickerConfirm
          :repository-names="dirtyUnselectedNames"
          @keep="keepDirtyRepositories"
          @confirm="confirmRemoval"
        />
      </div>
    </template>

    <template #footer>
      <span class="footer-count">{{ draftSelection.length }} of {{ gitRepositoryCount }} repositories selected</span>
      <div class="footer-actions">
        <button class="footer-btn is-secondary" type="button" @click="emit('close')">Cancel</button>
        <button class="footer-btn is-primary" type="button" :disabled="props.isSaving" @click="requestApply">
          {{ props.isSaving ? "Applying…" : "Apply" }}
        </button>
      </div>
    </template>
  </MicroservicesModal>
</template>

<style scoped>
.search-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.search { position: relative; flex: 1; min-width: 180px; }
.search svg {
  position: absolute; left: 11px; top: 50%; width: 14px; height: 14px; transform: translateY(-50%);
  stroke: var(--subtle); stroke-width: 1.8; fill: none; stroke-linecap: round; stroke-linejoin: round; pointer-events: none;
}
.search input {
  width: 100%; height: 36px; padding: 0 12px 0 34px; border: 1px solid var(--border); border-radius: 9px;
  outline: 0; background: var(--surface-raised); color: var(--text); font-size: 12px;
}
.search input::placeholder { color: var(--subtle); }
.search input:focus { border-color: var(--border-strong); }

.bulk-actions { display: flex; align-items: center; gap: 6px; }
.bulk-btn {
  min-height: 30px; padding: 0 10px; border: 1px solid var(--border); border-radius: 8px; background: transparent;
  color: var(--muted); font-size: 11px; font-weight: 500; white-space: nowrap; cursor: pointer;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease;
}
.bulk-btn:hover:not(:disabled) { border-color: var(--border-strong); background: var(--surface-hover); color: var(--text); }
.bulk-btn:disabled { opacity: 0.55; cursor: not-allowed; }

.picker-error {
  display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap;
  padding: 11px 13px; border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent); border-radius: 9px;
  background: color-mix(in srgb, var(--danger) 10%, transparent); color: var(--danger); font-size: 11.5px;
}
.picker-error-text { min-width: 0; overflow-wrap: anywhere; }
.picker-status { padding: 26px 8px; color: var(--subtle); font-size: 12px; text-align: center; }
.picker-list { display: grid; gap: 7px; }
.confirm-slot { padding: 0 20px 14px; }

.footer-count { color: var(--muted); font-size: 11px; font-variant-numeric: tabular-nums; }
.footer-actions { display: flex; align-items: center; gap: 8px; }
.footer-btn {
  display: inline-flex; align-items: center; justify-content: center; min-height: 34px; padding: 0 14px;
  border: 1px solid transparent; border-radius: 8px; font-size: 11.5px; font-weight: 600; white-space: nowrap;
  cursor: pointer; transition: opacity 150ms ease, border-color 150ms ease, background-color 150ms ease;
}
.footer-btn.is-primary { background: var(--accent); color: var(--accent-contrast); }
.footer-btn.is-primary:hover:not(:disabled) { opacity: 0.9; }
.footer-btn.is-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.footer-btn.is-secondary { border-color: var(--border); background: transparent; color: var(--text); }
.footer-btn.is-secondary:hover { border-color: var(--border-strong); background: var(--surface-hover); }

@media (max-width: 620px) {
  .footer-actions { flex-direction: column; align-items: stretch; }
  .footer-btn { width: 100%; }
}
</style>

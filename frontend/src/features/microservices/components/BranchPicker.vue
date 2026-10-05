<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import type { BranchOption } from "../types"
import { useAnchoredPopup } from "./anchored-popup"
import { useRepositoryActions } from "../composables/useRepositoryActions"

const props = defineProps<{
  repositoryName: string
  branch: string
}>()

const { resolveActionState, loadBranches, checkoutBranch } = useRepositoryActions()

const VISIBLE_BRANCH_ROW_COUNT = 5
const BRANCH_ROW_HEIGHT_PX = 34
const ESTIMATED_POPUP_HEIGHT_PX = 216
const ESTIMATED_POPUP_WIDTH_PX = 280
const BRANCH_LIST_FAILED_MESSAGE = "The branch list could not be loaded."

const { isOpen, placement, popupStyle, triggerElement, popupElement, open, close, updateAnchor } =
  useAnchoredPopup({
    estimatedHeight: ESTIMATED_POPUP_HEIGHT_PX,
    estimatedWidth: ESTIMATED_POPUP_WIDTH_PX,
    align: "start"
  })

const branches = ref<BranchOption[]>([])
const searchQuery = ref("")
const isLoadingBranches = ref(false)
const branchListFailedMessage = ref("")
const searchInputElement = ref<HTMLInputElement | null>(null)
const branchListElement = ref<HTMLElement | null>(null)

const isBusy = computed(() => resolveActionState(props.repositoryName).status === "running")

const orderedBranches = computed<BranchOption[]>(() => [
  ...branches.value.filter((entry) => entry.is_current),
  ...branches.value.filter((entry) => !entry.is_current)
])

const filteredBranches = computed<BranchOption[]>(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (query.length === 0) {
    return orderedBranches.value
  }
  return orderedBranches.value.filter((entry) => entry.name.toLowerCase().includes(query))
})

const branchListMaxHeight = computed(() => `${VISIBLE_BRANCH_ROW_COUNT * BRANCH_ROW_HEIGHT_PX}px`)

const loadBranchOptions = async (): Promise<void> => {
  isLoadingBranches.value = true
  branchListFailedMessage.value = ""
  const loadedBranches = await loadBranches(props.repositoryName)
  isLoadingBranches.value = false

  if (loadedBranches === null) {
    branchListFailedMessage.value = BRANCH_LIST_FAILED_MESSAGE
    return
  }

  branches.value = loadedBranches
  await nextTick()
  updateAnchor()
}

const openPicker = async (): Promise<void> => {
  if (isOpen.value || isBusy.value) {
    return
  }
  searchQuery.value = ""
  branchListFailedMessage.value = ""
  branches.value = []
  await open()
  searchInputElement.value?.focus()
  await loadBranchOptions()
}

const closePicker = (): void => {
  close(true)
}

const togglePicker = async (): Promise<void> => {
  if (isOpen.value) {
    closePicker()
    return
  }
  await openPicker()
}

const selectBranch = async (branchName: string): Promise<void> => {
  const selectedBranch = branches.value.find((entry) => entry.name === branchName)
  if (selectedBranch === undefined || selectedBranch.is_current || isBusy.value) {
    closePicker()
    return
  }
  closePicker()
  await checkoutBranch(props.repositoryName, branchName)
}

const focusFirstBranch = (): void => {
  branchListElement.value?.querySelector<HTMLButtonElement>("button")?.focus()
}

const focusSiblingBranch = (event: KeyboardEvent, offset: number): void => {
  const target = event.target
  if (!(target instanceof HTMLButtonElement)) {
    return
  }
  const listElement = target.closest(".branch-list")
  if (listElement === null) {
    return
  }
  const branchButtons = Array.from(listElement.querySelectorAll<HTMLButtonElement>("button"))
  const currentIndex = branchButtons.indexOf(target)
  if (currentIndex === -1) {
    return
  }
  branchButtons[(currentIndex + offset + branchButtons.length) % branchButtons.length]?.focus()
}
</script>

<template>
  <span class="branch-picker">
    <button
      ref="triggerElement"
      class="branch-pill"
      type="button"
      :class="{ 'is-busy': isBusy }"
      :aria-disabled="isBusy"
      :aria-expanded="isOpen"
      aria-haspopup="listbox"
      :aria-label="`Branch ${props.branch} for ${props.repositoryName}. Choose a different branch`"
      @click.stop="togglePicker"
    >
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="6" cy="6" r="2.5"></circle>
        <circle cx="6" cy="18" r="2.5"></circle>
        <circle cx="18" cy="12" r="2.5"></circle>
        <path d="M6 8.5v7"></path>
        <path d="M8.5 6h4a3 3 0 0 1 3 3v1"></path>
      </svg>
      <span class="branch-pill-name">{{ props.branch }}</span>
      <svg class="branch-pill-caret" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M6 9l6 6 6-6"></path>
      </svg>
    </button>

    <Teleport to="body">
      <div
        v-if="isOpen"
        ref="popupElement"
        class="branch-popup"
        :class="`is-${placement}`"
        :style="popupStyle"
      >
        <div class="branch-search">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="11" cy="11" r="7"></circle>
            <path d="m16.5 16.5 4 4"></path>
          </svg>
          <input
            ref="searchInputElement"
            v-model="searchQuery"
            class="branch-search-input"
            type="search"
            placeholder="Search branches"
            aria-label="Search branches"
            autocomplete="off"
            @keydown.down.prevent="focusFirstBranch"
          />
        </div>

        <div v-if="isLoadingBranches" class="branch-status">Loading branches…</div>

        <div v-else-if="branchListFailedMessage.length > 0" class="branch-status is-failed">
          <span>{{ branchListFailedMessage }}</span>
          <button class="branch-status-action" type="button" @click="loadBranchOptions">Retry</button>
        </div>

        <div v-else-if="filteredBranches.length === 0" class="branch-status">
          No branch matches “{{ searchQuery }}”.
        </div>

        <div
          v-else
          ref="branchListElement"
          class="branch-list"
          role="listbox"
          aria-label="Branches"
          :style="{ maxHeight: branchListMaxHeight }"
          @keydown.down.prevent="focusSiblingBranch($event, 1)"
          @keydown.up.prevent="focusSiblingBranch($event, -1)"
        >
          <button
            v-for="entry in filteredBranches"
            :key="entry.name"
            class="branch-row"
            :class="{ 'is-current': entry.is_current }"
            type="button"
            role="option"
            :aria-selected="entry.is_current"
            :aria-label="entry.is_current ? `${entry.name} (current branch)` : entry.name"
            @click="selectBranch(entry.name)"
          >
            <svg class="branch-row-icon" viewBox="0 0 24 24" aria-hidden="true">
              <path v-if="entry.is_current" d="M4.5 12.5l5 5L19.5 6.5"></path>
              <circle v-else cx="12" cy="12" r="3"></circle>
            </svg>
            <span class="branch-row-name">{{ entry.name }}</span>
            <span v-if="entry.updated_relative" class="branch-row-updated">
              {{ entry.updated_relative }}
            </span>
          </button>
        </div>
      </div>
    </Teleport>
  </span>
</template>

<style scoped>
.branch-picker { display: inline-flex; max-width: 100%; }

.branch-pill {
  display: inline-flex; align-items: center; gap: 5px; max-width: 100%; padding: 3px 7px; border: 1px solid var(--border);
  border-radius: 6px; background: var(--surface-raised); color: var(--muted); font-size: 10.5px; cursor: pointer;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  transition: border-color 150ms ease, color 150ms ease;
}
.branch-pill:hover { border-color: var(--border-strong); color: var(--text); }
.branch-pill:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.branch-pill.is-busy { opacity: 0.55; cursor: progress; }
.branch-pill svg {
  width: 10px; height: 10px; flex: 0 0 auto; stroke: currentColor; stroke-width: 2; fill: none;
  stroke-linecap: round; stroke-linejoin: round;
}
.branch-pill-name { min-width: 0; overflow-wrap: anywhere; }
.branch-pill-caret { opacity: 0.7; transition: transform 150ms ease; }
.branch-pill[aria-expanded="true"] .branch-pill-caret { transform: rotate(180deg); }

.branch-popup {
  position: fixed; z-index: 50; display: flex; flex-direction: column; width: max-content; min-width: 240px;
  max-width: 320px; border: 1px solid var(--border-strong); border-radius: 10px; background: var(--surface);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.45); overflow: hidden; animation: branch-popup-in 130ms ease;
}
@keyframes branch-popup-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

.branch-search {
  display: flex; align-items: center; gap: 7px; flex: 0 0 auto; padding: 9px 11px; border-bottom: 1px solid var(--border);
}
.branch-search svg {
  width: 12px; height: 12px; flex: 0 0 auto; stroke: var(--subtle); stroke-width: 2; fill: none;
  stroke-linecap: round; stroke-linejoin: round;
}
.branch-search-input {
  width: 100%; min-width: 0; border: 0; background: transparent; color: var(--text); font-size: 11.5px; outline: none;
}
.branch-search-input::placeholder { color: var(--subtle); }

.branch-list { overflow-y: auto; padding: 4px; }
.branch-row {
  display: flex; align-items: center; gap: 8px; width: 100%; min-height: 34px; padding: 0 8px; border: 0;
  border-radius: 7px; background: transparent; color: var(--text); font-size: 11.5px; text-align: left; cursor: pointer;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
.branch-row:hover { background: var(--surface-hover); }
.branch-row:focus-visible { outline: 2px solid var(--accent); outline-offset: -2px; }
.branch-row.is-current { position: sticky; top: 0; background: var(--surface-raised); }
.branch-row.is-current:hover { background: var(--surface-hover); }
.branch-row-icon {
  width: 11px; height: 11px; flex: 0 0 auto; stroke: var(--subtle); stroke-width: 2; fill: none;
  stroke-linecap: round; stroke-linejoin: round;
}
.branch-row.is-current .branch-row-icon { stroke: var(--positive); }
.branch-row-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.branch-row-updated { margin-left: auto; color: var(--subtle); font-size: 10px; white-space: nowrap; }

.branch-status {
  display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 13px 12px;
  color: var(--subtle); font-size: 11px;
}
.branch-status.is-failed { color: var(--danger); }
.branch-status-action {
  flex: 0 0 auto; min-height: 26px; padding: 0 10px; border: 1px solid currentColor; border-radius: 7px;
  background: transparent; color: inherit; font-size: 10.5px; font-weight: 600; cursor: pointer;
}
</style>

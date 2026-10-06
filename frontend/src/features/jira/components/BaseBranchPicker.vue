<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import type { RepositoryBaseBranchOptions } from "../branch-types"
import { baseBranchGroups, emptyBaseBranchMessage } from "../logic/base-branch-groups"
import { useAnchoredPopup } from "../../microservices/components/anchored-popup"

const props = defineProps<{
  repo: string
  selected: string
  options: RepositoryBaseBranchOptions | null
  isLoading: boolean
  errorMessage: string
}>()

const emit = defineEmits<{
  (event: "select", base: string): void
  (event: "load"): void
}>()

const ESTIMATED_POPUP_HEIGHT_PX = 320
const ESTIMATED_POPUP_WIDTH_PX = 360

const { isOpen, placement, popupStyle, triggerElement, popupElement, open, close, updateAnchor } =
  useAnchoredPopup({
    estimatedHeight: ESTIMATED_POPUP_HEIGHT_PX,
    estimatedWidth: ESTIMATED_POPUP_WIDTH_PX,
    align: "end"
  })

const searchTerm = ref("")
const searchInputElement = ref<HTMLInputElement | null>(null)
const optionListElement = ref<HTMLElement | null>(null)

const availableOptions = computed(() => props.options?.options ?? [])
const groups = computed(() => baseBranchGroups(availableOptions.value, searchTerm.value))
const matchingCount = computed(() =>
  groups.value.reduce((total, group) => total + group.options.length, 0)
)
const emptyMessage = computed(() =>
  emptyBaseBranchMessage(props.repo, searchTerm.value, availableOptions.value.length)
)
const selectedOption = computed(
  () => availableOptions.value.find((option) => option.name === props.selected) ?? null
)
const triggerLabel = computed(() => (props.selected.length > 0 ? props.selected : "Choose a base branch"))

const focusFirstOption = (): void => {
  optionListElement.value?.querySelector<HTMLButtonElement>("button")?.focus()
}

const focusSiblingOption = (event: KeyboardEvent, offset: number): void => {
  const target = event.target
  if (!(target instanceof HTMLButtonElement)) {
    return
  }
  const listElement = target.closest(".popup-list")
  if (listElement === null) {
    return
  }
  const optionButtons = Array.from(listElement.querySelectorAll<HTMLButtonElement>("button.option"))
  const currentIndex = optionButtons.indexOf(target)
  if (currentIndex === -1) {
    return
  }
  optionButtons[(currentIndex + offset + optionButtons.length) % optionButtons.length]?.focus()
}

const openPicker = async (): Promise<void> => {
  searchTerm.value = ""
  emit("load")
  await open()
  await nextTick()
  searchInputElement.value?.focus()
  updateAnchor()
}

const togglePicker = async (): Promise<void> => {
  if (isOpen.value) {
    close(true)
    return
  }
  await openPicker()
}

const selectOption = (name: string): void => {
  emit("select", name)
  close(true)
}

watch(availableOptions, async () => {
  await nextTick()
  updateAnchor()
})
</script>

<template>
  <span class="base-picker">
    <button
      ref="triggerElement"
      class="base-trigger"
      type="button"
      aria-haspopup="listbox"
      :aria-expanded="isOpen"
      :aria-label="`Base branch for ${repo}: ${triggerLabel}. Choose a different branch`"
      @click.stop="togglePicker"
    >
      <span v-if="selected.length > 0" class="base-trigger-name">{{ selected }}</span>
      <span v-else class="base-trigger-placeholder">Choose a base branch</span>
      <span v-if="selectedOption?.source === 'remembered'" class="tag is-remembered">remembered</span>
      <span v-else-if="selectedOption?.source === 'bitbucket_default'" class="tag is-default">default</span>
      <span class="caret" aria-hidden="true">▾</span>
    </button>

    <Teleport to="body">
      <div
        v-if="isOpen"
        ref="popupElement"
        class="base-popup"
        :class="`is-${placement}`"
        :style="popupStyle"
        data-testid="base-branch-popup"
      >
        <div class="popup-head">
          <input
            ref="searchInputElement"
            v-model="searchTerm"
            class="search"
            type="search"
            placeholder="Search branches"
            :aria-label="`Search branches in ${repo}`"
            autocomplete="off"
            data-testid="base-branch-search"
            @keydown.down.prevent="focusFirstOption"
          />
        </div>

        <p v-if="isLoading" class="popup-status">Loading branches…</p>

        <div v-else-if="errorMessage.length > 0" class="popup-status is-failed">
          <span>{{ errorMessage }}</span>
          <button class="link-btn" type="button" @click="emit('load')">Retry</button>
        </div>

        <p v-else-if="matchingCount === 0" class="option-empty" data-testid="base-branch-empty">
          {{ emptyMessage }}
        </p>

        <div
          v-else
          ref="optionListElement"
          class="popup-list"
          role="listbox"
          :aria-label="`Base branches in ${repo}`"
          @keydown.down.prevent="focusSiblingOption($event, 1)"
          @keydown.up.prevent="focusSiblingOption($event, -1)"
        >
          <div
            v-for="group in groups"
            :key="group.label"
            role="group"
            :aria-label="group.label.length > 0 ? group.label : `Branches in ${repo}`"
          >
            <div v-if="group.label.length > 0" class="group-label">{{ group.label }}</div>
            <button
              v-for="option in group.options"
              :key="option.name"
              class="option"
              type="button"
              role="option"
              :aria-selected="selected === option.name"
              :data-testid="`base-branch-option-${option.name}`"
              @click="selectOption(option.name)"
            >
              <span class="oname">{{ option.name }}</span>
              <span v-if="option.source === 'remembered'" class="tag is-remembered">remembered</span>
              <span v-else-if="option.source === 'bitbucket_default'" class="tag is-default">default</span>
              <span v-if="option.last_commit_relative !== null" class="owhen">
                {{ option.last_commit_relative }}
              </span>
              <span v-else class="ostale">empty</span>
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </span>
</template>

<style scoped src="./base-branch-picker.css"></style>

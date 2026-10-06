<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue"
import { useAnchoredPopup } from "../../microservices/components/anchored-popup"
import { useDebouncedValue } from "../../microservices/components/debounced-value"
import { useRepositoryCatalogue } from "../composables/useRepositoryCatalogue"

const props = defineProps<{
  takenRepos: string[]
}>()

const emit = defineEmits<{
  (event: "add", repo: string, isCloned: boolean): void
}>()

const ESTIMATED_POPUP_HEIGHT_PX = 320
const ESTIMATED_POPUP_WIDTH_PX = 360

const {
  repositories,
  localRepositoryCount,
  remoteRepositoryCount,
  catalogueError,
  isLoadingCatalogue,
  searchRepositories
} = useRepositoryCatalogue()

const { isOpen, placement, popupStyle, triggerElement, popupElement, open, close, updateAnchor } =
  useAnchoredPopup({
    estimatedHeight: ESTIMATED_POPUP_HEIGHT_PX,
    estimatedWidth: ESTIMATED_POPUP_WIDTH_PX,
    align: "start"
  })

const searchTerm = ref("")
const debouncedSearchTerm = useDebouncedValue(searchTerm, 200)
const searchInputElement = ref<HTMLInputElement | null>(null)
const optionListElement = ref<HTMLElement | null>(null)

const availableRepositories = computed(() =>
  repositories.value.filter((repository) => !props.takenRepos.includes(repository.repo))
)
const emptyMessage = computed(() => `No repository matches "${searchTerm.value}"`)
const catalogueSummary = computed(
  () => `The workspace has ${localRepositoryCount.value} of ${remoteRepositoryCount.value} repositories.`
)

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
  await open()
  await nextTick()
  searchInputElement.value?.focus()
  updateAnchor()
  await searchRepositories("")
}

const togglePicker = async (): Promise<void> => {
  if (isOpen.value) {
    close(true)
    return
  }
  await openPicker()
}

const selectRepository = (repo: string, isCloned: boolean): void => {
  emit("add", repo, isCloned)
  close(true)
}

watch(debouncedSearchTerm, (term) => {
  if (!isOpen.value) {
    return
  }
  void searchRepositories(term)
})

watch(availableRepositories, async () => {
  await nextTick()
  updateAnchor()
})
</script>

<template>
  <div class="add-row">
    <span class="repository-picker">
      <button
        ref="triggerElement"
        class="trigger add"
        type="button"
        aria-haspopup="listbox"
        :aria-expanded="isOpen"
        data-testid="add-repository"
        @click.stop="togglePicker"
      >
        <span class="plus" aria-hidden="true">+</span>
        <span>Add repository</span>
      </button>

      <Teleport to="body">
        <div
          v-if="isOpen"
          ref="popupElement"
          class="repository-popup"
          :class="`is-${placement}`"
          :style="popupStyle"
          data-testid="repository-popup"
        >
          <div class="popup-head">
            <input
              ref="searchInputElement"
              v-model="searchTerm"
              class="search"
              type="search"
              placeholder="Search all repositories"
              aria-label="Search all repositories"
              autocomplete="off"
              data-testid="repository-search"
              @keydown.down.prevent="focusFirstOption"
            />
          </div>

          <p v-if="isLoadingCatalogue" class="popup-status">Loading repositories…</p>

          <div v-else-if="catalogueError.length > 0" class="popup-status is-failed">
            <span>{{ catalogueError }}</span>
            <button class="link-btn" type="button" @click="searchRepositories(searchTerm)">Retry</button>
          </div>

          <template v-else-if="availableRepositories.length === 0">
            <p class="option-empty" data-testid="repository-empty">{{ emptyMessage }}</p>
            <p class="option-note">{{ catalogueSummary }}</p>
          </template>

          <div
            v-else
            ref="optionListElement"
            class="popup-list"
            role="listbox"
            aria-label="Repositories"
            @keydown.down.prevent="focusSiblingOption($event, 1)"
            @keydown.up.prevent="focusSiblingOption($event, -1)"
          >
            <button
              v-for="repository in availableRepositories"
              :key="repository.repo"
              class="option"
              type="button"
              role="option"
              :aria-selected="false"
              :data-testid="`repository-option-${repository.repo}`"
              @click="selectRepository(repository.repo, repository.is_cloned)"
            >
              <span class="oname">{{ repository.repo }}</span>
              <span class="tag">{{ repository.project }}</span>
              <span v-if="!repository.is_cloned" class="tag">not cloned</span>
            </button>
          </div>
        </div>
      </Teleport>
    </span>

    <span class="hint">Lunar suggests from the acceptance criteria. Add one it missed.</span>
  </div>
</template>

<style scoped src="./repository-picker.css"></style>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import ConfirmDialog from "../../../components/ui/ConfirmDialog.vue"
import { useAnchoredPopup } from "./anchored-popup"
import { useRepositoryActions } from "../composables/useRepositoryActions"

const props = defineProps<{
  repositoryName: string
  isDetailsOpen: boolean
  isBusy: boolean
  failureMessage: string
}>()

const emit = defineEmits<{
  (event: "toggle-details"): void
}>()

const { fetchRepository, pullRepository, stashRepository, discardRepository } = useRepositoryActions()

const ESTIMATED_MENU_HEIGHT_PX = 188
const ESTIMATED_MENU_WIDTH_PX = 208

const { isOpen, placement, popupStyle, triggerElement, popupElement, open, close } = useAnchoredPopup({
  estimatedHeight: ESTIMATED_MENU_HEIGHT_PX,
  estimatedWidth: ESTIMATED_MENU_WIDTH_PX,
  align: "end"
})

const isDiscardConfirmOpen = ref(false)

const discardMessage = computed(
  () =>
    `Every tracked file in ${props.repositoryName} is restored to the last commit. Untracked files stay where they are, and this cannot be undone.`
)

const closeMenu = (): void => {
  close(true)
}

const openMenu = async (): Promise<void> => {
  if (props.isBusy || isOpen.value) {
    return
  }
  await open()
  popupElement.value?.querySelector<HTMLButtonElement>("button:not(:disabled)")?.focus()
}

const runMenuAction = (run: () => Promise<boolean>): void => {
  closeMenu()
  void run()
}

const selectDetails = (): void => {
  closeMenu()
  emit("toggle-details")
}

const requestDiscard = (): void => {
  closeMenu()
  isDiscardConfirmOpen.value = true
}

const closeDiscardConfirm = (): void => {
  isDiscardConfirmOpen.value = false
  triggerElement.value?.focus()
}

const confirmDiscard = (): void => {
  closeDiscardConfirm()
  void discardRepository(props.repositoryName)
}

const handleConfirmKeydown = (event: KeyboardEvent): void => {
  if (event.key !== "Escape") {
    return
  }
  closeDiscardConfirm()
}

watch(isDiscardConfirmOpen, (isConfirmOpen) => {
  if (isConfirmOpen) {
    document.addEventListener("keydown", handleConfirmKeydown, true)
    return
  }
  document.removeEventListener("keydown", handleConfirmKeydown, true)
})

onBeforeUnmount(() => {
  document.removeEventListener("keydown", handleConfirmKeydown, true)
})

const focusSiblingItem = (event: KeyboardEvent, offset: number): void => {
  const target = event.target
  if (!(target instanceof HTMLButtonElement)) {
    return
  }
  const menuElement = target.closest(".action-menu")
  if (menuElement === null) {
    return
  }
  const menuItems = Array.from(menuElement.querySelectorAll<HTMLButtonElement>("button:not(:disabled)"))
  const currentIndex = menuItems.indexOf(target)
  if (currentIndex === -1) {
    return
  }
  menuItems[(currentIndex + offset + menuItems.length) % menuItems.length]?.focus()
}
</script>

<template>
  <div class="action-cell">
    <button
      ref="triggerElement"
      class="kebab-trigger"
      type="button"
      :class="{ 'is-busy': props.isBusy }"
      :aria-disabled="props.isBusy"
      :aria-expanded="isOpen"
      aria-haspopup="menu"
      :aria-label="`Repository actions for ${props.repositoryName}`"
      @click.stop="openMenu"
    >
      <svg v-if="props.isBusy" class="kebab-spinner" viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="12" r="9"></circle>
        <path d="M21 12a9 9 0 0 0-9-9"></path>
      </svg>
      <svg v-else viewBox="0 0 24 24" aria-hidden="true">
        <circle cx="12" cy="5.5" r="1.5"></circle>
        <circle cx="12" cy="12" r="1.5"></circle>
        <circle cx="12" cy="18.5" r="1.5"></circle>
      </svg>
      <span
        v-if="props.failureMessage.length > 0"
        class="kebab-alert"
        role="img"
        :aria-label="`The last repository action failed: ${props.failureMessage}`"
        :title="props.failureMessage"
      ></span>
    </button>

    <Teleport to="body">
      <div
        v-if="isOpen"
        ref="popupElement"
        class="action-menu"
        :class="`is-${placement}`"
        :style="popupStyle"
        role="menu"
        :aria-label="`Actions for ${props.repositoryName}`"
        @keydown.down.prevent="focusSiblingItem($event, 1)"
        @keydown.up.prevent="focusSiblingItem($event, -1)"
      >
        <button
          class="action-item"
          type="button"
          role="menuitem"
          :disabled="props.isBusy"
          @click="runMenuAction(() => pullRepository(props.repositoryName))"
        >
          <svg class="action-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M12 3v11"></path>
            <path d="m7.5 9.5 4.5 4.5 4.5-4.5"></path>
            <path d="M4 20h16"></path>
          </svg>
          <span>Pull</span>
        </button>

        <button
          class="action-item"
          type="button"
          role="menuitem"
          :disabled="props.isBusy"
          @click="runMenuAction(() => fetchRepository(props.repositoryName))"
        >
          <svg class="action-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M20.5 4.5v5h-5"></path>
            <path d="M3.5 19.5v-5h5"></path>
            <path d="M4.6 9.1a8 8 0 0 1 13.2-2.9l2.7 2.6"></path>
            <path d="M19.4 14.9a8 8 0 0 1-13.2 2.9l-2.7-2.6"></path>
          </svg>
          <span>Fetch</span>
        </button>

        <button
          class="action-item"
          type="button"
          role="menuitem"
          :disabled="props.isBusy"
          @click="runMenuAction(() => stashRepository(props.repositoryName))"
        >
          <svg class="action-icon" viewBox="0 0 24 24" aria-hidden="true">
            <rect x="3" y="4" width="18" height="4.5" rx="1.2"></rect>
            <path d="M5.2 8.5v10a1.5 1.5 0 0 0 1.5 1.5h10.6a1.5 1.5 0 0 0 1.5-1.5v-10"></path>
            <path d="M12 11.5v5"></path>
            <path d="m9.6 14.1 2.4 2.4 2.4-2.4"></path>
          </svg>
          <span>Stash changes local</span>
        </button>

        <button
          class="action-item is-danger"
          type="button"
          role="menuitem"
          :disabled="props.isBusy"
          @click="requestDiscard"
        >
          <svg class="action-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M4 12a8 8 0 1 0 2.4-5.7"></path>
            <path d="M3.4 4.6v5h5"></path>
            <path d="m9.8 9.8 4.4 4.4"></path>
            <path d="m14.2 9.8-4.4 4.4"></path>
          </svg>
          <span>Discard changes</span>
        </button>

        <button
          class="action-item"
          type="button"
          role="menuitem"
          :aria-expanded="props.isDetailsOpen"
          @click="selectDetails"
        >
          <svg class="action-icon" :class="{ 'is-expanded': props.isDetailsOpen }" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M6 9l6 6 6-6"></path>
          </svg>
          <span>Details</span>
        </button>
      </div>
    </Teleport>

    <div class="confirm-layer" @click.stop>
      <ConfirmDialog
        :open="isDiscardConfirmOpen"
        tone="danger" title="Discard local changes?"
        :message="discardMessage"
        confirm-label="Discard changes"
        cancel-label="Cancel"
        @confirm="confirmDiscard"
        @cancel="closeDiscardConfirm"
      />
    </div>
  </div>
</template>

<style scoped>
.action-cell { display: flex; justify-content: flex-end; }

.kebab-trigger {
  position: relative; display: grid; place-items: center; width: 30px; height: 30px; border: 1px solid var(--border);
  border-radius: 8px; background: transparent; color: var(--muted); cursor: pointer;
  transition: background-color 150ms ease, color 150ms ease, border-color 150ms ease;
}
.kebab-trigger:hover { border-color: var(--border-strong); background: var(--surface-hover); color: var(--text); }
.kebab-trigger:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
.kebab-trigger.is-busy { cursor: progress; color: var(--text); }
.kebab-trigger svg {
  width: 14px; height: 14px; stroke: currentColor; stroke-width: 2; fill: none; stroke-linecap: round;
  stroke-linejoin: round;
}
.kebab-trigger svg circle { fill: currentColor; stroke: none; }
.kebab-trigger .kebab-spinner { animation: kebab-spin 900ms linear infinite; }
.kebab-trigger .kebab-spinner circle { fill: none; stroke: currentColor; stroke-width: 2; opacity: 0.25; }
@keyframes kebab-spin {
  to { transform: rotate(360deg); }
}
.kebab-alert {
  position: absolute; top: -3px; right: -3px; width: 8px; height: 8px; border-radius: 50%;
  background: var(--danger); box-shadow: 0 0 0 2px var(--surface);
}

.action-menu {
  position: fixed; z-index: 50; display: flex; flex-direction: column; gap: 1px; width: max-content;
  min-width: 192px; max-width: 260px; padding: 5px; border: 1px solid var(--border-strong); border-radius: 10px;
  background: var(--surface); box-shadow: 0 18px 44px rgba(0, 0, 0, 0.45);
  animation: action-menu-in 130ms ease;
}
@keyframes action-menu-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

.action-item {
  display: flex; align-items: center; gap: 9px; width: 100%; min-height: 34px; padding: 0 9px; border: 0;
  border-radius: 7px; background: transparent; color: var(--text); font-size: 11.5px; text-align: left; cursor: pointer;
  white-space: nowrap;
}
.action-item:hover:not(:disabled) { background: var(--surface-hover); }
.action-item:focus-visible { outline: 2px solid var(--accent); outline-offset: -2px; }
.action-item:disabled { color: var(--subtle); cursor: not-allowed; }
.action-item.is-danger { color: var(--danger); }
.action-item.is-danger:hover:not(:disabled) { background: color-mix(in srgb, var(--danger) 14%, transparent); }
.action-icon {
  width: 13px; height: 13px; flex: 0 0 auto; stroke: currentColor; stroke-width: 1.8; fill: none;
  stroke-linecap: round; stroke-linejoin: round; transition: transform 180ms ease;
}
.action-icon.is-expanded { transform: rotate(180deg); }
</style>

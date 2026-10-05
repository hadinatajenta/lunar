import { nextTick, onBeforeUnmount, ref, watch, type CSSProperties } from "vue"

export type PopupPlacement = "dropdown" | "dropup"

interface AnchoredPopupConfig {
  estimatedHeight?: number
  estimatedWidth?: number
  align?: "start" | "end"
}

const VIEWPORT_GAP_PX = 8
const DEFAULT_ESTIMATED_SIZE_PX = 240

export function useAnchoredPopup(config: AnchoredPopupConfig = {}) {
  const estimatedHeight = config.estimatedHeight ?? DEFAULT_ESTIMATED_SIZE_PX
  const estimatedWidth = config.estimatedWidth ?? DEFAULT_ESTIMATED_SIZE_PX

  const isOpen = ref(false)
  const placement = ref<PopupPlacement>("dropdown")
  const popupStyle = ref<CSSProperties>({})
  const triggerElement = ref<HTMLElement | null>(null)
  const popupElement = ref<HTMLElement | null>(null)

  const updateAnchor = (): void => {
    const trigger = triggerElement.value
    if (trigger === null) {
      return
    }
    const triggerRect = trigger.getBoundingClientRect()
    const popupHeight = popupElement.value?.offsetHeight ?? estimatedHeight
    const popupWidth = popupElement.value?.offsetWidth ?? estimatedWidth
    const spaceBelow = window.innerHeight - triggerRect.bottom - VIEWPORT_GAP_PX
    const spaceAbove = triggerRect.top - VIEWPORT_GAP_PX
    const opensUpward = spaceBelow < popupHeight && spaceAbove > spaceBelow
    placement.value = opensUpward ? "dropup" : "dropdown"
    const anchoredLeft = config.align === "end" ? triggerRect.right - popupWidth : triggerRect.left
    const left = Math.max(
      VIEWPORT_GAP_PX,
      Math.min(anchoredLeft, window.innerWidth - popupWidth - VIEWPORT_GAP_PX)
    )
    popupStyle.value = opensUpward
      ? { left: `${left}px`, bottom: `${window.innerHeight - triggerRect.top + VIEWPORT_GAP_PX}px` }
      : { left: `${left}px`, top: `${triggerRect.bottom + VIEWPORT_GAP_PX}px` }
  }

  const close = (shouldRestoreFocus = true): void => {
    if (!isOpen.value) {
      return
    }
    isOpen.value = false
    if (shouldRestoreFocus) {
      triggerElement.value?.focus()
    }
  }

  const open = async (): Promise<void> => {
    if (isOpen.value) {
      return
    }
    isOpen.value = true
    await nextTick()
    updateAnchor()
  }

  const handlePointerDown = (event: PointerEvent): void => {
    const target = event.target
    if (!(target instanceof Node)) {
      return
    }
    const isInsideTrigger = triggerElement.value?.contains(target) === true
    const isInsidePopup = popupElement.value?.contains(target) === true
    if (isInsideTrigger || isInsidePopup) {
      return
    }
    close()
  }

  const handleKeydown = (event: KeyboardEvent): void => {
    if (event.key !== "Escape") {
      return
    }
    event.stopPropagation()
    close()
  }

  const handleViewportChange = (): void => {
    updateAnchor()
  }

  const bindListeners = (): void => {
    document.addEventListener("pointerdown", handlePointerDown, true)
    document.addEventListener("keydown", handleKeydown, true)
    window.addEventListener("scroll", handleViewportChange, true)
    window.addEventListener("resize", handleViewportChange)
  }

  const unbindListeners = (): void => {
    document.removeEventListener("pointerdown", handlePointerDown, true)
    document.removeEventListener("keydown", handleKeydown, true)
    window.removeEventListener("scroll", handleViewportChange, true)
    window.removeEventListener("resize", handleViewportChange)
  }

  watch(isOpen, (isPopupOpen) => {
    if (isPopupOpen) {
      bindListeners()
      return
    }
    unbindListeners()
  })

  onBeforeUnmount(unbindListeners)

  return {
    isOpen,
    placement,
    popupStyle,
    triggerElement,
    popupElement,
    updateAnchor,
    open,
    close
  }
}

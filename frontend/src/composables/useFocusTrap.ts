import { nextTick, onBeforeUnmount, ref, watch, type Ref } from "vue"

const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

export const useFocusTrap = (container: Ref<HTMLElement | null>, isActive: Ref<boolean>) => {
  const openerElement = ref<HTMLElement | null>(null)

  const collectFocusableElements = (): HTMLElement[] => {
    const root = container.value
    if (root === null) return []
    return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter(
      (element) => element.getClientRects().length > 0
    )
  }

  const handleKeydown = (event: KeyboardEvent): void => {
    if (event.key !== "Tab") return

    const focusableElements = collectFocusableElements()
    if (focusableElements.length === 0) return

    const firstElement = focusableElements[0]
    const lastElement = focusableElements[focusableElements.length - 1]
    const activeElement = document.activeElement
    const root = container.value
    const isOutside = root === null || !root.contains(activeElement)

    if (event.shiftKey && (activeElement === firstElement || isOutside)) {
      event.preventDefault()
      lastElement.focus()
      return
    }
    if (!event.shiftKey && (activeElement === lastElement || isOutside)) {
      event.preventDefault()
      firstElement.focus()
    }
  }

  const releaseFocusTrap = (): void => {
    window.removeEventListener("keydown", handleKeydown, true)

    const opener = openerElement.value
    openerElement.value = null
    if (opener !== null && document.contains(opener)) {
      opener.focus()
    }
  }

  watch(
    isActive,
    async (active) => {
      if (!active) {
        releaseFocusTrap()
        return
      }

      openerElement.value = document.activeElement instanceof HTMLElement ? document.activeElement : null
      window.addEventListener("keydown", handleKeydown, true)
      await nextTick()

      const focusableElements = collectFocusableElements()
      const target = focusableElements[0] ?? container.value
      target?.focus()
    },
    { immediate: true }
  )

  onBeforeUnmount(releaseFocusTrap)

  return { collectFocusableElements }
}

import { ref, watch } from "vue"

const MENU_COLLAPSED_STORAGE_KEY = "menu_sidebar_collapsed"

const readStoredCollapsed = (): boolean => localStorage.getItem(MENU_COLLAPSED_STORAGE_KEY) === "true"

const isMenuCollapsed = ref<boolean>(readStoredCollapsed())
const isMobileMenuOpen = ref(false)
const menuTriggerElement = ref<HTMLElement | null>(null)
const focusReturnElement = ref<HTMLElement | null>(null)

watch(isMenuCollapsed, (isCollapsed) => {
  localStorage.setItem(MENU_COLLAPSED_STORAGE_KEY, String(isCollapsed))
})

const isFocusableElement = (element: Element | null): element is HTMLElement =>
  element instanceof HTMLElement && element !== document.body

const restoreFocus = (): void => {
  const returnElement = focusReturnElement.value ?? menuTriggerElement.value
  focusReturnElement.value = null
  if (returnElement !== null && document.contains(returnElement)) returnElement.focus()
}

export const useMainSidebar = () => {
  const toggleMenuCollapsed = (): void => {
    isMenuCollapsed.value = !isMenuCollapsed.value
  }

  const registerMenuTrigger = (element: HTMLElement | null): void => {
    menuTriggerElement.value = element
  }

  const openMobileMenu = (): void => {
    const activeElement = document.activeElement
    focusReturnElement.value = isFocusableElement(activeElement) ? activeElement : menuTriggerElement.value
    isMobileMenuOpen.value = true
  }

  const closeMobileMenu = (): void => {
    if (!isMobileMenuOpen.value) return
    isMobileMenuOpen.value = false
    restoreFocus()
  }

  const resetMobileMenu = (): void => {
    isMobileMenuOpen.value = false
    focusReturnElement.value = null
  }

  return {
    isMenuCollapsed,
    isMobileMenuOpen,
    registerMenuTrigger,
    toggleMenuCollapsed,
    openMobileMenu,
    closeMobileMenu,
    resetMobileMenu
  }
}

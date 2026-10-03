import { onBeforeUnmount, onMounted } from "vue"
import { useCopilot } from "./useCopilot"

const TEXT_ENTRY_TAG_NAMES = new Set(["INPUT", "TEXTAREA", "SELECT"])

const isTextEntryTarget = (eventTarget: EventTarget | null): boolean => {
  if (!(eventTarget instanceof HTMLElement)) {
    return false
  }

  if (eventTarget.isContentEditable) {
    return true
  }

  return TEXT_ENTRY_TAG_NAMES.has(eventTarget.tagName)
}

export const useCopilotShortcuts = (): void => {
  const { startNewChat } = useCopilot()

  const handleKeydown = (event: KeyboardEvent): void => {
    const isNewChatShortcut = (event.metaKey || event.ctrlKey) && event.shiftKey && event.key.toLowerCase() === "o"

    if (!isNewChatShortcut || isTextEntryTarget(event.target)) {
      return
    }

    event.preventDefault()
    startNewChat()
  }

  onMounted(() => {
    window.addEventListener("keydown", handleKeydown)
  })

  onBeforeUnmount(() => {
    window.removeEventListener("keydown", handleKeydown)
  })
}

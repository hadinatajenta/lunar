import { ref } from "vue"

const toastMessage = ref("")
const isToastVisible = ref(false)
let toastTimer: ReturnType<typeof setTimeout> | null = null

export function useToast() {
  const showToast = (message: string, duration = 2400) => {
    toastMessage.value = message
    isToastVisible.value = true
    if (toastTimer) {
      clearTimeout(toastTimer)
    }
    toastTimer = setTimeout(() => {
      isToastVisible.value = false
    }, duration)
  }

  const hideToast = () => {
    isToastVisible.value = false
    if (toastTimer) {
      clearTimeout(toastTimer)
    }
  }

  return {
    toastMessage,
    isToastVisible,
    showToast,
    hideToast,
  }
}

import { onUnmounted, ref, watch, type Ref } from "vue"

export const SEARCH_DEBOUNCE_MS = 200

export function useDebouncedValue(source: Ref<string>, delayMs: number): Ref<string> {
  const debouncedValue = ref(source.value)
  let pendingTimer: ReturnType<typeof setTimeout> | null = null

  const cancelPendingUpdate = () => {
    if (pendingTimer !== null) {
      clearTimeout(pendingTimer)
      pendingTimer = null
    }
  }

  watch(source, (nextValue) => {
    cancelPendingUpdate()
    pendingTimer = setTimeout(() => {
      debouncedValue.value = nextValue
      pendingTimer = null
    }, delayMs)
  })

  onUnmounted(cancelPendingUpdate)

  return debouncedValue
}

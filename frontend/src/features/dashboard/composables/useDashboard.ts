import { computed, ref } from "vue"
import { ApiError } from "../../../lib/http"
import { fetchDashboardSummary } from "../api/dashboard-api"
import type { DashboardSummary } from "../types"

const summary = ref<DashboardSummary | null>(null)
const isLoading = ref(false)
const isSyncing = ref(false)
const errorMessage = ref<string | null>(null)

const lastSyncedAt = computed(() => {
  const syncedAt = summary.value?.synced_at
  if (!syncedAt) {
    return null
  }
  const syncedDate = new Date(syncedAt)
  if (Number.isNaN(syncedDate.getTime())) {
    return null
  }
  return syncedDate.toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  })
})

const resolveLoadErrorMessage = (err: unknown): string => {
  if (err instanceof ApiError && err.statusCode === 401) {
    return "Your session has expired. Sign in again to load the dashboard."
  }
  if (err instanceof Error && err.message) {
    return err.message
  }
  return "Failed to load the dashboard summary"
}

const loadSummary = async (): Promise<boolean> => {
  isLoading.value = true
  errorMessage.value = null
  try {
    summary.value = await fetchDashboardSummary()
    return true
  } catch (err: unknown) {
    summary.value = null
    errorMessage.value = resolveLoadErrorMessage(err)
    return false
  } finally {
    isLoading.value = false
  }
}

const syncWorkspace = async (): Promise<boolean> => {
  isSyncing.value = true
  try {
    return await loadSummary()
  } finally {
    isSyncing.value = false
  }
}

export const useDashboard = () => {
  return {
    summary,
    isLoading,
    isSyncing,
    errorMessage,
    lastSyncedAt,
    loadSummary,
    syncWorkspace
  }
}

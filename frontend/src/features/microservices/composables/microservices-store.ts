import { computed, ref } from "vue"
import { ApiError, getStoredToken } from "../../../lib/http"
import type {
  HelperConnection,
  HelperConnectionStatus,
  RepositoryDetail,
  RepositorySummary,
  WorkspaceSettings
} from "../types"
import { HelperConnectionError } from "../api/helper-api"

export const HELPER_TOKEN_MISSING_MESSAGE =
  "The local helper is not connected, so its token cannot be read. Start the helper on this machine, then reload. If it is already running, enter its token in Settings."
export const REPOSITORY_ROOT_MISSING_MESSAGE =
  "No repository root folder is configured. Open Settings and enter the folder that holds your repositories."

const HELPER_TOKEN_STORAGE_KEY = "lunar_helper_token"

function readStoredHelperToken(): string {
  try {
    return sessionStorage.getItem(HELPER_TOKEN_STORAGE_KEY) ?? ""
  } catch {
    return ""
  }
}

function writeStoredHelperToken(token: string): void {
  try {
    sessionStorage.setItem(HELPER_TOKEN_STORAGE_KEY, token)
  } catch {
    return
  }
}

function removeStoredHelperToken(): void {
  try {
    sessionStorage.removeItem(HELPER_TOKEN_STORAGE_KEY)
  } catch {
    return
  }
}

export const workspace = ref<WorkspaceSettings | null>(null)
export const repositories = ref<RepositorySummary[]>([])
export const services = ref<RepositoryDetail[]>([])
export const selectedNames = ref<string[]>([])
export const isLoading = ref(false)
export const isSaving = ref(false)
export const error = ref<string | null>(null)
export const helperStatus = ref<HelperConnectionStatus>("unknown")
export const helperVersion = ref("")
const helperToken = ref(readStoredHelperToken())
export const hasHelperToken = computed(() => helperToken.value.length > 0)

let loadedSessionToken: string | null = null

function clearMicroservicesState(): void {
  workspace.value = null
  repositories.value = []
  services.value = []
  selectedNames.value = []
  isLoading.value = false
  isSaving.value = false
  error.value = null
  helperStatus.value = "unknown"
  helperVersion.value = ""
}

export function storeHelperToken(token: string): void {
  helperToken.value = token
  writeStoredHelperToken(token)
}

export function discardHelperToken(): void {
  helperToken.value = ""
  removeStoredHelperToken()
}

export function resetMicroservicesStore(): void {
  clearMicroservicesState()
  discardHelperToken()
  loadedSessionToken = getStoredToken()
}

export function syncSessionScope(): void {
  const currentSessionToken = getStoredToken()
  if (currentSessionToken === loadedSessionToken) {
    return
  }
  clearMicroservicesState()
  if (loadedSessionToken !== null) {
    discardHelperToken()
  }
  loadedSessionToken = currentSessionToken
}

export function readSelectedNames(settings: WorkspaceSettings): string[] | null {
  const selected: unknown = settings.selected
  if (!Array.isArray(selected)) {
    return null
  }
  return selected.filter((name: unknown): name is string => typeof name === "string")
}

export function resolveErrorMessage(thrownError: unknown, fallbackMessage: string): string {
  if (thrownError instanceof Error && thrownError.message.length > 0) {
    return thrownError.message
  }
  return fallbackMessage
}

export function registerFetchError(thrownError: unknown, fallbackMessage: string): void {
  if (thrownError instanceof HelperConnectionError) {
    helperStatus.value = "offline"
    helperVersion.value = ""
  }
  error.value = resolveErrorMessage(thrownError, fallbackMessage)
}

export function resolveHelperConnection(): HelperConnection | null {
  if (workspace.value === null || helperToken.value.length === 0) {
    return null
  }
  return { helperUrl: workspace.value.helper_url, helperToken: helperToken.value }
}

export function requireHelperConnection(): HelperConnection {
  const connection = resolveHelperConnection()
  if (connection === null) {
    throw new ApiError(HELPER_TOKEN_MISSING_MESSAGE, 0)
  }
  return connection
}

export function hasConfiguredRoot(settings: WorkspaceSettings): boolean {
  return settings.root_path.trim().length > 0
}

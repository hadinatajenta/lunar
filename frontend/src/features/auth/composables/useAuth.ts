import { computed, ref } from "vue"
import { clearStoredToken, getStoredToken, setStoredToken } from "@/lib/http"
import { formatErrorMessage } from "@/lib/utils"
import { getCurrentUser, loginUser, logoutUser } from "../api/auth-api"
import type { LoginRequest, UserProfile } from "../types"

const userState = ref<UserProfile | null>(null)
const tokenState = ref<string | null>(getStoredToken())
const isLoadingState = ref<boolean>(false)
const errorMessageState = ref<string | null>(null)

export function useAuth() {
  const isAuthenticated = computed(() => Boolean(tokenState.value))

  function clearError(): void {
    errorMessageState.value = null
  }

  async function login(payload: LoginRequest): Promise<boolean> {
    isLoadingState.value = true
    errorMessageState.value = null

    try {
      const response = await loginUser(payload)
      tokenState.value = response.token
      userState.value = response.user
      setStoredToken(response.token)
      return true
    } catch (error) {
      errorMessageState.value = formatErrorMessage(error, "Failed to sign in. Please verify your credentials.")
      return false
    } finally {
      isLoadingState.value = false
    }
  }

  async function fetchUser(): Promise<UserProfile | null> {
    if (!tokenState.value) {
      userState.value = null
      return null
    }

    isLoadingState.value = true
    try {
      const profile = await getCurrentUser()
      userState.value = profile
      return profile
    } catch {
      userState.value = null
      tokenState.value = null
      clearStoredToken()
      return null
    } finally {
      isLoadingState.value = false
    }
  }

  async function logout(): Promise<void> {
    isLoadingState.value = true
    try {
      await logoutUser()
    } finally {
      userState.value = null
      tokenState.value = null
      clearStoredToken()
      isLoadingState.value = false
    }
  }

  return {
    user: computed(() => userState.value),
    token: computed(() => tokenState.value),
    isLoading: computed(() => isLoadingState.value),
    errorMessage: computed(() => errorMessageState.value),
    isAuthenticated,
    login,
    logout,
    fetchUser,
    clearError
  }
}

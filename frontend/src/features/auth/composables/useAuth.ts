import { computed, ref } from "vue"
import { ApiError, clearStoredToken, getStoredToken, setStoredToken } from "@/lib/http"
import { formatErrorMessage } from "@/lib/utils"
import {
  getCurrentUser,
  loginUser,
  logoutUser,
  registerWithJira as apiRegisterWithJira,
  verifyJiraPat as apiVerifyJiraPat
} from "../api/auth-api"
import type { LoginRequest, UserProfile, VerifiedJiraProfile } from "../types"

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

  async function verifyJiraPat(pat: string): Promise<VerifiedJiraProfile | null> {
    isLoadingState.value = true
    errorMessageState.value = null

    try {
      const profile = await apiVerifyJiraPat(pat)
      return profile
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.statusCode === 401) {
          errorMessageState.value = "Invalid or expired Jira Personal Access Token."
          return null
        }
        if (error.statusCode === 409) {
          errorMessageState.value = "An account with this Jira identity already exists. Please sign in."
          return null
        }
        if (error.statusCode === 502) {
          errorMessageState.value = "Unable to reach BRI Jira. Please verify your VPN or network connection."
          return null
        }
      }
      errorMessageState.value = formatErrorMessage(error, "Failed to verify Jira token.")
      return null
    } finally {
      isLoadingState.value = false
    }
  }

  async function registerWithJira(pat: string, password: string): Promise<boolean> {
    isLoadingState.value = true
    errorMessageState.value = null

    try {
      const response = await apiRegisterWithJira(pat, password)
      tokenState.value = response.token
      userState.value = response.user
      setStoredToken(response.token)
      return true
    } catch (error) {
      if (error instanceof ApiError) {
        if (error.statusCode === 409) {
          errorMessageState.value = "An account with this Jira identity already exists. Please sign in."
          return false
        }
        if (error.statusCode === 401) {
          errorMessageState.value = "Invalid or expired Jira Personal Access Token."
          return false
        }
        if (error.statusCode === 502) {
          errorMessageState.value = "Unable to reach BRI Jira. Please verify your VPN or network connection."
          return false
        }
      }
      errorMessageState.value = formatErrorMessage(error, "Failed to create account.")
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
    userState.value = null
    tokenState.value = null
    clearStoredToken()
    isLoadingState.value = true
    try {
      await logoutUser()
    } catch {
    } finally {
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
    verifyJiraPat,
    registerWithJira,
    clearError
  }
}

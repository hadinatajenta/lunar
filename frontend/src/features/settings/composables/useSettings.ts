import { ref } from "vue"
import { getSystemConfig, getUserSecrets, saveUserSecrets } from "../api/settings-api"
import type { RedactedSecrets, SystemConfig, UserSecretsInput } from "../types"

const secrets = ref<RedactedSecrets | null>(null)
const config = ref<SystemConfig | null>(null)
const isLoading = ref(false)
const isSaving = ref(false)
const saveSuccess = ref(false)
const errorMessage = ref<string | null>(null)

export const useSettings = () => {
  const fetchSettings = async () => {
    isLoading.value = true
    errorMessage.value = null
    try {
      const [fetchedConfig, fetchedSecrets] = await Promise.all([
        getSystemConfig(),
        getUserSecrets()
      ])
      config.value = fetchedConfig
      secrets.value = fetchedSecrets
    } catch (err: unknown) {
      errorMessage.value = err instanceof Error ? err.message : "Failed to load settings"
    } finally {
      isLoading.value = false
    }
  }

  const save = async (input: UserSecretsInput) => {
    isSaving.value = true
    saveSuccess.value = false
    errorMessage.value = null
    try {
      const updated = await saveUserSecrets(input)
      secrets.value = updated
      saveSuccess.value = true
      setTimeout(() => {
        saveSuccess.value = false
      }, 3000)
      return true
    } catch (err: unknown) {
      errorMessage.value = err instanceof Error ? err.message : "Failed to save settings"
      return false
    } finally {
      isSaving.value = false
    }
  }

  return {
    secrets,
    config,
    isLoading,
    isSaving,
    saveSuccess,
    errorMessage,
    fetchSettings,
    save
  }
}
